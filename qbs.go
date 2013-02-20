package qbs

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
)

type Qbs struct {
	Db           *sql.DB
	Dialect      Dialect
	Log          bool
	Tx           *sql.Tx
	criteria     *Criteria
	firstTxError error
}

type Validator interface {
	Validate(*Qbs) error
}

// New creates a new Qbs instance using the specified DB and dialect.
func New(database *sql.DB, dialect Dialect) *Qbs {
	q := &Qbs{
		Db:      database,
		Dialect: dialect,
	}
	q.Reset()
	return q
}

// Create a new criteria for subsequent query
func (q *Qbs) Reset() {
	q.criteria = new(Criteria)
}

// Begin create a transaction object internally
// You can perform queries with the same Qbs object
// no matter it is in transaction or not.
// It panics if it's already in a transaction.
func (q *Qbs) Begin() {
	if q.Tx != nil {
		panic("cannot start nested transaction")
	}
	tx, err := q.Db.Begin()
	q.Tx = tx
	if err != nil {
		panic(err)
	}
}

func (q *Qbs) updateTxError(e error) error {
	if e != nil {
		q.log("ERROR: ", e)
		// don't shadow the first error
		if q.firstTxError == nil {
			q.firstTxError = e
		}
	}
	return e
}

// Commit commits a started transaction and will report the first error that
// occurred inside the transaction.
func (q *Qbs) Commit() error {
	err := q.Tx.Commit()
	q.updateTxError(err)
	q.Tx = nil
	return q.firstTxError
}

// Rollback rolls back a started transaction.
func (q *Qbs) Rollback() error {
	err := q.Tx.Rollback()
	q.Tx = nil
	return err
}

// Where is a shortcut method to call Condtion(NewCondtition(expr, args...)).
func (q *Qbs) Where(expr string, args ...interface{}) *Qbs {
	q.criteria.condition = NewCondition(expr, args...)
	return q
}

//Condition defines the SQL "WHERE" clause
//If other condition can be inferred by the struct argument in
//Find method, it will be merged with AND
func (q *Qbs) Condition(condition *Condition) *Qbs {
	q.criteria.condition = condition
	return q
}

func (q *Qbs) Limit(limit int) *Qbs {
	q.criteria.limit = limit
	return q
}

func (q *Qbs) Offset(offset int) *Qbs {
	q.criteria.offset = offset
	return q
}

func (q *Qbs) OrderBy(path string) *Qbs {
	q.criteria.orderBy = q.Dialect.Quote(path)
	return q
}

func (q *Qbs) OrderByDesc(path string) *Qbs {
	q.criteria.orderBy = q.Dialect.Quote(path)
	q.criteria.orderDesc = true
	return q
}

func (q *Qbs) OmitFields(fieldName ...string) *Qbs {
	q.criteria.omitFields = fieldName
	return q
}

// Perform select query by parsing the struct's type and then fill the values into the struct
// All fields of supported types in the struct will be added in select clause.
// If Id value is provided, it will be added into the where clause
// If a foreign key field with its referenced struct pointer field are provided,
// It will perform a join query, the referenced struct pointer field will be filled in
// the values obtained by the query.
func (q *Qbs) Find(structPtr interface{}) error {
	q.criteria.model = structPtrToModel(structPtr, true, q.criteria.omitFields)
	q.criteria.limit = 1
	if !q.criteria.model.pkZero() {
		idPath := q.Dialect.Quote(q.criteria.model.Table) + "." + q.Dialect.Quote(q.criteria.model.Pk.Name)
		idCondition := NewCondition(idPath+" = ?", q.criteria.model.Pk.Value)
		if q.criteria.condition == nil {
			q.criteria.condition = idCondition
		} else {
			q.criteria.condition = idCondition.AndCondition(q.criteria.condition)
		}
	}
	query, args := q.Dialect.QuerySql(q.criteria)
	return q.doQueryRow(structPtr, query, args...)
}

// Similar to Find, except that FindAll accept pointer of slice of struct pointer,
// rows will be appended to the slice.
func (q *Qbs) FindAll(ptrOfSliceOfStructPtr interface{}) error {
	strucType := reflect.TypeOf(ptrOfSliceOfStructPtr).Elem().Elem().Elem()
	strucPtr := reflect.New(strucType).Interface()
	q.criteria.model = structPtrToModel(strucPtr, true, q.criteria.omitFields)
	query, args := q.Dialect.QuerySql(q.criteria)
	return q.doQueryRows(ptrOfSliceOfStructPtr, query, args...)
}

func (q *Qbs) doQueryRow(out interface{}, query string, args ...interface{}) error {
	defer q.Reset()
	rowValue := reflect.ValueOf(out)
	stmt, err := q.Prepare(query)
	if err != nil {
		stmt.Close()
		return q.updateTxError(err)
	}
	rows, err := stmt.Query(args...)
	defer rows.Close()
	if err != nil {
		return q.updateTxError(err)
	}
	if rows.Next() {
		err = q.scanRows(rowValue, rows)
		if err != nil {
			return err
		}
	} else {
		return sql.ErrNoRows
	}
	return nil
}

func (q *Qbs) doQueryRows(out interface{}, query string, args ...interface{}) error {
	defer q.Reset()
	sliceValue := reflect.Indirect(reflect.ValueOf(out))
	sliceType := sliceValue.Type().Elem().Elem()
	q.log(query, args...)
	stmt, err := q.Prepare(query)
	if err != nil {
		stmt.Close()
		return q.updateTxError(err)
	}

	rows, err := stmt.Query(args...)
	defer rows.Close()
	if err != nil {
		return q.updateTxError(err)
	}
	for rows.Next() {
		rowValue := reflect.New(sliceType)
		err = q.scanRows(rowValue, rows)
		if err != nil {
			return err
		}
		sliceValue.Set(reflect.Append(sliceValue, rowValue))
	}
	return nil
}

func (q *Qbs) scanRows(rowValue reflect.Value, rows *sql.Rows) (err error) {
	cols, _ := rows.Columns()
	containers := make([]interface{}, 0, len(cols))
	for i := 0; i < cap(containers); i++ {
		var v interface{}
		containers = append(containers, &v)
	}
	err = rows.Scan(containers...)
	if err != nil {
		return
	}
	for i, v := range containers {
		value := reflect.Indirect(reflect.ValueOf(v))
		key := cols[i]
		paths := strings.Split(key, "___")
		if len(paths) == 2 {
			subStruct := rowValue.Elem().FieldByName(snakeToUpperCamel(paths[0]))
			if subStruct.IsNil() {
				subStruct.Set(reflect.New(subStruct.Type().Elem()))
			}
			subField := subStruct.Elem().FieldByName(snakeToUpperCamel(paths[1]))
			if subField.IsValid() {
				err = q.Dialect.SetModelValue(value, subField)
				if err != nil {
					return
				}
			}
		} else {
			field := rowValue.Elem().FieldByName(snakeToUpperCamel(key))
			if field.IsValid() {
				err = q.Dialect.SetModelValue(value, field)
				if err != nil {
					return
				}
			}
		}
	}
	return
}

// Same as sql.Db.Exec or sql.Tx.Exec depends on if transaction has began
func (q *Qbs) Exec(query string, args ...interface{}) (sql.Result, error) {
	defer q.Reset()
	query = q.Dialect.SubstituteMarkers(query)
	q.log(query, args...)
	stmt, err := q.Prepare(query)
	if err != nil {
		return nil, q.updateTxError(err)
	}
	defer stmt.Close()
	result, err := stmt.Exec(args...)
	if err != nil {
		return nil, q.updateTxError(err)
	}
	return result, nil
}

// Same as sql.Db.QueryRow or sql.Tx.QueryRow depends on if transaction has began
func (q *Qbs) QueryRow(query string, args ...interface{}) *sql.Row {
	q.log(query, args...)
	query = q.Dialect.SubstituteMarkers(query)
	if q.Tx != nil {
		return q.Tx.QueryRow(query, args...)
	}
	return q.Db.QueryRow(query, args...)
}

// Same as sql.Db.Query or sql.Tx.Query depends on if transaction has began
func (q *Qbs) Query(query string, args ...interface{}) (*sql.Rows, error) {
	q.log(query, args...)
	query = q.Dialect.SubstituteMarkers(query)
	if q.Tx != nil {
		return q.Tx.Query(query, args...)
	}
	return q.Db.Query(query, args...)
}

// Same as sql.Db.Prepare or sql.Tx.Prepare depends on if transaction has began
func (q *Qbs) Prepare(query string) (*sql.Stmt, error) {
	if q.Tx != nil {
		return q.Tx.Prepare(query + ";")
	}
	return q.Db.Prepare(query + ";")
}

// If Id value is not provided, save will insert the record, and the Id value will
// be filled in the struct after insertion.
// If Id value is provided, save will try to update the record first, if no row is affected,
// It will insert the record.
// If struct implements Validator interface, it will be validated first
func (q *Qbs) Save(structPtr interface{}) (affected int64, err error) {
	if v, ok := structPtr.(Validator); ok {
		err = v.Validate(q)
		if err != nil {
			return
		}
	}
	model := structPtrToModel(structPtr, true, q.criteria.omitFields)
	if model.Pk == nil {
		panic("no primary key field")
	}
	q.criteria.model = model
	preservedCriteria := q.criteria
	now := q.Dialect.Now()
	var id int64 = 0
	updateModelField := model.timeFiled("updated")
	if updateModelField != nil {
		updateModelField.Value = now
	}
	createdModelField := model.timeFiled("created")
	canBeUpdate := !model.pkZero()
	var isInsert bool
	if canBeUpdate {
		q.criteria.mergePkCondition(q.Dialect)
		affected, err = q.Dialect.Update(q)
		if affected == 0 && err == nil {
			if createdModelField != nil {
				createdModelField.Value = now
			}
			q.criteria = preservedCriteria
			id, err = q.Dialect.Insert(q)
			isInsert = true
			if err == nil {
				affected = 1
			}
		}
	} else {
		if createdModelField != nil {
			createdModelField.Value = now
		}
		id, err = q.Dialect.Insert(q)
		isInsert = true
		if err == nil {
			affected = 1
		}
	}
	if err == nil {
		structValue := reflect.Indirect(reflect.ValueOf(structPtr))
		if _, ok := model.Pk.Value.(int64); ok && id != 0 {
			idField := structValue.FieldByName(model.Pk.CamelName)
			idField.SetInt(id)
		}
		if updateModelField != nil {
			updateField := structValue.FieldByName(updateModelField.CamelName)
			updateField.Set(reflect.ValueOf(now))
		}
		if isInsert {
			if createdModelField != nil {
				createdField := structValue.FieldByName(createdModelField.CamelName)
				createdField.Set(reflect.ValueOf(now))
			}
		}
	}
	return affected, err
}

// If the struct type implements Validator interface, values will be validated before update.
// In order to avoid inadvertently update the struct field to zero value, it is better to define a
// temporary struct in function, only define the fields that should be updated.
// But the temporary struct can not implement Validator interface, we have to validate values manually.
// The update condition can be inferred by the Id value of the struct.
// If neither Id value or condition are provided, it would cause runtime panic
func (q *Qbs) Update(structPtr interface{}) (affected int64, err error) {
	if v, ok := structPtr.(Validator); ok {
		err := v.Validate(q)
		if err != nil {
			return 0, err
		}
	}
	model := structPtrToModel(structPtr, true, q.criteria.omitFields)
	q.criteria.model = model
	q.criteria.mergePkCondition(q.Dialect)
	if q.criteria.condition == nil {
		panic("Can not update without condition")
	}
	return q.Dialect.Update(q)
}

// The delete condition can be inferred by the Id value of the struct
// If neither Id value or condition are provided, it would cause runtime panic
func (q *Qbs) Delete(structPtr interface{}) (affected int64, err error) {
	model := structPtrToModel(structPtr, true, q.criteria.omitFields)
	q.criteria.model = model
	q.criteria.mergePkCondition(q.Dialect)
	if q.criteria.condition == nil {
		panic("Can not delete without condition")
	}
	return q.Dialect.Delete(q)
}

// This method can be used to validate unique column before trying to save
// The table parameter can be either a string or a struct pointer
func (q *Qbs) ContainsValue(table interface{}, column string, value interface{}) bool {
	quotedColumn := q.Dialect.Quote(column)
	quotedTable := q.Dialect.Quote(tableName(table))
	query := fmt.Sprintf("SELECT %v FROM %v WHERE %v = ?", quotedColumn, quotedTable, quotedColumn)
	row := q.QueryRow(query, value)
	var result interface{}
	err := row.Scan(&result)
	return err == nil
}

func (q *Qbs) log(query string, args ...interface{}) {
	if q.Log {
		fmt.Println(query)
		if len(args) > 0 {
			fmt.Println(args...)
		}
	}
}
