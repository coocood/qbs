package qbs

import (
	"database/sql"
	"testing"
	"time"
)

import (
	"fmt"
//	_ "github.com/bmizerany/pq"
	"github.com/coocood/assrt"
	_ "github.com/ziutek/mymysql/godrv"
	"errors"
)

var toRun = []dialectInfo{
	// allDialectInfos[0],
	allDialectInfos[1],
}

const (
	dbName         = "qbs_test"
	dbUser         = "qbs_test"
	mysqlDriver    = "mymysql"
	mysqlDrvformat = "%v/%v/"
	pgDriver       = "postgres"
	pgDrvFormat    = "user=%v dbname=%v sslmode=disable"
)

var allDialectInfos = []dialectInfo{
	dialectInfo{
		NewPostgres(),
		openPgDb,
	},
	dialectInfo{
		NewMysql(),
		openMysqlDb,
	},
}

type dialectInfo struct {
	dialect    Dialect
	openDbFunc func() (*sql.DB, error)
}

func setupDb(assert *assrt.Assert, info dialectInfo) (*Migration, *Qbs) {
	db1, err := info.openDbFunc()
	assert.MustNil(err)
	mg := NewMigration(db1, dbName, info.dialect)
	db2, err := info.openDbFunc()
	assert.MustNil(err)
	q := New(db2, info.dialect)
	q.Log = true
	return mg, q
}

func openPgDb() (*sql.DB, error) {
	return sql.Open(pgDriver, fmt.Sprintf(pgDrvFormat, dbUser, dbName))
}

func openMysqlDb() (*sql.DB, error) {
	return sql.Open(mysqlDriver, fmt.Sprintf(mysqlDrvformat, dbName, dbUser))
}

func TestTransaction(t *testing.T) {
	for _, info := range toRun {
		DoTestTransaction(assrt.NewAssert(t), info)
	}
}

func DoTestTransaction(assert *assrt.Assert, info dialectInfo) {
	mg, q := setupDb(assert, info)
	type txModel struct {
		Id Id
		A  string
	}
	table := txModel{
		A: "A",
	}
	mg.dropTableIfExists(&table)
	mg.CreateTableIfNotExists(&table)
	q.Begin()
	assert.NotNil(q.Tx)
	_, err := q.Save(&table)
	assert.Nil(err)
	err = q.Rollback()
	assert.Nil(err)
	out := new(txModel)
	err = q.Find(out)
	assert.Nil(err)
	assert.Zero(out.Id)
	q.Begin()
	table.Id = 0
	_, err = q.Save(&table)
	assert.Nil(err)
	err = q.Commit()
	assert.Nil(err)
	err = q.Find(out)
	assert.Nil(err)
	assert.Equal("A", out.A)
}

func TestSaveAndDelete(t *testing.T) {
	for _, info := range toRun {
		DoTestSaveAndDelete(assrt.NewAssert(t), info)
	}
}

func DoTestSaveAndDelete(assert *assrt.Assert, info dialectInfo) {
	x := time.Now()
	assert.MustZero(x.Sub(x.UTC()))
	now := time.Now()
	mg, q := setupDb(assert, info)
	type saveModel struct {
		Id      Id
		A       string
		B       int
		Updated time.Time
		Created time.Time
	}
	model1 := saveModel{
		A: "banana",
		B: 5,
	}
	model2 := saveModel{
		A: "orange",
		B: 4,
	}

	mg.dropTableIfExists(&model1)
	mg.CreateTableIfNotExists(&model1)
	affected, err := q.Save(&model1)
	assert.MustNil(err)
	assert.Equal(1, affected)
	assert.True(model1.Created.Sub(now) > 0)
	assert.True(model1.Updated.Sub(now) > 0)

	// make sure created/updated values match the db
	var model1r []*saveModel
	err = q.Where("id = ?", model1.Id).FindAll(&model1r)
	assert.MustNil(err)
	assert.MustOneLen(model1r)
	assert.Equal(model1.Created.Unix(), model1r[0].Created.Unix())
	assert.Equal(model1.Updated.Unix(), model1r[0].Updated.Unix())

	oldCreate := model1.Created
	oldUpdate := model1.Updated
	model1.A = "grape"
	model1.B = 9

	time.Sleep(time.Second * 1) // sleep for 1 sec

	affected, err = q.Save(&model1)
	assert.MustNil(err)
	assert.MustEqual(1, affected)
	assert.True(model1.Created.Equal(oldCreate))
	assert.True(model1.Updated.Sub(oldUpdate) > 0)

	// make sure created/updated values match the db
	var model1r2 []*saveModel
	err = q.Where("id = ?", model1.Id).FindAll(&model1r2)
	assert.MustNil(err)
	assert.MustOneLen(model1r2)
	assert.True(model1r2[0].Updated.Sub(model1r2[0].Created) >= 1)
	assert.Equal(model1.Created.Unix(), model1r2[0].Created.Unix())
	assert.Equal(model1.Updated.Unix(), model1r2[0].Updated.Unix())

	affected, err = q.Save(&model2)
	assert.MustNil(err)
	assert.Equal(1, affected)

	affected, err = q.Delete(&model2)
	assert.MustNil(err)
	assert.Equal(1, affected)
}

func TestForeignKey(t *testing.T) {
	for _, info := range toRun {
		DoTestForeignKey(assrt.NewAssert(t), info)
	}
}

func DoTestForeignKey(assert *assrt.Assert, info dialectInfo) {
	mg, q := setupDb(assert, info)
	type user struct {
		Id   Id
		Name string
	}
	type post struct {
		Id       Id
		Title    string
		AuthorId int64
		Author   *user
	}
	aUser := &user{
		Name: "john",
	}
	aPost := &post{
		Title: "A Title",
	}
	mg.dropTableIfExists(aPost)
	mg.dropTableIfExists(aUser)
	mg.CreateTableIfNotExists(aUser)
	mg.CreateTableIfNotExists(aPost)

	uid, err := q.Save(aUser)
	assert.Nil(err)
	aPost.AuthorId = int64(uid)
	affected, err := q.Save(aPost)
	assert.Equal(1, affected)
	pst := new(post)
	pst.Id = aPost.Id
	err = q.Find(pst)
	assert.MustNil(err)
	assert.MustNotNil(pst)
	assert.Equal(uid, pst.Id)
	assert.Equal("john", pst.Author.Name)
}

func TestFind(t *testing.T) {
	for _, info := range toRun {
		DoTestFind(assrt.NewAssert(t), info)
	}
}

func DoTestFind(assert *assrt.Assert, info dialectInfo) {
	mg, q := setupDb(assert, info)
	now := time.Now()

	type types struct {
		Id    Id
		Str   string
		Intgr int64
		Flt   float64
		Bytes []byte
		Time  time.Time
	}
	modelData := &types{
		Str:   "string!",
		Intgr: -1,
		Flt:   3.8,
		Bytes: []byte("bytes!"),
		Time:  now,
	}

	mg.dropTableIfExists(modelData)
	mg.CreateTableIfNotExists(modelData)

	out := new(types)
	condition := NewCondition("str = ?", "string!").And("intgr = ?", -1)
	err := q.Condition(condition).Find(out)
	assert.Nil(err)
	assert.Zero(out.Id)

	affected, err := q.Save(modelData)
	assert.Nil(err)
	assert.Equal(1, affected)

	err = q.Condition(condition).Find(out)
	assert.Nil(err)
	assert.Equal(1, out.Id)
	assert.Equal("string!", out.Str)
	assert.Equal(-1, out.Intgr)
	assert.Equal(3.8, out.Flt)
	assert.Equal([]byte("bytes!"), out.Bytes)
	diff := now.Sub(out.Time)
	assert.True(diff < time.Second && diff > -time.Second)

	modelData.Id = 5
	modelData.Str = "New row"
	_, err = q.Save(modelData)
	assert.Nil(err)

	out = new(types)
	condition = NewCondition("str = ?", "New row").And("flt = ?", 3.8)
	err = q.Condition(condition).Find(out)
	assert.Nil(err)
	assert.Equal(5, out.Id)

	allOut := []*types{}
	err = q.Where("intgr = ?", -1).FindAll(&allOut)
	assert.Nil(err)
	assert.Equal(2, len(allOut))
}

func TestCreateTable(t *testing.T) {
	for _, info := range toRun {
		DoTestCreateTable(assrt.NewAssert(t), info)
	}
}

type AddColumn struct {
	Prim   Id
	First  string `sql:"size:64,notnull"`
	Last   string `sql:"size:128,default:'defaultValue'"`
	Amount int
}

func (table *AddColumn) Indexes(indexes *Indexes) {
	indexes.AddUnique("first", "last")
}

func DoTestCreateTable(assert *assrt.Assert, info dialectInfo) {
	assert.Logf("Dialect %T\n", info.dialect)
	mg, _ := setupDb(assert, info)
	{
		type AddColumn struct {
			Prim Id
		}
		table := &AddColumn{}
		mg.dropTableIfExists(table)
		mg.CreateTableIfNotExists(table)
		columns := mg.Dialect.ColumnsInTable(mg, table)
		assert.OneLen(columns)
		assert.True(columns["prim"])
	}
	table := &AddColumn{}
	mg.CreateTableIfNotExists(table)
	columns := mg.Dialect.ColumnsInTable(mg, table)
	assert.Equal(4, len(columns))
}

type basic struct {
	Id    Id
	Name  string `sql:"size:64"`
	State int64
}

func TestUpdate(t *testing.T) {
	for _, info := range toRun {
		DoTestUpdate(assrt.NewAssert(t), info)
	}
}

func DoTestUpdate(assert *assrt.Assert, info dialectInfo) {
	mg, q := setupDb(assert, info)
	mg.dropTableIfExists(&basic{})
	mg.CreateTableIfNotExists(&basic{})
	_, err := q.Save(&basic{Name: "a", State: 1})
	_, err = q.Save(&basic{Name: "b", State: 1})
	_, err = q.Save(&basic{Name: "c", State: 0})
	assert.MustNil(err)
	{
		// define a temporary struct in a block to update partial columns of a table
		// as the type is in a block, so it will not conflict with other types with the same name in the same method
		type basic struct {
			Name string
		}
		affected, err := q.Where("state = ?", 1).Update(&basic{Name: "d"})
		assert.MustNil(err)
		assert.Equal(2, affected)

		var datas []*basic
		q.Where("state = ?", 1).FindAll(&datas)
		assert.MustEqual(2, len(datas))
		assert.Equal("d", datas[0].Name)
		assert.Equal("d", datas[1].Name)
	}

	// if choose basic table type to update, all zero value in the struct will be updated too.
	// this may be cause problems, so define a temporary struct to update table is the recommended way.
	affected, err := q.Where("state = ?", 1).Update(&basic{Name: "e"})
	assert.MustNil(err)
	assert.Equal(2, affected)
	var datas []*basic
	q.Where("state = ?", 1).FindAll(&datas)
	assert.MustEqual(0, len(datas))
}

func TestValidation(t *testing.T){
	for _, info := range toRun {
		DoTestValidation(assrt.NewAssert(t), info)
	}
}
//
type ValidatorTable struct {
	Id Id
	Name string
}

func (v *ValidatorTable) Validate(q *Qbs) error{
	if q.ContainsValue(v,"name", v.Name) {
		return errors.New("name already taken")
	}
	return nil
}

func DoTestValidation(assert *assrt.Assert, info dialectInfo){
	mg, q := setupDb(assert, info)
	valid := new(ValidatorTable)
	mg.dropTableIfExists(valid)
	mg.CreateTableIfNotExists(valid)
	valid.Name = "ok"
	q.Save(valid)
	valid.Id = 0
	_, err := q.Save(valid)
	assert.MustNotNil(err)
	assert.Equal("name already taken", err.Error())
}
