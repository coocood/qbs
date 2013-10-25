package qbs

import (
	"database/sql"
	"fmt"
	"strings"
)

type Migration struct {
	db      *sql.DB
	dbName  string
	dialect Dialect
	Log     bool
}

// CreateTableIfNotExists creates a new table and its indexes based on the table struct type
// It will panic if table creation failed, and it will return error if the index creation failed.
func (mg *Migration) CreateTableIfNotExists(structPtr interface{}) error {
	model := structPtrToModel(structPtr, true, nil)
	sql := mg.dialect.createTableSql(model, true)
	if mg.Log {
		fmt.Println(sql)
	}
	sqls := strings.Split(sql, ";")
	for _, v := range sqls {
		_, err := mg.db.Exec(v)
		if err != nil && !mg.dialect.catchMigrationError(err) {
			panic(err)
		}
	}
	columns := mg.dialect.columnsInTable(mg, model.table)
	if len(model.fields) > len(columns) {
		oldFields := []*modelField{}
		newFields := []*modelField{}
		for _, v := range model.fields {
			if _, ok := columns[v.name]; ok {
				oldFields = append(oldFields, v)
			} else {
				newFields = append(newFields, v)
			}
		}
		if len(oldFields) != len(columns) {
			panic("Column name has changed, rename column migration is not supported.")
		}
		for _, v := range newFields {
			mg.addColumn(model.table, v)
		}
	}
	var indexErr error
	for _, i := range model.indexes {
		indexErr = mg.CreateIndexIfNotExists(model.table, i.name, i.unique, i.columns...)
	}
	return indexErr
}

// this is only used for testing.
func (mg *Migration) dropTableIfExists(structPtr interface{}) {
	tn := tableName(structPtr)
	_, err := mg.db.Exec(mg.dialect.dropTableSql(tn))
	if err != nil && !mg.dialect.catchMigrationError(err) {
		panic(err)
	}
}

//Can only drop table on database which name has "test" suffix.
//Used for testing
func (mg *Migration) DropTable(strutPtr interface{}) {
	if !strings.HasSuffix(mg.dbName, "test") {
		panic("Drop table can only be executed on database which name has 'test' suffix")
	}
	mg.dropTableIfExists(strutPtr)
}

func (mg *Migration) addColumn(table string, column *modelField) {
	sql := mg.dialect.addColumnSql(table, *column)
	if mg.Log {
		fmt.Println(sql)
	}
	_, err := mg.db.Exec(sql)
	if err != nil {
		panic(err)
	}
}

// CreateIndex creates the specified index on table.
// Some databases like mysql do not support this feature directly,
// So dialect may need to query the database schema table to find out if an index exists.
// Normally you don't need to do it explicitly, it will be created automatically in CreateTableIfNotExists method.
func (mg *Migration) CreateIndexIfNotExists(table interface{}, name string, unique bool, columns ...string) error {
	tn := tableName(table)
	name = tn + "_" + name
	if !mg.dialect.indexExists(mg, tn, name) {
		sql := mg.dialect.createIndexSql(name, tn, unique, columns...)
		if mg.Log {
			fmt.Println(sql)
		}
		_, err := mg.db.Exec(sql)
		return err
	}
	return nil
}

func (mg *Migration) Close() {
	if mg.db != nil {
		err := mg.db.Close()
		if err != nil {
			panic(err)
		}
	}
}

// Get a Migration instance should get closed like Qbs instance.
func GetMigration() (mg *Migration, err error) {
	if driver == "" || dial == nil {
		panic("database driver has not been registered, should call Register first.")
	}
	db, err := sql.Open(driver, driverSource)
	if err != nil {
		return nil, err
	}
	return &Migration{db, dbName, dial, false}, nil
}

// A safe and easy way to work with Migration instance without the need to open and close it.
func WithMigration(task func(mg *Migration) error) error {
	mg, err := GetMigration()
	if err != nil {
		return err
	}
	defer mg.Close()
	return task(mg)
}
