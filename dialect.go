package qbs

import (
	"reflect"
	"time"
)

type Dialect interface {

	//Substitute "?" marker if database use other symbol as marker
	SubstituteMarkers(query string) string

	// Quote will quote identifiers in a SQL statement.
	Quote(s string) string

	// Now returns the current time. Some drivers have problems with time zones,
	// so this method can be used to prepare the time (e.g. convert to UTC).
	Now() time.Time

	SqlType(f interface{}, size int) string

	ParseBool(value reflect.Value) bool

	SetModelValue(value reflect.Value, field reflect.Value) error

	QuerySql(criteria *Criteria) (sql string, args []interface{})

	Insert(q *Qbs) (Id, error)

	InsertSql(criteria *Criteria) (sql string, args []interface{})

	Update(q *Qbs) (int64, error)

	UpdateSql(criteria *Criteria) (string, []interface{})

	Delete(q *Qbs) (int64, error)

	DeleteSql(criteria *Criteria) (string, []interface{})

	CreateTableSql(model *Model, ifNotExists bool) string

	DropTableSql(table string) string

	AddColumnSql(table, column string, typ interface{}, size int) string

	CreateIndexSql(name, table string, unique bool, columns ...string) string

	IndexExists(mg *Migration, tableName string, indexName string) bool

	ColumnsInTable(mg *Migration, tableName interface{}) map[string]bool

	KeywordNotNull() string

	KeywordDefault(s string) string

	KeywordPrimaryKey() string

	KeywordAutoIncrement() string
}
