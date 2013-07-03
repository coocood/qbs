package qbs

import (
	"database/sql"
	"fmt"
	"reflect"
	"time"
)

type mysql struct {
	base
}

func NewMysql() Dialect {
	d := new(mysql)
	d.base.dialect = d
	return d
}

func DefaultMysqlDataSourceName(dbName string) *DataSourceName {
	dsn := new(DataSourceName)
	dsn.Dialect = new(mysql)
	dsn.Username = "root"
	dsn.DbName = dbName
	dsn.Append("loc", "Local")
	dsn.Append("charset", "utf8")
	dsn.Append("parseTime", "true")
	return dsn
}

func (d mysql) parseBool(value reflect.Value) bool {
	return value.Int() != 0
}

func (d mysql) sqlType(f interface{}, size int) string {
	fieldValue := reflect.ValueOf(f)
	switch fieldValue.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return "int"
	case reflect.Uint, reflect.Uint64, reflect.Int, reflect.Int64:
		return "bigint"
	case reflect.Float32, reflect.Float64:
		return "double"
	case reflect.String:
		if size > 0 && size < 65532 {
			return fmt.Sprintf("varchar(%d)", size)
		}
		return "longtext"
	case reflect.Slice:
		if reflect.TypeOf(f).Elem().Kind() == reflect.Uint8 {
			if size > 0 && size < 65532 {
				return fmt.Sprintf("varbinary(%d)", size)
			}
			return "longblob"
		}
	case reflect.Struct:
		switch fieldValue.Interface().(type) {
		case time.Time:
			return "timestamp"
		case sql.NullBool:
			return "boolean"
		case sql.NullInt64:
			return "bigint"
		case sql.NullFloat64:
			return "double"
		case sql.NullString:
			if size > 0 && size < 65532 {
				return fmt.Sprintf("varchar(%d)", size)
			}
			return "longtext"
		}
	}
	panic("invalid sql type")
}

func (d mysql) indexExists(mg *Migration, tableName, indexName string) bool {
	var row *sql.Row
	var name string
	row = mg.db.QueryRow("SELECT INDEX_NAME FROM INFORMATION_SCHEMA.STATISTICS "+
		"WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? AND INDEX_NAME = ?", mg.dbName, tableName, indexName)
	row.Scan(&name)
	return name != ""
}

func (d mysql) primaryKeySql(isString bool, size int) string {
	if isString {
		return fmt.Sprintf("varchar(%d) PRIMARY KEY", size)
	}
	return "bigint PRIMARY KEY AUTO_INCREMENT"
}
