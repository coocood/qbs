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

func (d mysql) sqlType(field modelField) string {
	f := field.value
	fieldValue := reflect.ValueOf(f)
	kind := fieldValue.Kind()
	if field.nullable != reflect.Invalid {
		kind = field.nullable
	}
	switch kind {
	case reflect.Bool:
		return "boolean"
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return "int"
	case reflect.Uint, reflect.Uint64, reflect.Int, reflect.Int64:
		return "bigint"
	case reflect.Float32, reflect.Float64:
		return "double"
	case reflect.String:
		if field.size > 0 && field.size < 65532 {
			return fmt.Sprintf("varchar(%d)", field.size)
		}
		return "longtext"
	case reflect.Slice:
		if reflect.TypeOf(f).Elem().Kind() == reflect.Uint8 {
			if field.size > 0 && field.size < 65532 {
				return fmt.Sprintf("varbinary(%d)", field.size)
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
			if field.size > 0 && field.size < 65532 {
				return fmt.Sprintf("varchar(%d)", field.size)
			}
			return "longtext"
		default:
			if len(field.colType) != 0 {
				switch field.colType {
				case QBS_COLTYPE_BOOL, QBS_COLTYPE_INT, QBS_COLTYPE_BIGINT, QBS_COLTYPE_DOUBLE, QBS_COLTYPE_TIME:
					return field.colType
				case QBS_COLTYPE_TEXT:
					if field.size > 0 && field.size < 65532 {
						return fmt.Sprintf("varchar(%d)", field.size)
					}
					return "longtext"
				default:
					panic("Qbs doesn't support column type " + field.colType + " for MySQL")
				}
			}
		}
	}
	panic("invalid sql type for field:" + field.name)
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
