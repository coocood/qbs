package qbs

import (
	"bytes"
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type postgres struct {
	base
}

func NewPostgres() Dialect {
	d := new(postgres)
	d.base.dialect = d
	return d
}

func DefaultPostgresDataSourceName(dbName string) *DataSourceName {
	dsn := new(DataSourceName)
	dsn.Dialect = NewPostgres()
	dsn.Username = "postgres"
	dsn.DbName = dbName
	dsn.Append("sslmode", "disable")
	return dsn
}

func (d postgres) quote(s string) string {
	segs := strings.Split(s, ".")
	buf := new(bytes.Buffer)
	buf.WriteByte('"')
	buf.WriteString(segs[0])
	for i := 1; i < len(segs); i++ {
		buf.WriteString(`"."`)
		buf.WriteString(segs[i])
	}
	buf.WriteByte('"')
	return buf.String()
}

func (d postgres) sqlType(f interface{}, size int) string {
	fieldValue := reflect.ValueOf(f)
	switch fieldValue.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return "integer"
	case reflect.Uint, reflect.Uint64, reflect.Int, reflect.Int64:
		return "bigint"
	case reflect.Float32, reflect.Float64:
		return "double precision"
	case reflect.String:
		if size > 0 && size < 65532 {
			return fmt.Sprintf("varchar(%d)", size)
		}
		return "text"
	case reflect.Slice:
		if reflect.TypeOf(f).Elem().Kind() == reflect.Uint8 {
			if size > 0 && size < 65532 {
				return fmt.Sprintf("varbinary(%d)", size)
			}
			return "bytea"
		}
	case reflect.Struct:
		switch fieldValue.Interface().(type) {
		case time.Time:
			return "timestamp with time zone"
		case sql.NullBool:
			return "boolean"
		case sql.NullInt64:
			return "bigint"
		case sql.NullFloat64:
			return "double precision"
		case sql.NullString:
			if size > 0 && size < 65532 {
				return fmt.Sprintf("varchar(%d)", size)
			}
			return "text"
		}
	}
	panic("invalid sql type")
}

func (d postgres) insert(q *Qbs) (int64, error) {
	sql, args := d.dialect.insertSql(q.criteria)
	row := q.QueryRow(sql, args...)
	value := q.criteria.model.pk.value
	var err error
	var id int64
	if _, ok := value.(int64); ok {
		err = row.Scan(&id)
	} else if _, ok := value.(string); ok {
		var str string
		err = row.Scan(&str)
	}
	return id, err
}

func (d postgres) insertSql(criteria *criteria) (string, []interface{}) {
	sql, values := d.base.insertSql(criteria)
	sql += " RETURNING " + d.dialect.quote(criteria.model.pk.name)
	return sql, values
}

func (d postgres) indexExists(mg *Migration, tableName, indexName string) bool {
	var row *sql.Row
	var name string
	query := "SELECT indexname FROM pg_indexes "
	query += "WHERE tablename = ? AND indexname = ?"
	query = d.substituteMarkers(query)
	row = mg.db.QueryRow(query, tableName, indexName)
	row.Scan(&name)
	return name != ""
}

func (d postgres) substituteMarkers(query string) string {
	position := 1
	buf := new(bytes.Buffer)
	for i := 0; i < len(query); i++ {
		c := query[i]
		if c == '?' {
			buf.WriteByte('$')
			buf.WriteString(strconv.Itoa(position))
			position++
		} else {
			buf.WriteByte(c)
		}
	}
	return buf.String()
}

func (d postgres) columnsInTable(mg *Migration, table interface{}) map[string]bool {
	tn := tableName(table)
	columns := make(map[string]bool)
	query := "SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = ?"
	query = mg.dialect.substituteMarkers(query)
	rows, err := mg.db.Query(query, tn)
	defer rows.Close()
	if err != nil {
		panic(err)
	}
	for rows.Next() {
		column := ""
		err := rows.Scan(&column)
		if err == nil {
			columns[column] = true
		}
	}
	return columns
}

func (d postgres) primaryKeySql(isString bool, size int) string {
	if isString {
		return "text PRIMARY KEY"
	}
	return "bigserial PRIMARY KEY"
}
