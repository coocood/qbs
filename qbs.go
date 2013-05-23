package qbs

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
)

var connectionPool chan *sql.DB = make(chan *sql.DB, 10)
var driver, driverSource, dbName string
var dial Dialect

type Qbs struct {
	Db           *sql.DB
	Dialect      Dialect
	Log          bool
	Tx           *sql.Tx
	criteria     *criteria
	firstTxError error
}

type Validator interface {
	Validate(*Qbs) error
}

// Deprecated, call Register and GetQbs instead
// New creates a new Qbs instance using the specified DB and dialect.
func New(database *sql.DB, dialect Dialect) *Qbs {
	q := &Qbs{
		Db:      database,
		Dialect: dialect,
	}
	q.Reset()
	return q
}

//Register a database, should be call at the beginning of the application
func Register(driverName, driverSourceName, databaseName string, dialect Dialect) {
	driver = driverName
	driverSource = driverSourceName
	dial = dialect
	dbName = databaseName
}

//A safe and easy way to work with *Qbs instance without the need to open and close it.
func WithQbs(task func(*Qbs) error) error {
	q, err := GetQbs()
	if err != nil {
		return err
	}
	defer q.Close()
	return task(q)
}

//Get an Qbs instance, should call `defer q.Close()` next, like:
//
//		q, err := qbs.GetQbs()
//	  	if err != nil {
//			fmt.Println(err)
//			return
//		}
//		defer q.Close()
//		...
//
func GetQbs() (q *Qbs, err error) {
	if driver == "" || dial == nil {
		panic("database driver has not been registered, should call Register first.")
	}
	db := GetFreeDB()
	if db == nil {
		db, err = sql.Open(driver, driverSource)
		if err != nil {
			return nil, err
		}
	}
	q = new(Qbs)
	q.Db = db
	q.Dialect = dial
	q.criteria = new(criteria)
	return q, nil
}

//Deprecated: call GetQbs instead.
//Try to get a free *sql.DB from the connection pool.
//This function do not block, if the pool is empty, it returns nil
//Then you should open a new one.
func GetFreeDB() *sql.DB {
	select {
	case db := <-connectionPool:
		return db
	default:
	}
	return nil
}

//The default connection pool size is 10.
func ChangePoolSize(size int) {
	connectionPool = make(chan *sql.DB, size)
}

// Create a new criteria for subsequent query
func (q *Qbs) Reset() {
	q.criteria = new(criteria)
}

// Begin create a transaction object internally
// You can perform queries with the same Qbs object
// no matter it is in transaction or not.
// It panics if it's already in a transaction.
func (q *Qbs) Begin() error {
	if q.Tx != nil {
		panic("cannot start nested transaction")
	}
	tx, err := q.Db.Begin()
	q.Tx = tx
	return err
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
	return q.updateTxError(err)
}

// Where is a shortcut method to call Condtion(NewCondtition(expr, args...)).
func (q *Qbs) Where(expr string, args ...interface{}) *Qbs {
	q.criteria.condition = NewCondition(expr, args...)
	return q
}

//Snakecase column name
func (q *Qbs) WhereEqual(column string, value interface{}) *Qbs {
	q.criteria.condition = NewEqualCondition(column, value)
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
	q.criteria.orderBys = append(q.criteria.orderBys, order{q.Dialect.quote(path), false})
	return q
}

func (q *Qbs) OrderByDesc(path string) *Qbs {
	q.criteria.orderBys = append(q.criteria.orderBys, order{q.Dialect.quote(path), true})
	return q
}

// Camel case field names
func (q *Qbs) OmitFields(fieldName ...string) *Qbs {
	q.criteria.omitFields = fieldName
	return q
}

func (q *Qbs) LoadM2mFields(fieldName ...string) *Qbs {
	q.criteria.loadM2m = fieldName
	return q
}

func (q *Qbs) OmitJoin() *Qbs {
	q.criteria.omitJoin = true
	return q
}

// Perform select query by parsing the struct's type and then fill the values into the struct
// All fields of supported types in the struct will be added in select clause.
// If Id value is provided, it will be added into the where clause
// If a foreign key field with its referenced struct pointer field are provided,
// It will perform a join query, the referenced struct pointer field will be filled in
// the values obtained by the query.
// If not found, "sql.ErrNoRows" will be returned.
func (q *Qbs) Find(structPtr interface{}) error {
	defer q.Reset()
	q.criteria.model = structPtrToModel(structPtr, !q.criteria.omitJoin, q.criteria.omitFields)
	q.criteria.limit = 1
	if !q.criteria.model.pkZero() {
		idPath := q.Dialect.quote(q.criteria.model.table) + "." + q.Dialect.quote(q.criteria.model.pk.name)
		idCondition := NewCondition(idPath+" = ?", q.criteria.model.pk.value)
		if q.criteria.condition == nil {
			q.criteria.condition = idCondition
		} else {
			q.criteria.condition = idCondition.AndCondition(q.criteria.condition)
		}
	}
	query, args := q.Dialect.querySql(q.criteria)
	return q.doQueryRow(structPtr, query, args...)
}

// Similar to Find, except that FindAll accept pointer of slice of struct pointer,
// rows will be appended to the slice.
func (q *Qbs) FindAll(ptrOfSliceOfStructPtr interface{}) error {
	defer q.Reset()
	strucType := reflect.TypeOf(ptrOfSliceOfStructPtr).Elem().Elem().Elem()
	strucPtr := reflect.New(strucType).Interface()
	q.criteria.model = structPtrToModel(strucPtr, !q.criteria.omitJoin, q.criteria.omitFields)
	query, args := q.Dialect.querySql(q.criteria)
	return q.doQueryRows(ptrOfSliceOfStructPtr, query, args...)
}

func (q *Qbs) LoadM2m(structPtr interface{}) error {
	defer q.Reset()
	q.criteria.m2mUseCond = true
	q.criteria.model = structPtrToModel(structPtr, !q.criteria.omitJoin, q.criteria.omitFields)
	if q.criteria.model.pkZero() {
		return errors.New("pk must be specified!")
	}
	rowValue := reflect.ValueOf(structPtr)
	if err := q.loadM2m(rowValue); err != nil {
		return err
	}
	return nil
}

func (q *Qbs) loadM2m(rowValue reflect.Value) error {
	for fieldname, m2m := range q.criteria.model.m2m {
		if q.criteria.omitJoin {
			continue
		}
		omit := true
		// call LoadM2mFields("m2mfieldname", "field2"...), otherwise omit loading
		for _, loadField := range q.criteria.loadM2m {
			if loadField == fieldname {
				omit = false
				break
			}
		}
		if omit {
			continue
		}
		pkValue := rowValue.Elem().FieldByName(q.criteria.model.pk.camelName).Interface()
		mquery, margs := q.Dialect.queryM2m(q.criteria, fieldname, pkValue)
		m2mfield := rowValue.Elem().FieldByName(m2m.fieldName)
		q.log("load many to many relation:" + q.criteria.model.table + "." + fieldname)
		if err := q.doQueryRows(m2mfield.Addr().Interface(), mquery, margs...); err != nil {
			return err
		}
	}
	return nil
}

func (q *Qbs) doQueryRow(out interface{}, query string, args ...interface{}) error {
	rowValue := reflect.ValueOf(out)
	stmt, err := q.Prepare(query)
	q.log(query, args...)
	if err != nil {
		return q.updateTxError(err)
	}
	rows, err := stmt.Query(args...)
	if err != nil {
		return q.updateTxError(err)
	}
	defer rows.Close()
	if rows.Next() {
		err = q.scanRows(rowValue, rows)
		if err != nil {
			return err
		}
	} else {
		return sql.ErrNoRows
	}
	// load many to many relation values
	if err = q.loadM2m(rowValue); err != nil {
		return err
	}

	return nil
}

func (q *Qbs) doQueryRows(out interface{}, query string, args ...interface{}) error {
	sliceValue := reflect.Indirect(reflect.ValueOf(out))
	sliceType := sliceValue.Type().Elem().Elem()
	q.log(query, args...)
	stmt, err := q.Prepare(query)
	if err != nil {
		return q.updateTxError(err)
	}

	rows, err := stmt.Query(args...)
	if err != nil {
		return q.updateTxError(err)
	}
	defer rows.Close()
	for rows.Next() {
		rowValue := reflect.New(sliceType)
		err = q.scanRows(rowValue, rows)
		if err != nil {
			return err
		}
		sliceValue.Set(reflect.Append(sliceValue, rowValue))
	}
	// if is root query
	if toSnake(sliceType.Name()) == q.criteria.model.table {
		for i := 0; i < sliceValue.Len(); i += 1 {
			if err = q.loadM2m(sliceValue.Index(i)); err != nil {
				return err
			}
		}
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
		if !value.Elem().IsValid() {
			continue
		}
		key := cols[i]
		paths := strings.Split(key, "___")
		if len(paths) == 2 {
			subStruct := rowValue.Elem().FieldByName(snakeToUpperCamel(paths[0]))
			if subStruct.IsNil() {
				subStruct.Set(reflect.New(subStruct.Type().Elem()))
			}
			subField := subStruct.Elem().FieldByName(snakeToUpperCamel(paths[1]))
			if subField.IsValid() {
				err = q.Dialect.setModelValue(value, subField)
				if err != nil {
					return
				}
			}
		} else {
			field := rowValue.Elem().FieldByName(snakeToUpperCamel(key))
			if field.IsValid() {
				err = q.Dialect.setModelValue(value, field)
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
	query = q.Dialect.substituteMarkers(query)
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
	query = q.Dialect.substituteMarkers(query)
	if q.Tx != nil {
		return q.Tx.QueryRow(query, args...)
	}
	return q.Db.QueryRow(query, args...)
}

// Same as sql.Db.Query or sql.Tx.Query depends on if transaction has began
func (q *Qbs) Query(query string, args ...interface{}) (rows *sql.Rows, err error) {
	q.log(query, args...)
	query = q.Dialect.substituteMarkers(query)
	if q.Tx != nil {
		rows, err = q.Tx.Query(query, args...)
	} else {
		rows, err = q.Db.Query(query, args...)
	}
	q.updateTxError(err)
	return
}

// Same as sql.Db.Prepare or sql.Tx.Prepare depends on if transaction has began
func (q *Qbs) Prepare(query string) (stmt *sql.Stmt, err error) {
	if q.Tx != nil {
		stmt, err = q.Tx.Prepare(query + ";")
	} else {
		stmt, err = q.Db.Prepare(query + ";")
	}
	q.updateTxError(err)
	return
}

// If Id value is not provided, save will insert the record, and the Id value will
// be filled in the struct after insertion.
// If Id value is provided, save will do a query count first to see if the row exists, if not then insert it,
// otherwise update it.
// If struct implements Validator interface, it will be validated first
func (q *Qbs) Save(structPtr interface{}) (affected int64, err error) {
	if v, ok := structPtr.(Validator); ok {
		err = v.Validate(q)
		if err != nil {
			return
		}
	}
	model := structPtrToModel(structPtr, true, q.criteria.omitFields)
	if model.pk == nil {
		panic("no primary key field")
	}
	q.criteria.model = model
	now := time.Now()
	var id int64 = 0
	updateModelField := model.timeFiled("updated")
	if updateModelField != nil {
		updateModelField.value = now
	}
	createdModelField := model.timeFiled("created")
	var isInsert bool
	if !model.pkZero() && q.WhereEqual(model.pk.name, model.pk.value).Count(model.table) > 0 { //id is given, can be an update operation.
		affected, err = q.Dialect.update(q)
	} else {
		if createdModelField != nil {
			createdModelField.value = now
		}
		id, err = q.Dialect.insert(q)
		isInsert = true
		if err == nil {
			affected = 1
		}
	}
	if err == nil {
		structValue := reflect.Indirect(reflect.ValueOf(structPtr))
		if _, ok := model.pk.value.(int64); ok && id != 0 {
			idField := structValue.FieldByName(model.pk.camelName)
			idField.SetInt(id)
		}
		if updateModelField != nil {
			updateField := structValue.FieldByName(updateModelField.camelName)
			updateField.Set(reflect.ValueOf(now))
		}
		if isInsert {
			if createdModelField != nil {
				createdField := structValue.FieldByName(createdModelField.camelName)
				createdField.Set(reflect.ValueOf(now))
			}
		}
	}
	return affected, err
}

func (q *Qbs) BulkInsert(sliceOfStructPtr interface{}) error {
	defer q.Reset()
	var err error
	if q.Tx == nil {
		q.Begin()
		defer func() {
			if err != nil {
				q.Rollback()
			} else {
				q.Commit()
			}
		}()
	}
	sliceValue := reflect.ValueOf(sliceOfStructPtr)
	for i := 0; i < sliceValue.Len(); i++ {
		structPtr := sliceValue.Index(i)
		structPtrInter := structPtr.Interface()
		if v, ok := structPtrInter.(Validator); ok {
			err = v.Validate(q)
			if err != nil {
				return err
			}
		}
		model := structPtrToModel(structPtrInter, false, nil)
		if model.pk == nil {
			panic("no primary key field")
		}
		q.criteria.model = model
		var id int64
		id, err = q.Dialect.insert(q)
		if err != nil {
			return err
		}
		if _, ok := model.pk.value.(int64); ok && id != 0 {
			idField := structPtr.Elem().FieldByName(model.pk.camelName)
			idField.SetInt(id)
		}
	}
	return nil
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
	return q.Dialect.update(q)
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
	return q.Dialect.delete(q)
}

// This method can be used to validate unique column before trying to save
// The table parameter can be either a string or a struct pointer
func (q *Qbs) ContainsValue(table interface{}, column string, value interface{}) bool {
	quotedColumn := q.Dialect.quote(column)
	quotedTable := q.Dialect.quote(tableName(table))
	query := fmt.Sprintf("SELECT %v FROM %v WHERE %v = ?", quotedColumn, quotedTable, quotedColumn)
	row := q.QueryRow(query, value)
	var result interface{}
	err := row.Scan(&result)
	q.updateTxError(err)
	return err == nil
}

// It is safe to call it even if *sql.DB is nil.
// So it's better to call "defer q.Close()" right after qbs.New() to release resource.
// If the connection pool is not full, the Db will be sent back into the pool, otherwise the Db will get closed.
func (q *Qbs) Close() error {
	if q.Tx != nil {
		q.Tx.Rollback()
	}
	if q.Db != nil {
		select {
		case connectionPool <- q.Db:
			return nil
		default:
		}
		err := q.Db.Close()
		q.Db = nil
		return err
	}
	return nil
}

//Query the count of rows in a table the talbe parameter can be either a string or struct pointer.
//If condition is given, the count will be the count of rows meet that condition.
func (q *Qbs) Count(table interface{}) int64 {
	quotedTable := q.Dialect.quote(tableName(table))
	query := "SELECT COUNT(*) FROM " + quotedTable
	var row *sql.Row
	if q.criteria.condition != nil {
		conditionSql, args := q.criteria.condition.Merge()
		query += " WHERE " + conditionSql
		row = q.QueryRow(query, args...)
	} else {
		row = q.QueryRow(query)
	}
	var count int64
	err := row.Scan(&count)
	if err == sql.ErrNoRows {
		return 0
	} else if err != nil {
		q.updateTxError(err)
	}
	return count
}

//Query raw sql and return a map.
func (q *Qbs) QueryMap(query string, args ...interface{}) (map[string]interface{}, error) {
	mapSlice, err := q.doQueryMap(query, true, args...)
	if len(mapSlice) == 1 {
		return mapSlice[0], err
	}
	return nil, sql.ErrNoRows

}

//Query raw sql and return a slice of map..
func (q *Qbs) QueryMapSlice(query string, args ...interface{}) ([]map[string]interface{}, error) {
	return q.doQueryMap(query, false, args...)
}

func (q *Qbs) doQueryMap(query string, once bool, args ...interface{}) ([]map[string]interface{}, error) {
	query = q.Dialect.substituteMarkers(query)
	stmt, err := q.Prepare(query)
	if err != nil {
		return nil, q.updateTxError(err)
	}
	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, q.updateTxError(err)
	}
	defer rows.Close()
	var results []map[string]interface{}
	columns, _ := rows.Columns()
	containers := make([]interface{}, len(columns))
	for i := 0; i < len(columns); i++ {
		var container interface{}
		containers[i] = &container
	}
	for rows.Next() {
		if err := rows.Scan(containers...); err != nil {
			return nil, q.updateTxError(err)
		}
		result := make(map[string]interface{}, len(columns))
		for i, key := range columns {
			if containers[i] == nil {
				continue
			}
			value := reflect.Indirect(reflect.ValueOf(containers[i]))
			if value.Elem().Kind() == reflect.Slice {
				result[key] = string(value.Interface().([]byte))
			} else {
				result[key] = value.Interface()
			}
		}
		results = append(results, result)
		if once {
			return results, nil
		}
	}
	return results, nil
}

func (q *Qbs) log(query string, args ...interface{}) {
	if q.Log {
		fmt.Println(query)
		if len(args) > 0 {
			fmt.Println(args...)
		}
	}
}
