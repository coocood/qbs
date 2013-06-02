package qbs

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	sqlite3Driver = "sqlite3"
)

var sqlite3Syntax = dialectSyntax{
	NewSqlite3(),
	"CREATE TABLE IF NOT EXISTS `without_pk` ( `first` text, `last` text, `amount` integer )",
	"CREATE TABLE `with_pk` ( `primary` integer PRIMARY KEY AUTOINCREMENT NOT NULL, `first` text, `last` text, `amount` integer )",
	"INSERT INTO `sql_gen_model` (`prim`, `first`, `last`, `amount`) VALUES (?, ?, ?, ?)",
	"UPDATE `sql_gen_model` SET `first` = ?, `last` = ?, `amount` = ? WHERE `prim` = ?",
	"DELETE FROM `sql_gen_model` WHERE `prim` = ?",
	"SELECT `post`.`id`, `post`.`author_id`, `post`.`content`, `author`.`id` AS author___id, `author`.`name` AS author___name FROM `post` LEFT JOIN `user` AS `author` ON `post`.`author_id` = `author`.`id`",
	"SELECT `name`, `grade`, `score` FROM `student` WHERE (grade IN (?, ?, ?)) AND ((score <= ?) OR (score >= ?)) ORDER BY `name`, `grade` DESC LIMIT ? OFFSET ?",
	"DROP TABLE IF EXISTS `drop_table`",
	"ALTER TABLE `a` ADD COLUMN `c` text",
	"CREATE UNIQUE INDEX `iname` ON `itable` (`a`, `b`, `c`)",
	"CREATE INDEX `iname2` ON `itable2` (`d`, `e`)",
}

func openSqlite3Db() (*sql.DB, error) {
	return sql.Open(sqlite3Driver, "/tmp/foo.db")
}
func registerSqlite3Test() {
	//	os.Remove("/tmp/foo.db")
	Register(sqlite3Driver, "/tmp/foo.db", testDbName, NewSqlite3())
}

func setupSqlite3Db() (*Migration, *Qbs) {
	registerSqlite3Test()
	mg, _ := GetMigration()
	q, _ := GetQbs()
	return mg, q
}

func TestSqlite3SqlType(t *testing.T) {
	assert := NewAssert(t)
	d := NewSqlite3()
	assert.Equal("integer", d.sqlType(true, 0))
	var indirect interface{} = true
	assert.Equal("integer", d.sqlType(indirect, 0))
	assert.Equal("integer", d.sqlType(uint32(2), 0))
	assert.Equal("integer", d.sqlType(int64(1), 0))
	assert.Equal("real", d.sqlType(1.8, 0))
	assert.Equal("text", d.sqlType([]byte("asdf"), 0))
	assert.Equal("text", d.sqlType("astring", 0))
	assert.Equal("text", d.sqlType("a", 65536))
	assert.Equal("text", d.sqlType("b", 128))
	assert.Equal("text", d.sqlType(time.Now(), 0))
}

func TestSqlite3Transaction(t *testing.T) {
	registerSqlite3Test()
	doTestTransaction(t)
}

func TestSqlite3SaveAndDelete(t *testing.T) {
	mg, q := setupSqlite3Db()
	doTestSaveAndDelete(t, mg, q)
}

func TestSqlite3SaveAgain(t *testing.T) {
	mg, q := setupSqlite3Db()
	doTestSaveAgain(t, mg, q)
}

func TestSqlite3ForeignKey(t *testing.T) {
	registerSqlite3Test()
	doTestForeignKey(t)
}

func TestSqlite3Find(t *testing.T) {
	registerSqlite3Test()
	doTestFind(t)
}

func TestSqlite3CreateTable(t *testing.T) {
	mg, _ := setupSqlite3Db()
	doTestCreateTable(t, mg)
}

func TestSqlite3Update(t *testing.T) {
	mg, q := setupSqlite3Db()
	doTestUpdate(t, mg, q)
}

func TestSqlite3Validation(t *testing.T) {
	mg, q := setupSqlite3Db()
	doTestValidation(t, mg, q)
}

func TestSqlite3BoolType(t *testing.T) {
	mg, q := setupSqlite3Db()
	doTestBoolType(t, mg, q)
}

func TestSqlite3StringPk(t *testing.T) {
	mg, q := setupSqlite3Db()
	doTestStringPk(t, mg, q)
}

func TestSqlite3Count(t *testing.T) {
	registerSqlite3Test()
	doTestCount(t)
}

func TestSqlite3QueryMap(t *testing.T) {
	mg, q := setupSqlite3Db()
	doTestQueryMap(t, mg, q)
}

func TestSqlite3BulkInsert(t *testing.T) {
	registerSqlite3Test()
	doTestBulkInsert(t)
}

func TestSqlite3QueryStruct(t *testing.T) {
	registerSqlite3Test()
	doTestQueryStruct(t)
}

func TestSqlite3CustomNameConvertion(t *testing.T) {
	registerSqlite3Test()
	ColumnNameToFieldName = noConvert
	FieldNameToColumnName = noConvert
	TableNameToStructName = noConvert
	StructNameToTableName = noConvert
	doTestForeignKey(t)
	ColumnNameToFieldName = snakeToUpperCamel
	FieldNameToColumnName = toSnake
	TableNameToStructName = snakeToUpperCamel
	StructNameToTableName = toSnake
}

func TestSqlite3ConnectionLimit(t *testing.T) {
	registerSqlite3Test()
	doTestConnectionLimit(t)
}

func TestSqlite3Iterate(t *testing.T) {
	registerSqlite3Test()
	doTestIterate(t)
}

func TestSqlite3AddColumnSQL(t *testing.T) {
	doTestAddColumSQL(t, sqlite3Syntax)
}

func TestSqlite3CreateTableSQL(t *testing.T) {
	doTestCreateTableSQL(t, sqlite3Syntax)
}

func TestSqlite3CreateIndexSQL(t *testing.T) {
	doTestCreateIndexSQL(t, sqlite3Syntax)
}

func TestSqlite3InsertSQL(t *testing.T) {
	doTestInsertSQL(t, sqlite3Syntax)
}

func TestSqlite3UpdateSQL(t *testing.T) {
	doTestUpdateSQL(t, sqlite3Syntax)
}

func TestSqlite3DeleteSQL(t *testing.T) {
	doTestDeleteSQL(t, sqlite3Syntax)
}

func TestSqlite3SelectionSQL(t *testing.T) {
	doTestSelectionSQL(t, sqlite3Syntax)
}

func TestSqlite3QuerySQL(t *testing.T) {
	doTestQuerySQL(t, sqlite3Syntax)
}

func TestSqlite3DropTableSQL(t *testing.T) {
	doTestDropTableSQL(t, sqlite3Syntax)
}

func BenchmarkSqlite3Find(b *testing.B) {
	registerSqlite3Test()
	doBenchmarkFind(b)
}

func BenchmarkSqlite3DbQuery(b *testing.B) {
	registerSqlite3Test()
	doBenchmarkDbQuery(b)
}

func BenchmarkSqlite3StmtQuery(b *testing.B) {
	registerSqlite3Test()
	doBenchmarkStmtQuery(b)
}

func BenchmarkSqlite3Transaction(b *testing.B) {
	registerSqlite3Test()
	doBenchmarkTransaction(b)
}
