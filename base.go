package qbs

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

type base struct {
	Dialect Dialect
}

func (d *base) SubstituteMarkers(query string) string {
	return query
}

func (d *base) Quote(s string) string {
	sep := "."
	a := []string{}
	c := strings.Split(s, sep)
	for _, v := range c {
		a = append(a, fmt.Sprintf("`%s`", v))
	}
	return strings.Join(a, sep)
}

func (d *base) Now() time.Time {
	return time.Now()
}

func (d *base) ParseBool(value reflect.Value) bool {
	return value.Bool()
}

func (d *base) SetModelValue(driverValue, fieldValue reflect.Value) error {
	switch fieldValue.Type().Kind() {
	case reflect.Bool:
		fieldValue.SetBool(d.Dialect.ParseBool(driverValue.Elem()))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fieldValue.SetInt(driverValue.Elem().Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		// reading uint from int value causes panic
		switch driverValue.Elem().Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			fieldValue.SetUint(uint64(driverValue.Elem().Int()))
		default:
			fieldValue.SetUint(driverValue.Elem().Uint())
		}
	case reflect.Float32, reflect.Float64:
		fieldValue.SetFloat(driverValue.Elem().Float())
	case reflect.String:
		fieldValue.SetString(string(driverValue.Elem().Bytes()))
	case reflect.Slice:
		if reflect.TypeOf(driverValue.Interface()).Elem().Kind() == reflect.Uint8 {
			fieldValue.SetBytes(driverValue.Elem().Bytes())
		}
	case reflect.Struct:
		if _, ok := fieldValue.Interface().(time.Time); ok {
			fieldValue.Set(driverValue.Elem())
		}
	}
	return nil
}

func (d *base) QuerySql(criteria *Criteria) (string, []interface{}) {
	query := make([]string, 0, 20)
	args := make([]interface{}, 0, 20)

	table := d.Dialect.Quote(criteria.model.Table)
	columns := []string{}
	tables := []string{table}
	hasJoin := len(criteria.model.Refs) > 0
	for _, v := range criteria.model.Fields {
		colName := d.Dialect.Quote(v.Name)
		if hasJoin {
			colName = d.Dialect.Quote(criteria.model.Table) + "." + colName
		}
		columns = append(columns, colName)
	}
	for k, v := range criteria.model.Refs {
		tableAlias := toSnake(k)
		quotedTableAlias := d.Dialect.Quote(tableAlias)
		quotedParentTable := d.Dialect.Quote(v.Model.Table)
		leftKey := table + "." + d.Dialect.Quote(v.RefKey)
		parentPrimary := quotedTableAlias + "." + d.Dialect.Quote(v.Model.Pk.Name)
		joinClause := fmt.Sprintf("LEFT JOIN %v AS %v ON %v = %v", quotedParentTable, quotedTableAlias, leftKey, parentPrimary)
		tables = append(tables, joinClause)
		for _, f := range v.Model.Fields {
			alias := tableAlias + "___" + f.Name
			columns = append(columns, d.Dialect.Quote(tableAlias+"."+f.Name)+" AS "+alias)
		}
	}
	query = append(query, "SELECT", strings.Join(columns, ", "), "FROM", strings.Join(tables, " "))

	if criteria.condition != nil {
		cexpr, cargs := criteria.condition.Merge()
		query = append(query, "WHERE", cexpr)
		args = append(args, cargs...)
	}
	orderByLen := len(criteria.orderBys)
	if orderByLen > 0 {
		query = append(query, "ORDER BY")
		for i, order := range criteria.orderBys {
			query = append(query, order.path)
			if order.desc {
				query = append(query, "DESC")
			}
			if i < orderByLen -1 {
				query = append(query, ",")
			}
		}
	}

	if x := criteria.limit; x > 0 {
		query = append(query, "LIMIT ?")
		args = append(args, criteria.limit)
	}
	if x := criteria.offset; x > 0 {
		query = append(query, "OFFSET ?")
		args = append(args, criteria.offset)
	}
	return d.Dialect.SubstituteMarkers(strings.Join(query, " ")), args
}

func (d *base) Insert(q *Qbs) (int64, error) {
	sql, args := d.Dialect.InsertSql(q.criteria)
	result, err := q.Exec(sql, args...)
	if err != nil {
		return -1, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return -1, err
	}
	return id, nil
}

func (d *base) InsertSql(criteria *Criteria) (string, []interface{}) {
	columns, values := criteria.model.columnsAndValues(false)
	quotedColumns := make([]string, 0, len(columns))
	markers := make([]string, 0, len(columns))
	for _, c := range columns {
		quotedColumns = append(quotedColumns, d.Dialect.Quote(c))
		markers = append(markers, "?")
	}
	sql := fmt.Sprintf(
		"INSERT INTO %v (%v) VALUES (%v)",
		d.Dialect.Quote(criteria.model.Table),
		strings.Join(quotedColumns, ", "),
		strings.Join(markers, ", "),
	)
	return sql, values
}

func (d *base) Update(q *Qbs) (int64, error) {
	sql, args := d.Dialect.UpdateSql(q.criteria)
	result, err := q.Exec(sql, args...)
	if err != nil {
		return 0, err
	}
	affected, err := result.RowsAffected()
	return affected, err
}

func (d *base) UpdateSql(criteria *Criteria) (string, []interface{}) {
	columns, values := criteria.model.columnsAndValues(true)
	pairs := make([]string, 0, len(columns))
	for _, column := range columns {
		pairs = append(pairs, fmt.Sprintf("%v = ?", d.Dialect.Quote(column)))
	}
	conditionSql, args := criteria.condition.Merge()
	sql := fmt.Sprintf(
		"UPDATE %v SET %v WHERE %v",
		d.Dialect.Quote(criteria.model.Table),
		strings.Join(pairs, ", "),
		conditionSql,
	)
	values = append(values, args...)
	return sql, values
}

func (d *base) Delete(q *Qbs) (int64, error) {
	sql, args := d.Dialect.DeleteSql(q.criteria)
	result, err := q.Exec(sql, args...)
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return affected, err
}

func (d *base) DeleteSql(criteria *Criteria) (string, []interface{}) {
	conditionSql, args := criteria.condition.Merge()
	sql := "DELETE FROM " + d.Dialect.Quote(criteria.model.Table) + " WHERE " + conditionSql
	return sql, args
}

func (d *base) CreateTableSql(model *Model, ifNotExists bool) string {
	a := []string{"CREATE TABLE "}
	if ifNotExists {
		a = append(a, "IF NOT EXISTS ")
	}
	a = append(a, d.Dialect.Quote(model.Table), " ( ")
	for i, field := range model.Fields {
		b := []string{
			d.Dialect.Quote(field.Name),
		}
		if field.PK {
			_, ok := field.Value.(string)
			b = append(b, d.Dialect.PrimaryKeySql(ok, field.Size()))
		} else {
			b = append(b, d.Dialect.SqlType(field.Value, field.Size()))
			if field.NotNull() {
				b = append(b, "NOT NULL")
			}
			if x := field.Default(); x != "" {
				b = append(b, "DEFAULT "+x)
			}
		}
		a = append(a, strings.Join(b, " "))
		if i < len(model.Fields)-1 {
			a = append(a, ", ")
		}
	}
	for _, v := range model.Refs {
		if v.ForeignKey {
			a = append(a, ", FOREIGN KEY (", d.Dialect.Quote(v.RefKey), ") REFERENCES ")
			a = append(a, d.Dialect.Quote(v.Model.Table), " (", d.Dialect.Quote(v.Model.Pk.Name), ") ON DELETE CASCADE")
		}
	}
	a = append(a, " )")
	return strings.Join(a, "")
}

func (d *base) DropTableSql(table string) string {
	a := []string{"DROP TABLE IF EXISTS"}
	a = append(a, d.Dialect.Quote(table))
	return strings.Join(a, " ")
}

func (d *base) AddColumnSql(table, column string, typ interface{}, size int) string {
	return fmt.Sprintf(
		"ALTER TABLE %v ADD COLUMN %v %v",
		d.Dialect.Quote(table),
		d.Dialect.Quote(column),
		d.Dialect.SqlType(typ, size),
	)
}

func (d *base) CreateIndexSql(name, table string, unique bool, columns ...string) string {
	a := []string{"CREATE"}
	if unique {
		a = append(a, "UNIQUE")
	}
	quotedColumns := make([]string, 0, len(columns))
	for _, c := range columns {
		quotedColumns = append(quotedColumns, d.Dialect.Quote(c))
	}
	a = append(a, fmt.Sprintf(
		"INDEX %v ON %v (%v)",
		d.Dialect.Quote(name),
		d.Dialect.Quote(table),
		strings.Join(quotedColumns, ", "),
	))
	return strings.Join(a, " ")
}

func (d *base) ColumnsInTable(mg *Migration, table interface{}) map[string]bool {
	tn := tableName(table)
	columns := make(map[string]bool)
	query := "SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?"
	query = mg.Dialect.SubstituteMarkers(query)
	rows, err := mg.Db.Query(query, mg.DbName, tn)
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
