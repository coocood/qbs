package qbs

import (
	"bytes"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"
)

type base struct {
	dialect Dialect
}

func (d base) substituteMarkers(query string) string {
	return query
}

func (d base) quote(s string) string {
	segs := strings.Split(s, ".")
	buf := new(bytes.Buffer)
	buf.WriteByte('`')
	buf.WriteString(segs[0])
	for i := 1; i < len(segs); i++ {
		buf.WriteString("`.`")
		buf.WriteString(segs[i])
	}
	buf.WriteByte('`')
	return buf.String()
}

func (d base) parseBool(value reflect.Value) bool {
	return value.Bool()
}

func (d base) setPtrValue(driverValue, fieldValue reflect.Value) {
	t := fieldValue.Type().Elem()
	v := reflect.New(t)
	fieldValue.Set(v)
	switch t.Kind() {
	case reflect.String:
		v.Elem().SetString(string(driverValue.Interface().([]uint8)))
	case reflect.Int64:
		v.Elem().SetInt(driverValue.Interface().(int64))
	case reflect.Float64:
		v.Elem().SetFloat(driverValue.Interface().(float64))
	case reflect.Bool:
		v.Elem().SetBool(driverValue.Interface().(bool))
	}
}
func (d base) setModelValue(driverValue, fieldValue reflect.Value) error {
	switch fieldValue.Type().Kind() {
	case reflect.Bool:
		fieldValue.SetBool(d.dialect.parseBool(driverValue.Elem()))
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
	case reflect.Ptr:
		d.setPtrValue(driverValue, fieldValue)
	case reflect.Struct:
		switch fieldValue.Interface().(type) {
		case time.Time:
			fieldValue.Set(driverValue.Elem())
		default:
			if scanner, ok := fieldValue.Addr().Interface().(sql.Scanner); ok {
				return scanner.Scan(driverValue.Interface())
			}
		}

	}
	return nil
}

func (d base) querySql(criteria *criteria) (string, []interface{}) {
	query := new(bytes.Buffer)
	args := make([]interface{}, 0, 20)
	table := d.dialect.quote(criteria.model.table)
	columns := []string{}
	tables := []string{table}
	hasJoin := len(criteria.model.refs) > 0
	for _, v := range criteria.model.fields {
		colName := d.dialect.quote(v.name)
		if hasJoin {
			colName = d.dialect.quote(criteria.model.table) + "." + colName
		}
		columns = append(columns, colName)
	}
	for k, v := range criteria.model.refs {
		tableAlias := StructNameToTableName(k)
		quotedTableAlias := d.dialect.quote(tableAlias)
		quotedParentTable := d.dialect.quote(v.model.table)
		leftKey := table + "." + d.dialect.quote(v.refKey)
		parentPrimary := quotedTableAlias + "." + d.dialect.quote(v.model.pk.name)
		joinClause := fmt.Sprintf("LEFT JOIN %v AS %v ON %v = %v", quotedParentTable, quotedTableAlias, leftKey, parentPrimary)
		tables = append(tables, joinClause)
		for _, f := range v.model.fields {
			alias := tableAlias + "___" + f.name
			columns = append(columns, d.dialect.quote(tableAlias+"."+f.name)+" AS "+alias)
		}
	}
	query.WriteString("SELECT ")
	query.WriteString(strings.Join(columns, ", "))
	query.WriteString(" FROM ")
	query.WriteString(strings.Join(tables, " "))

	if criteria.condition != nil {
		cexpr, cargs := criteria.condition.Merge()
		query.WriteString(" WHERE ")
		query.WriteString(cexpr)
		args = append(args, cargs...)
	}
	orderByLen := len(criteria.orderBys)
	if orderByLen > 0 {
		query.WriteString(" ORDER BY ")
		for i, order := range criteria.orderBys {
			query.WriteString(order.path)
			if order.desc {
				query.WriteString(" DESC")
			}
			if i < orderByLen-1 {
				query.WriteString(", ")
			}
		}
	}

	if x := criteria.limit; x > 0 {
		query.WriteString(" LIMIT ?")
		args = append(args, criteria.limit)
	}
	if x := criteria.offset; x > 0 {
		query.WriteString(" OFFSET ?")
		args = append(args, criteria.offset)
	}
	return d.dialect.substituteMarkers(query.String()), args
}

func (d base) insert(q *Qbs) (int64, error) {
	sql, args := d.dialect.insertSql(q.criteria)
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

func (d base) insertSql(criteria *criteria) (string, []interface{}) {
	columns, values := criteria.model.columnsAndValues(false)
	quotedColumns := make([]string, 0, len(columns))
	markers := make([]string, 0, len(columns))
	for _, c := range columns {
		quotedColumns = append(quotedColumns, d.dialect.quote(c))
		markers = append(markers, "?")
	}
	sql := fmt.Sprintf(
		"INSERT INTO %v (%v) VALUES (%v)",
		d.dialect.quote(criteria.model.table),
		strings.Join(quotedColumns, ", "),
		strings.Join(markers, ", "),
	)
	return sql, values
}

func (d base) update(q *Qbs) (int64, error) {
	sql, args := d.dialect.updateSql(q.criteria)
	result, err := q.Exec(sql, args...)
	if err != nil {
		return 0, err
	}
	affected, err := result.RowsAffected()
	return affected, err
}

func (d base) updateSql(criteria *criteria) (string, []interface{}) {
	columns, values := criteria.model.columnsAndValues(true)
	pairs := make([]string, 0, len(columns))
	for _, column := range columns {
		pairs = append(pairs, fmt.Sprintf("%v = ?", d.dialect.quote(column)))
	}
	conditionSql, args := criteria.condition.Merge()
	sql := fmt.Sprintf(
		"UPDATE %v SET %v WHERE %v",
		d.dialect.quote(criteria.model.table),
		strings.Join(pairs, ", "),
		conditionSql,
	)
	values = append(values, args...)
	return sql, values
}

func (d base) delete(q *Qbs) (int64, error) {
	sql, args := d.dialect.deleteSql(q.criteria)
	result, err := q.Exec(sql, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (d base) deleteSql(criteria *criteria) (string, []interface{}) {
	conditionSql, args := criteria.condition.Merge()
	sql := "DELETE FROM " + d.dialect.quote(criteria.model.table) + " WHERE " + conditionSql
	return sql, args
}

func (d base) createTableSql(model *model, ifNotExists bool) string {
	a := []string{"CREATE TABLE "}
	if ifNotExists {
		a = append(a, "IF NOT EXISTS ")
	}
	a = append(a, d.dialect.quote(model.table), " ( ")
	for i, field := range model.fields {
		b := []string{
			d.dialect.quote(field.name),
		}
		if field.pk {
			_, ok := field.value.(string)
			b = append(b, d.dialect.primaryKeySql(ok, field.size))
		} else {
			b = append(b, d.dialect.sqlType(*field))
			if field.notnull {
				b = append(b, "NOT NULL")
			}
			if x := field.dfault; x != "" {
				b = append(b, "DEFAULT "+x)
			}
		}
		a = append(a, strings.Join(b, " "))
		if i < len(model.fields)-1 {
			a = append(a, ", ")
		}
	}
	for _, v := range model.refs {
		if v.foreignKey {
			a = append(a, ", FOREIGN KEY (", d.dialect.quote(v.refKey), ") REFERENCES ")
			a = append(a, d.dialect.quote(v.model.table), " (", d.dialect.quote(v.model.pk.name), ") ON DELETE CASCADE")
		}
	}
	a = append(a, " )")
	return strings.Join(a, "")
}

func (d base) dropTableSql(table string) string {
	a := []string{"DROP TABLE IF EXISTS"}
	a = append(a, d.dialect.quote(table))
	return strings.Join(a, " ")
}

func (d base) addColumnSql(table string, column modelField) string {
	return fmt.Sprintf(
		"ALTER TABLE %v ADD COLUMN %v %v",
		d.dialect.quote(table),
		d.dialect.quote(column.name),
		d.dialect.sqlType(column),
	)
}

func (d base) createIndexSql(name, table string, unique bool, columns ...string) string {
	a := []string{"CREATE"}
	if unique {
		a = append(a, "UNIQUE")
	}
	quotedColumns := make([]string, 0, len(columns))
	for _, c := range columns {
		quotedColumns = append(quotedColumns, d.dialect.quote(c))
	}
	a = append(a, fmt.Sprintf(
		"INDEX %v ON %v (%v)",
		d.dialect.quote(name),
		d.dialect.quote(table),
		strings.Join(quotedColumns, ", "),
	))
	return strings.Join(a, " ")
}

func (d base) columnsInTable(mg *Migration, table interface{}) map[string]bool {
	tn := tableName(table)
	columns := make(map[string]bool)
	query := "SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?"
	query = mg.dialect.substituteMarkers(query)
	rows, err := mg.db.Query(query, mg.dbName, tn)
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

func (d base) catchMigrationError(err error) bool {
	return false
}
