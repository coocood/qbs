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
	d := &mysql{}
	d.base.Dialect = d
	return d
}

func (d *mysql) ParseBool(value reflect.Value) bool {
	return value.Int() != 0
}

func (d *mysql) SqlType(f interface{}, size int) string {
	switch f.(type) {
	case time.Time:
		return "timestamp"
	case bool:
		return "boolean"
	case int, int8, int16, int32, uint, uint8, uint16, uint32:
		return "int"
	case int64, uint64:
		return "bigint"
	case float32, float64:
		return "double"
	case []byte:
		if size > 0 && size < 65532 {
			return fmt.Sprintf("varbinary(%d)", size)
		}
		return "longblob"
	case string:
		if size > 0 && size < 65532 {
			return fmt.Sprintf("varchar(%d)", size)
		}
		return "longtext"
	}
	panic("invalid sql type")
}

func (d *mysql) KeywordAutoIncrement() string {
	return "AUTO_INCREMENT"
}

func (d *mysql) IndexExists(mg *Migration, tableName, indexName string) bool {
	var row *sql.Row
	var name string
	row = mg.Db.QueryRow("SELECT INDEX_NAME FROM INFORMATION_SCHEMA.STATISTICS "+
		"WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? AND INDEX_NAME = ?", mg.DbName, tableName, indexName)
	row.Scan(&name)
	return name != ""
}

func (d *mysql) PrimaryKeySql(isString bool, size int) string {
	if isString {
		return fmt.Sprintf("varchar(%d) PRIMARY KEY", size)
	}
	return "bigint PRIMARY KEY AUTO_INCREMENT"
}
