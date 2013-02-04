package qbs

import (
	"github.com/coocood/assrt"
	"testing"
	"time"
)

func TestParseTags(t *testing.T) {
	assert := assrt.NewAssert(t)
	m := parseTags(`fk`)
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
		A string `sql:"-"`
		B string
	}
	m := structPtrToModel(&Schema{}, true)
	assert.OneLen(m.Fields)
}

func TestInterfaceToModelWithReference(t *testing.T) {
	assert := assrt.NewAssert(t)
	type parent struct {
		Id    Id
		Name  string
		Value string
	}
	type table struct {
		ColPrimary Id
		FatherId   int64 `sql:"fk:Father"`
		Father     *parent
	}
	table1 := &table{
		6, 3, &parent{3, "Mrs. A", "infinite"},
	}
	m := structPtrToModel(table1, true)
	ref, ok := m.Refs["Father"]
	assert.MustTrue(ok)
	f := ref.Model.Fields[1]
	x, ok := f.Value.(string)
	assert.True(ok)
	assert.Equal("Mrs. A", x)
}

type indexedTable struct {
	ColPrimary Id
	ColNotNull string `sql:"notnull,default:'banana'"`
	ColVarChar string `sql:"size:64"`
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
	m := structPtrToModel(table1, true)
	assert.Equal("col_primary", m.Pk.Name)
	assert.Equal(4, len(m.Fields))
	assert.Equal(2, len(m.Indexes))
	assert.Equal("col_primary_col_time", m.Indexes[0].Name)
	assert.True(!m.Indexes[0].Unique)
	assert.Equal("col_var_char_col_time", m.Indexes[1].Name)
	assert.True(m.Indexes[1].Unique)

	f := m.Fields[0]
	id, _ := f.Value.(Id)
	assert.Equal(6, id)
	assert.True(f.PrimaryKey())

	f = m.Fields[1]
	assert.Equal("'banana'", f.Default())

	f = m.Fields[2]
	str, _ := f.Value.(string)
	assert.Equal("orange", str)
	assert.Equal(64, f.Size())

	f = m.Fields[3]
	tm, _ := f.Value.(time.Time)
	assert.Equal(now, tm)
}

func TestInterfaceToSubModel(t *testing.T) {
	assert := assrt.NewAssert(t)
	type User struct {
		Id   Id
		Name string
	}
	type Post struct {
		Id       Id
		AuthorId int64 `sql:"fk:Author"`
		Author   *User
		Content  string
	}
	pst := new(Post)
	model := structPtrToModel(pst, true)
	assert.OneLen(model.Refs)
}
