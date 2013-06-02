package qbs

import (
	"testing"
)

type dialectSyntax struct {
	dialect                         Dialect
	createTableWithoutPkIfExistsSql string
	createTableWithPkSql            string
	insertSql                       string
	updateSql                       string
	deleteSql                       string
	selectionSql                    string
	querySql                        string
	dropTableIfExistsSql            string
	addColumnSql                    string
	createUniqueIndexSql            string
	createIndexSql                  string
}

type sqlGenModel struct {
	Prim   int64 `qbs:"pk"`
	First  string
	Last   string
	Amount int
}

var sqlGenSampleData = &sqlGenModel{3, "FirstName", "LastName", 6}

func doTestAddColumSQL(t *testing.T, info dialectSyntax) {
	assert := NewAssert(t)
	sql := info.dialect.addColumnSql("a", "c", "", 100)
	assert.Equal(info.addColumnSql, sql)
}

func doTestCreateTableSQL(t *testing.T, info dialectSyntax) {
	assert := NewAssert(t)
	type withoutPk struct {
		First  string
		Last   string
		Amount int
	}
	table := &withoutPk{"a", "b", 5}
	model := structPtrToModel(table, true, nil)
	sql := info.dialect.createTableSql(model, true)
	assert.Equal(info.createTableWithoutPkIfExistsSql, sql)
	type withPk struct {
		Primary int64 `qbs:"pk"`
		First   string
		Last    string
		Amount  int
	}
	table2 := &withPk{First: "a", Last: "b", Amount: 5}
	model = structPtrToModel(table2, true, nil)
	sql = info.dialect.createTableSql(model, false)
	assert.Equal(info.createTableWithPkSql, sql)
}

func doTestCreateIndexSQL(t *testing.T, info dialectSyntax) {
	assert := NewAssert(t)
	sql := info.dialect.createIndexSql("iname", "itable", true, "a", "b", "c")
	assert.Equal(info.createUniqueIndexSql, sql)
	sql = info.dialect.createIndexSql("iname2", "itable2", false, "d", "e")
	assert.Equal(info.createIndexSql, sql)
}

func doTestInsertSQL(t *testing.T, info dialectSyntax) {
	assert := NewAssert(t)
	model := structPtrToModel(sqlGenSampleData, true, nil)
	criteria := &criteria{model: model}
	criteria.mergePkCondition(info.dialect)
	sql, _ := info.dialect.insertSql(criteria)
	sql = info.dialect.substituteMarkers(sql)
	assert.Equal(info.insertSql, sql)
}

func doTestUpdateSQL(t *testing.T, info dialectSyntax) {
	assert := NewAssert(t)
	model := structPtrToModel(sqlGenSampleData, true, nil)
	criteria := &criteria{model: model}
	criteria.mergePkCondition(info.dialect)
	sql, _ := info.dialect.updateSql(criteria)
	sql = info.dialect.substituteMarkers(sql)
	assert.Equal(info.updateSql, sql)
}

func doTestDeleteSQL(t *testing.T, info dialectSyntax) {
	assert := NewAssert(t)
	model := structPtrToModel(sqlGenSampleData, true, nil)
	criteria := &criteria{model: model}
	criteria.mergePkCondition(info.dialect)
	sql, _ := info.dialect.deleteSql(criteria)
	sql = info.dialect.substituteMarkers(sql)
	assert.Equal(info.deleteSql, sql)
}

func doTestSelectionSQL(t *testing.T, info dialectSyntax) {
	assert := NewAssert(t)
	type User struct {
		Id   int64
		Name string
	}
	type Post struct {
		Id       int64
		AuthorId int64 `qbs:"fk:Author"`
		Author   *User
		Content  string
	}
	model := structPtrToModel(new(Post), true, nil)
	criteria := new(criteria)
	criteria.model = model

	sql, _ := info.dialect.querySql(criteria)
	assert.Equal(info.selectionSql, sql)
}

func doTestQuerySQL(t *testing.T, info dialectSyntax) {
	assert := NewAssert(t)
	type Student struct {
		Name  string
		Grade int
		Score int
	}
	model := structPtrToModel(new(Student), true, nil)
	criteria := new(criteria)
	criteria.model = model
	condition := NewInCondition("grade", []interface{}{6, 7, 8})
	subCondition := NewCondition("score <= ?", 60).Or("score >= ?", 80)
	condition.AndCondition(subCondition)
	criteria.condition = condition
	criteria.orderBys = []order{order{info.dialect.quote("name"), false}, order{info.dialect.quote("grade"), true}}
	criteria.offset = 3
	criteria.limit = 10
	sql, _ := info.dialect.querySql(criteria)
	sql = info.dialect.substituteMarkers(sql)
	assert.Equal(info.querySql, sql)
}

func doTestDropTableSQL(t *testing.T, info dialectSyntax) {
	assert := NewAssert(t)
	sql := info.dialect.dropTableSql("drop_table")
	assert.Equal(info.dropTableIfExistsSql, sql)
}
