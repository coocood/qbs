package qbs

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type postgres struct {
	base
}

func NewPostgres() Dialect {
	d := &postgres{}
	d.base.Dialect = d
	return d
}

func (d *postgres) Quote(s string) string {
	sep := "."
	a := []string{}
	c := strings.Split(s, sep)
	for _, v := range c {
		a = append(a, fmt.Sprintf(`"%s"`, v))
	}
	return strings.Join(a, sep)
}

func (d *postgres) SqlType(f interface{}, size int) string {
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

func (d *postgres) Insert(q *Qbs) (int64, error) {
	var id int64
	sql, args := d.Dialect.InsertSql(q.criteria)
	rows, err := q.Query(sql, args...)
	if err != nil {
		return id, err
	}
	value := q.criteria.model.Pk.Value
	rows.Next() // Must call Next() before Scan() on *Rows
	if _, ok := value.(int64); ok {
		err = rows.Scan(&id)
	} else if _, ok := value.(string); ok {
		var str string
		err = rows.Scan(&str)
	}
	return id, err
}

func (d *postgres) InsertSql(criteria *Criteria) (string, []interface{}) {
	sql, values := d.base.InsertSql(criteria)
	sql += " RETURNING " + d.Dialect.Quote(criteria.model.Pk.Name)
	return sql, values
}

func (d *postgres) KeywordAutoIncrement() string {
	// postgres has not auto increment keyword, uses SERIAL type
	return ""
}

func (d *postgres) IndexExists(mg *Migration, tableName, indexName string) bool {
	var row *sql.Row
	var name string
	query := "SELECT indexname FROM pg_indexes "
	query += "WHERE tablename = ? AND indexname = ?"
	query = d.SubstituteMarkers(query)
	row = mg.Db.QueryRow(query, tableName, indexName)
	row.Scan(&name)
	return name != ""
}

func (d *postgres) SubstituteMarkers(query string) string {
	position := 1
	chunks := make([]string, 0, len(query)*2)
	for _, v := range query {
		if v == '?' {
			chunks = append(chunks, fmt.Sprintf("$%d", position))
			position++
		} else {
			chunks = append(chunks, string(v))
		}
	}
	return strings.Join(chunks, "")
}

func (d *postgres) ColumnsInTable(mg *Migration, table interface{}) map[string]bool {
	tn := tableName(table)
	columns := make(map[string]bool)
	query := "SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = ?"
	query = mg.Dialect.SubstituteMarkers(query)
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

func (d *postgres) PrimaryKeySql(isString bool, size int) string {
	if isString {
		return "text PRIMARY KEY"
	}
	return "bigserial PRIMARY KEY"
}
