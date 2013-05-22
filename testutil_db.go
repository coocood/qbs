package qbs

import (
	"database/sql"
	"errors"
	"github.com/coocood/assrt"
	"testing"
	"time"
)

var testDbName = "qbs_test"
var testDbUser = "qbs_test"

type addColumn struct {
	Prim   int64  `qbs:"pk"`
	First  string `qbs:"size:64,notnull"`
	Last   string `qbs:"size:128,default:'defaultValue'"`
	Amount int
}

func (table *addColumn) Indexes(indexes *Indexes) {
	indexes.AddUnique("first", "last")
}

func doTestTransaction(t *testing.T, mg *Migration, q *Qbs) {
	defer closeMigrationAndQbs(mg, q)
	assert := assrt.NewAssert(t)
	type txModel struct {
		Id int64
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
	assert.Equal(sql.ErrNoRows, err)
	q.Begin()
	table.Id = 0
	_, err = q.Save(&table)
	assert.Nil(err)
	err = q.Commit()
	assert.Nil(err)
	out.Id = table.Id
	err = q.Find(out)
	assert.Nil(err)
	assert.Equal("A", out.A)
}

func doTestSaveAndDelete(t *testing.T, mg *Migration, q *Qbs) {
	defer closeMigrationAndQbs(mg, q)
	assert := assrt.NewAssert(t)
	x := time.Now()
	assert.Zero(x.Sub(x.UTC()))
	now := time.Now()
	type saveModel struct {
		Id      int64
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
	err = q.WhereEqual("id", model1.Id).FindAll(&model1r)

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

func doTestSaveAgain(t *testing.T, mg *Migration, q *Qbs) {
	defer closeMigrationAndQbs(mg, q)
	assert := assrt.NewAssert(t)
	b := new(basic)
	mg.dropTableIfExists(b)
	mg.CreateTableIfNotExists(b)
	b.Name = "a"
	b.State = 2
	affected, err := q.Save(b)
	assert.Nil(err)
	assert.Equal(1, affected)
	affected, err = q.Save(b)
	assert.Nil(err)
	if _, ok := q.Dialect.(*mysql); ok {
		assert.Equal(0, affected)
	}else{
		assert.Equal(1, affected)
	}
}

func doTestForeignKey(t *testing.T, mg *Migration, q *Qbs) {
	defer closeMigrationAndQbs(mg, q)
	assert := assrt.NewAssert(t)
	type user struct {
		Id   int64
		Name string
	}
	type post struct {
		Id       int64
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

	affected, err := q.Save(aUser)
	assert.Nil(err)
	aPost.AuthorId = int64(aUser.Id)
	affected, err = q.Save(aPost)
	assert.Equal(1, affected)
	pst := new(post)
	pst.Id = aPost.Id
	err = q.Find(pst)
	assert.MustNil(err)
	assert.Equal(aPost.Id, pst.Id)
	assert.Equal("john", pst.Author.Name)

	pst.Author = nil
	err = q.OmitFields("Author").Find(pst)
	assert.MustNil(err)
	assert.MustNil(pst.Author)

	err = q.OmitJoin().Find(pst)
	assert.MustNil(err)
	assert.MustNil(pst.Author)

	var psts []*post
	err = q.FindAll(&psts)
	assert.MustNil(err)
	assert.OneLen(psts)
	assert.Equal("john", psts[0].Author.Name)
}

func doTestFind(t *testing.T) {
	assert := assrt.NewAssert(t)
	now := time.Now()
	type types struct {
		Id    int64
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
	WithMigration(func(mg *Migration)error{
		mg.dropTableIfExists(modelData)
		mg.CreateTableIfNotExists(modelData)
		return nil
	})
	WithQbs(func(q *Qbs)error{
		out := new(types)
		condition := NewCondition("str = ?", "string!").And("intgr = ?", -1)
		err := q.Condition(condition).Find(out)
		assert.Equal(sql.ErrNoRows, err)

		affected, err := q.Save(modelData)
		assert.Nil(err)
		assert.Equal(1, affected)
		out.Id = modelData.Id
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

		out = new(types)
		out.Id = 100
		err = q.Find(out)
		assert.NotNil(err)

		allOut := []*types{}
		err = q.WhereEqual("intgr", -1).FindAll(&allOut)
		assert.Nil(err)
		assert.Equal(2, len(allOut))
		return nil
	})
}

func doTestCreateTable(t *testing.T, mg *Migration) {
	defer mg.Close()
	assert := assrt.NewAssert(t)
	{
		type AddColumn struct {
			Prim int64 `qbs:"pk"`
		}
		table := &AddColumn{}
		mg.dropTableIfExists(table)
		mg.CreateTableIfNotExists(table)
		columns := mg.Dialect.columnsInTable(mg, table)
		assert.OneLen(columns)
		assert.True(columns["prim"])
	}
	table := &addColumn{}
	mg.CreateTableIfNotExists(table)
	assert.True(mg.Dialect.indexExists(mg, "add_column", "add_column_first_last"))
	columns := mg.Dialect.columnsInTable(mg, table)
	assert.Equal(4, len(columns))
}

type basic struct {
	Id    int64
	Name  string `qbs:"size:64"`
	State int64
}

func doTestUpdate(t *testing.T, mg *Migration, q *Qbs) {
	defer closeMigrationAndQbs(mg, q)
	assert := assrt.NewAssert(t)
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
		affected, err := q.WhereEqual("state", 1).Update(&basic{Name: "d"})
		assert.MustNil(err)
		assert.Equal(2, affected)

		var datas []*basic
		q.WhereEqual("state", 1).FindAll(&datas)
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
	q.WhereEqual("state", 1).FindAll(&datas)
	assert.MustEqual(0, len(datas))
}

type validatorTable struct {
	Id   int64
	Name string
}

func (v *validatorTable) Validate(q *Qbs) error {
	if q.ContainsValue(v, "name", v.Name) {
		return errors.New("name already taken")
	}
	return nil
}

func doTestValidation(t *testing.T, mg *Migration, q *Qbs) {
	defer closeMigrationAndQbs(mg, q)
	assert := assrt.NewAssert(t)
	valid := new(validatorTable)
	mg.dropTableIfExists(valid)
	mg.CreateTableIfNotExists(valid)
	valid.Name = "ok"
	q.Save(valid)
	valid.Id = 0
	_, err := q.Save(valid)
	assert.MustNotNil(err)
	assert.Equal("name already taken", err.Error())
}

func doTestBoolType(t *testing.T, mg *Migration, q *Qbs) {
	defer closeMigrationAndQbs(mg, q)
	assert := assrt.NewAssert(t)
	type BoolType struct {
		Id     int64
		Active bool
	}
	bt := new(BoolType)
	mg.dropTableIfExists(bt)
	mg.CreateTableIfNotExists(bt)
	bt.Active = true
	q.Save(bt)
	bt.Active = false
	q.WhereEqual("active", true).Find(bt)
	assert.True(bt.Active)
}

func doTestStringPk(t *testing.T, mg *Migration, q *Qbs) {
	defer closeMigrationAndQbs(mg, q)
	assert := assrt.NewAssert(t)
	type StringPk struct {
		Tag   string `qbs:"pk,size:16"`
		Count int32
	}
	spk := new(StringPk)
	spk.Tag = "health"
	spk.Count = 10
	mg.dropTableIfExists(spk)
	mg.CreateTableIfNotExists(spk)
	affected, _ := q.Save(spk)
	assert.Equal(1, affected)
	spk.Count = 0
	q.Find(spk)
	assert.Equal(10, spk.Count)
}

func doTestCount(t *testing.T, mg *Migration, q *Qbs) {
	defer closeMigrationAndQbs(mg, q)
	assert := assrt.NewAssert(t)
	basic := new(basic)
	mg.dropTableIfExists(basic)
	mg.CreateTableIfNotExists(basic)
	basic.Name = "name"
	basic.State = 1
	q.Save(basic)
	for i := 0; i < 5; i++ {
		basic.Id = 0
		basic.State = 2
		q.Save(basic)
	}
	count1 := q.Count("basic")
	assert.Equal(6, count1)
	count2 := q.WhereEqual("state", 2).Count(basic)
	assert.Equal(5, count2)
}

func doTestQueryMap(t *testing.T, mg *Migration, q *Qbs) {
	defer closeMigrationAndQbs(mg, q)
	assert := assrt.NewAssert(t)
	type types struct {
		Id      int64
		Name    string `qbs:"size:64"`
		Created time.Time
	}
	tp := new(types)
	mg.dropTableIfExists(tp)
	mg.CreateTableIfNotExists(tp)
	result, err := q.QueryMap("SELECT * FROM types")
	assert.Nil(result)
	assert.Equal(sql.ErrNoRows, err)
	for i := 0; i < 3; i++ {
		tp.Id = 0
		tp.Name = "abc"
		q.Save(tp)
	}
	result, err = q.QueryMap("SELECT * FROM types")
	assert.NotNil(result)
	assert.Equal(1, result["id"])
	assert.Equal("abc", result["name"])
	if _, sql3 := q.Dialect.(*sqlite3); !sql3 {
		_, ok := result["created"].(time.Time)
		assert.True(ok)
	}else{
		_, ok := result["created"].(string)
		assert.True(ok)
	}
	results, err := q.QueryMapSlice("SELECT * FROM types")
	assert.Equal(3, len(results))
}

func doTestBulkInsert(t *testing.T) {
	assert := assrt.NewAssert(t)
	WithMigration(func(mg *Migration)error{
		b := new(basic)
		mg.dropTableIfExists(b)
		mg.Log = true
		mg.CreateTableIfNotExists(b)
		return nil
	})
	WithQbs(func(q *Qbs)error{
		var bulk []*basic
		for i:=0; i < 10; i++{
			b := new(basic)
			b.Name = "basic"
			b.State = int64(i)
			bulk = append(bulk, b)
		}
		err := q.BulkInsert(bulk)
		assert.Nil(err)
		for i:=0; i<10; i++{
			assert.Equal(i+1, bulk[i].Id)
		}
		return nil
	})
}

func closeMigrationAndQbs(mg *Migration, q *Qbs) {
	mg.Close()
	q.Close()
}
