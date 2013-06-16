package qbs

import (
	"fmt"
	"reflect"
	"strings"
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

type DataSourceName struct {
	DbName     string
	Username   string
	Password   string
	UnixSocket bool
	Host       string
	Port       string
	Variables  []string
	Dialect    Dialect
}

func (dsn *DataSourceName) String() string {
	if dsn.Dialect == nil {
		panic("DbDialect is not set")
	}
	switch dsn.Dialect.(type) {
	case *mysql:
		dsnformat := "%v@%v/%v%v"
		login := dsn.Username
		if dsn.Password != "" {
			login += ":" + dsn.Password
		}
		var address string
		if dsn.Host != "" {
			address = dsn.Host
			if dsn.Port != "" {
				address += ":" + dsn.Port
			}
			protocol := "tcp"
			if dsn.UnixSocket {
				protocol = "unix"
			}
			address = protocol + "(" + address + ")"
		}
		var variables string
		if dsn.Variables != nil {
			variables = "?" + strings.Join(dsn.Variables, "&")
		}
		return fmt.Sprintf(dsnformat, login, address, dsn.DbName, variables)
	case *sqlite3:
		return dsn.DbName
	case *postgres:
		pairs := []string{"user=" + dsn.Username}
		if dsn.Password != "" {
			pairs = append(pairs, "password="+dsn.Password)
		}
		if dsn.DbName != "" {
			pairs = append(pairs, "dbname="+dsn.DbName)
		}
		pairs = append(pairs, dsn.Variables...)
		if dsn.Host != "" {
			host := dsn.Host
			if dsn.UnixSocket {
				host = "/" + host
			}
			pairs = append(pairs, "host="+host)
		}
		if dsn.Port != "" {
			pairs = append(pairs, "port="+dsn.Port)
		}
		return strings.Join(pairs, " ")
	}
	panic("Unknown DbDialect.")
}

func (dsn *DataSourceName) Append(key, value string) *DataSourceName {
	dsn.Variables = append(dsn.Variables, key+"="+value)
	return dsn
}

func RegisterWithDataSourceName(dsn *DataSourceName) {
	var driverName string
	switch dsn.Dialect.(type) {
	case *mysql:
		driverName = "mysql"
	case *sqlite3:
		driverName = "sqlite3"
	case *postgres:
		driverName = "postgres"
	}
	dbName := dsn.DbName
	if driverName == "sqlite3" {
		dbName = ""
	}
	Register(driverName, dsn.String(), dbName, dsn.Dialect)
}
