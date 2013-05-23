package qbs

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
	"strconv"
	"bytes"
)

type postgres struct {
	base
}

func NewPostgres() Dialect {
	d := new(postgres)
	d.base.Dialect = d
	return d
}

func (d postgres) quote(s string) string {
	buf := new(bytes.Buffer)
	buf.WriteByte('"')
	segs := strings.Split(s, ".")
	buf.WriteString(segs[0])
	for i:=1; i<len(segs); i++ {
		buf.WriteString(`"."`)
		buf.WriteString(segs[i])
	}
	buf.WriteByte('"')
	return buf.String()
}

func (d postgres) sqlType(f interface{}, size int) string {
	switch f.(type) {
	case time.Time:
		return "timestamp with time zone"
	case bool:
		return "boolean"
	case int, int8, int16, int32, uint, uint8, uint16, uint32:
		return "integer"
	case int64, uint64:
		return "bigint"
	case float32, float64:
		return "double precision"
	case []byte:
		return "bytea"
	case string:
		if size > 0 && size < 65532 {
			return fmt.Sprintf("varchar(%d)", size)
		}
		return "text"
	}
	panic("invalid sql type")
}

func (d postgres) insert(q *Qbs) (int64, error) {
	sql, args := d.Dialect.insertSql(q.criteria)
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
	sql += " RETURNING " + d.Dialect.quote(criteria.model.pk.name)
	return sql, values
}

func (d postgres) indexExists(mg *Migration, tableName, indexName string) bool {
	var row *sql.Row
	var name string
	query := "SELECT indexname FROM pg_indexes "
	query += "WHERE tablename = ? AND indexname = ?"
	query = d.substituteMarkers(query)
	row = mg.Db.QueryRow(query, tableName, indexName)
	row.Scan(&name)
	return name != ""
}

func (d postgres) substituteMarkers(query string) string {
	position := 1
	buf := new(bytes.Buffer)
	for i:=0; i < len(query); i++ {
		c := query[i]
		if c == '?' {
			buf.WriteByte('$')
			buf.WriteString(strconv.Itoa(position))
			position++
		}else{
			buf.WriteByte(c)
		}
	}
	return buf.String()
}

func (d postgres) columnsInTable(mg *Migration, table interface{}) map[string]bool {
	tn := tableName(table)
	columns := make(map[string]bool)
	query := "SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = ?"
	query = mg.Dialect.substituteMarkers(query)
	rows, err := mg.Db.Query(query, tn)
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
