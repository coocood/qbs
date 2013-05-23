package qbs

import (
	"github.com/coocood/assrt"
	"testing"
	"time"
)

func TestParseTags(t *testing.T) {
	assert := assrt.NewAssert(t)
	m := parseTags(`fk:User`)
	_, ok := m["fk"]
	assert.True(ok)
	m = parseTags(`notnull,default:'banana'`)
	_, ok = m["notnull"]
	assert.True(ok)
	x, _ := m["default"]
	assert.Equal("'banana'", x)
}

func TestFieldOmit(t *testing.T) {
	assert := assrt.NewAssert(t)
	type Schema struct {
		A string `qbs:"-"`
		B string
		C string
	}
	m := structPtrToModel(&Schema{}, true, []string{"C"})
	assert.OneLen(m.fields)
}

func TestInterfaceToModelWithReference(t *testing.T) {
	assert := assrt.NewAssert(t)
	type parent struct {
		Id    int64
		Name  string
		Value string
	}
	type table struct {
		ColPrimary int64 `qbs:"pk"`
		FatherId   int64 `qbs:"fk:Father"`
		Father     *parent
	}
	table1 := &table{
		6, 3, &parent{3, "Mrs. A", "infinite"},
	}
	m := structPtrToModel(table1, true, nil)
	ref, ok := m.refs["Father"]
	assert.MustTrue(ok)
	f := ref.model.fields[1]
	x, ok := f.value.(string)
	assert.True(ok)
	assert.Equal("Mrs. A", x)
}

type indexedTable struct {
	ColPrimary int64  `qbs:"pk"`
	ColNotNull string `qbs:"notnull,default:'banana'"`
	ColVarChar string `qbs:"size:64"`
	ColTime    time.Time
}

func (table *indexedTable) Indexes(indexes *Indexes) {
	indexes.Add("col_primary", "col_time")
	indexes.AddUnique("col_var_char", "col_time")
}

func TestInterfaceToModel(t *testing.T) {
	assert := assrt.NewAssert(t)
	now := time.Now()
	table1 := &indexedTable{
		ColPrimary: 6,
		ColVarChar: "orange",
		ColTime:    now,
	}
	m := structPtrToModel(table1, true, nil)
	assert.Equal("col_primary", m.pk.name)
	assert.Equal(4, len(m.fields))
	assert.Equal(2, len(m.indexes))
	assert.Equal("col_primary_col_time", m.indexes[0].name)
	assert.True(!m.indexes[0].unique)
	assert.Equal("col_var_char_col_time", m.indexes[1].name)
	assert.True(m.indexes[1].unique)

	f := m.fields[0]
	assert.Equal(6, f.value)
	assert.True(f.pk)

	f = m.fields[1]
	assert.Equal("'banana'", f.dfault())

	f = m.fields[2]
	str, _ := f.value.(string)
	assert.Equal("orange", str)
	assert.Equal(64, f.size())

	f = m.fields[3]
	tm, _ := f.value.(time.Time)
	assert.Equal(now, tm)
}

func TestInterfaceToSubModel(t *testing.T) {
	assert := assrt.NewAssert(t)
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
	pst := new(Post)
	model := structPtrToModel(pst, true, nil)
	assert.OneLen(model.refs)
}

func TestColumnsAndValues(t *testing.T) {
	assert := assrt.NewAssert(t)
	type User struct {
		Id   int64
		Name string
	}
	user := new(User)
	model := structPtrToModel(user, true, nil)
	columns, values := model.columnsAndValues(false)
	assert.MustOneLen(columns)
	assert.MustOneLen(values)
}
