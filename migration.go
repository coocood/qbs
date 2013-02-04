package qbs

import (
	"database/sql"
)

type Migration struct {
	Db      *sql.DB
	DbName  string
	Dialect Dialect
}

// CreateTableIfNotExists creates a new table and its indexes based on the table struct type
// It will panic if table creation failed, and it will return error if the index creation failed.
func (mg *Migration) CreateTableIfNotExists(structPtr interface{}) error {
	model := structPtrToModel(structPtr, true)
	_, err := mg.Db.Exec(mg.Dialect.CreateTableSql(model, true))
	if err != nil {
		panic(err)
	}
	columns := mg.Dialect.ColumnsInTable(mg, model.Table)
	if len(model.Fields) > len(columns) {
		oldFields := []*ModelField{}
		newFields := []*ModelField{}
		for _, v := range model.Fields {
			if _, ok := columns[v.Name]; ok {
				oldFields = append(oldFields, v)
			} else {
				newFields = append(newFields, v)
			}
		}
		if len(oldFields) != len(columns) {
			panic("Column name has changed, rename column migration is not supported.")
		}
		for _, v := range newFields {
			mg.addColumn(model.Table, v)
		}
	}
	var indexErr error
	for _, i := range model.Indexes {
		indexErr = mg.CreateIndexIfNotExists(model.Table, i.Name, i.Unique, i.Columns...)
	}
	return indexErr
}

// this is only used for testing.
func (mg *Migration) dropTableIfExists(structPtr interface{}) {
	tn := tableName(structPtr)
	_, err := mg.Db.Exec(mg.Dialect.DropTableSql(tn))
	if err != nil {
		panic(err)
	}
}

func (mg *Migration) addColumn(table string, column *ModelField) {
	_, err := mg.Db.Exec(mg.Dialect.AddColumnSql(table, column.Name, column.Value, column.Size()))
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
	if !mg.Dialect.IndexExists(mg, tn, name) {
		_, err := mg.Db.Exec(mg.Dialect.CreateIndexSql(name, tn, unique, columns...))
		return err
	}
	return nil
}

// Migration only support incremental migrations like create table if not exists
// create index if not exists, add columns, so it's safe to keep it in production environment.
func NewMigration(db *sql.DB, dbName string, dialect Dialect) *Migration {
	return &Migration{db, dbName, dialect}
}
