package qbs

import (
	"database/sql"
	"reflect"
	"time"
	"unsafe"
)

type sqlite3 struct {
	base
}

func NewSqlite3() Dialect {
	d := new(sqlite3)
	d.base.dialect = d
	return d
}

func RegisterSqlite3(dbFileName string) {
	dsn := new(DataSourceName)
	dsn.DbName = dbFileName
	dsn.Dialect = NewSqlite3()
	RegisterWithDataSourceName(dsn)
}

func (d sqlite3) sqlType(field modelField) string {
	f := field.value
	fieldValue := reflect.ValueOf(f)
	kind := fieldValue.Kind()
	if field.nullable != reflect.Invalid {
		kind = field.nullable
	}
	switch kind {
	case reflect.Bool:
		return "integer"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "real"
	case reflect.String:
		return "text"
	case reflect.Slice:
		if reflect.TypeOf(f).Elem().Kind() == reflect.Uint8 {
			return "text"
		}
	case reflect.Struct:
		switch fieldValue.Interface().(type) {
		case time.Time:
			return "text"
		case sql.NullBool:
			return "integer"
		case sql.NullInt64:
			return "integer"
		case sql.NullFloat64:
			return "real"
		case sql.NullString:
			return "text"
		default:
			if len(field.colType) != 0 {
				switch field.colType {
				case QBS_COLTYPE_INT:
					return "integer"
				case QBS_COLTYPE_BIGINT:
					return "integer"
				case QBS_COLTYPE_BOOL:
					return "integer"
				case QBS_COLTYPE_TIME:
					return "text"
				case QBS_COLTYPE_DOUBLE:
					return "real"
				case QBS_COLTYPE_TEXT:
					return "text"
				default:
					panic("Qbs doesn't support column type " + field.colType + "for SQLite3")
				}
			}
		}
	}
	panic("invalid sql type for field:" + field.name)
}

func (d sqlite3) setModelValue(value reflect.Value, field reflect.Value) error {
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
		if value.Elem().Kind() == reflect.Slice {
			field.SetString(string(value.Elem().Bytes()))
		} else {
			field.SetString(value.Elem().String())
		}
	case reflect.Slice:
		if reflect.TypeOf(value.Interface()).Elem().Kind() == reflect.Uint8 {
			field.SetBytes(value.Elem().Bytes())
		}
	case reflect.Ptr:
		d.setPtrValue(value, field)
	case reflect.Struct:
		switch field.Interface().(type) {
		case time.Time:
			var t time.Time
			var err error
			switch value.Elem().Kind() {
			case reflect.String:
				t, err = time.Parse("2006-01-02 15:04:05", value.Elem().String())
				if err != nil {
					return err
				}
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				t = time.Unix(value.Elem().Int(), 0)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				t = time.Unix(int64(value.Elem().Uint()), 0)
			case reflect.Slice:
				t, err = time.Parse("2006-01-02 15:04:05", string(value.Elem().Bytes()))
				if err != nil {
					return err
				}
			}
			v := reflect.NewAt(reflect.TypeOf(time.Time{}), unsafe.Pointer(&t))
			field.Set(v.Elem())
		case sql.NullBool:
			b := true
			if value.Elem().Int() == 0 {
				b = false
			}
			field.Set(reflect.ValueOf(sql.NullBool{b, true}))
		case sql.NullFloat64:
			if f, ok := value.Elem().Interface().(float64); ok {
				field.Set(reflect.ValueOf(sql.NullFloat64{f, true}))
			}
		case sql.NullInt64:
			if i, ok := value.Elem().Interface().(int64); ok {
				field.Set(reflect.ValueOf(sql.NullInt64{i, true}))
			}
		case sql.NullString:
			str := string(value.Elem().String())
			field.Set(reflect.ValueOf(sql.NullString{str, true}))
		}
	}
	return nil
}

func (d sqlite3) indexExists(mg *Migration, tableName string, indexName string) bool {
	query := "PRAGMA index_list('" + tableName + "')"
	rows, err := mg.db.Query(query)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var c0, c1, c2 string
		rows.Scan(&c0, &c1, &c2)
		if c1 == indexName {
			return true
		}
	}
	return false
}

func (d sqlite3) columnsInTable(mg *Migration, table interface{}) map[string]bool {
	tn := tableName(table)
	columns := make(map[string]bool)
	query := "PRAGMA table_info('" + tn + "')"
	rows, err := mg.db.Query(query)
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
			if value.Elem().Kind() == reflect.Slice {
				columns[string(value.Elem().Bytes())] = true
			} else {
				columns[value.Elem().String()] = true
			}
		}
	}
	return columns
}

func (d sqlite3) primaryKeySql(isString bool, size int) string {
	if isString {
		return "text PRIMARY KEY NOT NULL"
	}
	return "integer PRIMARY KEY AUTOINCREMENT NOT NULL"
}
