package qbs

import (
	"reflect"
)

type Dialect interface {

	//Substitute "?" marker if database use other symbol as marker
	substituteMarkers(query string) string

	// Quote will quote identifiers in a SQL statement.
	quote(s string) string

	sqlType(f interface{}, size int) string

	parseBool(value reflect.Value) bool

	setModelValue(value reflect.Value, field reflect.Value) error

	querySql(criteria *criteria) (sql string, args []interface{})

	queryM2m(criteria *criteria, field string) (sql string)

	insert(q *Qbs) (int64, error)

	insertSql(criteria *criteria) (sql string, args []interface{})

	update(q *Qbs) (int64, error)

	updateSql(criteria *criteria) (string, []interface{})

	delete(q *Qbs) (int64, error)

	deleteSql(criteria *criteria) (string, []interface{})

	createTableSql(model *model, ifNotExists bool) string

	dropTableSql(table string) string

	addColumnSql(table, column string, typ interface{}, size int) string

	createIndexSql(name, table string, unique bool, columns ...string) string

	indexExists(mg *Migration, tableName string, indexName string) bool

	columnsInTable(mg *Migration, tableName interface{}) map[string]bool

	primaryKeySql(isString bool, size int) string

	catchMigrationError(err error) bool
}
