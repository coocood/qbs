package qbs

import (
	"database/sql"
	"fmt"
	"strings"
)

type Migration struct {
	Db      *sql.DB
	DbName  string
	Dialect Dialect
	Log     bool
}

// CreateTableIfNotExists creates a new table and its indexes based on the table struct type
// It will panic if table creation failed, and it will return error if the index creation failed.
func (mg *Migration) CreateTableIfNotExists(structPtr interface{}) error {
	model := structPtrToModel(structPtr, true, nil)
	sql := mg.Dialect.createTableSql(model, true)
	if mg.Log {
		fmt.Println(sql)
	}
	sqls := strings.Split(sql, ";")
	for _, v := range sqls {
		_, err := mg.Db.Exec(v)
		if err != nil && !mg.Dialect.catchMigrationError(err) {
			panic(err)
		}
	}
	columns := mg.Dialect.columnsInTable(mg, model.table)
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
	_, err := mg.Db.Exec(mg.Dialect.dropTableSql(tn))
	if err != nil && !mg.Dialect.catchMigrationError(err) {
		panic(err)
	}
}

//Can only drop table on database which name has "test" suffix.
//Used for testing
func (mg *Migration) DropTable(strutPtr interface{}) {
	if !strings.HasSuffix(mg.DbName, "test") {
		panic("Drop table can only be executed on database which name has 'test' suffix")
	}
	mg.dropTableIfExists(strutPtr)
}

func (mg *Migration) addColumn(table string, column *modelField) {
	sql := mg.Dialect.addColumnSql(table, column.name, column.value, column.size())
	if mg.Log {
		fmt.Println(sql)
	}
	_, err := mg.Db.Exec(sql)
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
	if !mg.Dialect.indexExists(mg, tn, name) {
		sql := mg.Dialect.createIndexSql(name, tn, unique, columns...)
		if mg.Log {
			fmt.Println(sql)
		}
		_, err := mg.Db.Exec(sql)
		return err
	}
	return nil
}

func (mg *Migration) Close() {
	if mg.Db != nil {
		err := mg.Db.Close()
		if err != nil {
			panic(err)
		}
	}
}

// Deprecated, call Register and GetMigration instead.
// Migration only support incremental migrations like create table if not exists
// create index if not exists, add columns, so it's safe to keep it in production environment.
func NewMigration(db *sql.DB, dbName string, dialect Dialect) *Migration {
	return &Migration{db, dbName, dialect, false}
}

// Get a Migration instance should get closed like Qbs instance.
func GetMigration() (mg *Migration, err error) {
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
