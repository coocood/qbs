package qbs

import (
	"reflect"
	"time"
	"unsafe"
)

type Sqlite3 struct {
	base
}

func NewSqlite3() Dialect {
	d := &Sqlite3{}
	d.base.Dialect = d
	return d
}

func (d *Sqlite3) SqlType(f interface{}, size int) string {
	switch f.(type) {
	case bool:
		return "integer"
	case int, int8, int16, int32, uint, uint8, uint16, uint32, int64, uint64:
		return "integer"
	case float32, float64:
		return "real"
	case []byte:
		return "text"
	case string:
		return "text"
	case time.Time:
		return "text"
	}
	panic("invalid sql type")
}

func (d *Sqlite3) SetModelValue(value reflect.Value, field reflect.Value) error {
	switch field.Type().Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		field.SetInt(value.Elem().Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		// reading uint from int value causes panic
		switch value.Elem().Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			field.SetUint(uint64(value.Elem().Int()))
		default:
			field.SetUint(value.Elem().Uint())
		}
	case reflect.Bool:
		if value.Elem().Int() == 0 {
			field.SetBool(false)
		} else {
			field.SetBool(true)
		}
	case reflect.Float32, reflect.Float64:
		field.SetFloat(value.Elem().Float())
	case reflect.String:
		field.SetString(value.Elem().String())
	case reflect.Slice:
		if reflect.TypeOf(value.Interface()).Elem().Kind() == reflect.Uint8 {
			field.SetBytes(value.Elem().Bytes())
		}
	case reflect.Struct:
		if _, ok := field.Interface().(time.Time); ok {
			t, err := time.Parse("2006-01-02 15:04:05", value.Elem().String())
			if err != nil {
				return err
			}
			v := reflect.NewAt(reflect.TypeOf(time.Time{}), unsafe.Pointer(&t))
			field.Set(v.Elem())
		}
	}
	return nil
}

func (d *Sqlite3) IndexExists(mg *Migration, tableName string, indexName string) bool {
	return false
}

func (d *Sqlite3) ColumnsInTable(mg *Migration, table interface{}) map[string]bool {
	tn := tableName(table)
	columns := make(map[string]bool)
	query := "PRAGMA table_info('" + tn + "')"
	rows, err := mg.Db.Query(query)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		cols, _ := rows.Columns()
		containers := make([]interface{}, 0, len(cols))
		for i := 0; i < cap(containers); i++ {
			var v interface{}
			containers = append(containers, &v)
		}
		err = rows.Scan(containers...)
		value := reflect.Indirect(reflect.ValueOf(containers[1]))
		if err == nil {
			columns[value.Elem().String()] = true
		}
	}
	return columns
}

func (d *Sqlite3) PrimaryKeySql(isString bool, size int) string {
	if isString {
		return "text PRIMARY KEY NOT NULL"
	}
	return "integer PRIMARY KEY AUTOINCREMENT NOT NULL"
}
