package qbs

import (
	"testing"
	//"time"

	_ "github.com/mattn/go-sqlite3"
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
	"ALTER TABLE `a` ADD COLUMN `newc` text",
	"CREATE UNIQUE INDEX `iname` ON `itable` (`a`, `b`, `c`)",
	"CREATE INDEX `iname2` ON `itable2` (`d`, `e`)",
}

func registerSqlite3Test() {
	RegisterSqlite3("/tmp/foo.db")
}

func setupSqlite3Db() (*Migration, *Qbs) {
	registerSqlite3Test()
	mg, _ := GetMigration()
	q, _ := GetQbs()
	return mg, q
}

var sqlite3SqlTypeResults []string = []string{
	"integer",
	"integer",
	"integer",
	"integer",
	"integer",
	"integer",
	"integer",
	"integer",
	"integer",
	"integer",
	"integer",
	"real",
	"real",
	"text",
	"text",
	"text",
	"text",
	"integer",
	"integer",
	"integer",
	"real",
	"text",
	"text",
	"text",
}

func TestSqlite3SqlType(t *testing.T) {
	assert := NewAssert(t)
	d := NewSqlite3()
	testModel := structPtrToModel(new(typeTestTable), false, nil)
	for index, column := range testModel.fields {
		if storedResult := sqlite3SqlTypeResults[index]; storedResult != "-" {
			result := d.sqlType(*column)
			assert.Equal(storedResult, result)
		}
	}
	/*for _, column := range testModel.fields {
		result := d.sqlType(*column)

		switch column.camelName {
		case "Bool":
			assert.Equal("integer", result)

		case "Int8":
			assert.Equal("integer", result)
		case "Int16":
			assert.Equal("integer", result)
		case "Int32":
			assert.Equal("integer", result)
		case "UInt8":
			assert.Equal("integer", result)
		case "UInt16":
			assert.Equal("integer", result)
		case "UInt32":
			assert.Equal("integer", result)

		case "Int":
			assert.Equal("integer", result)
		case "UInt":
			assert.Equal("integer", result)
		case "Int64":
			assert.Equal("integer", result)
		case "UInt64":
			assert.Equal("integer", result)

		case "Float32":
			assert.Equal("real", result)
		case "Float64":
			assert.Equal("real", result)

		case "Varchar":
			assert.Equal("text", result)
		case "LongText":
			assert.Equal("text", result)

		case "Time":
			assert.Equal("text", result)

		case "Slice":
			assert.Equal("text", result)

		case "DerivedInt":
			assert.Equal("integer", result)
		case "DerivedInt16":
			assert.Equal("integer", result)
		case "DerivedBool":
			assert.Equal("integer", result)
		case "DerivedFloat":
			assert.Equal("real", result)
		case "DerivedTime":
			assert.Equal("text", result)
		}
	}*/
}

func TestSqlite3Transaction(t *testing.T) {
	registerSqlite3Test()
	doTestTransaction(NewAssert(t))
}

func TestSqlite3SaveAndDelete(t *testing.T) {
	mg, q := setupSqlite3Db()
	doTestSaveAndDelete(NewAssert(t), mg, q)
}

func TestSqlite3SaveAgain(t *testing.T) {
	mg, q := setupSqlite3Db()
	doTestSaveAgain(NewAssert(t), mg, q)
}

func TestSqlite3ForeignKey(t *testing.T) {
	registerSqlite3Test()
	doTestForeignKey(NewAssert(t))
}

func TestSqlite3Find(t *testing.T) {
	registerSqlite3Test()
	doTestFind(NewAssert(t))
}

func TestSqlite3CreateTable(t *testing.T) {
	mg, _ := setupSqlite3Db()
	doTestCreateTable(NewAssert(t), mg)
}

func TestSqlite3Update(t *testing.T) {
	mg, q := setupSqlite3Db()
	doTestUpdate(NewAssert(t), mg, q)
}

func TestSqlite3Validation(t *testing.T) {
	mg, q := setupSqlite3Db()
	doTestValidation(NewAssert(t), mg, q)
}

func TestSqlite3BoolType(t *testing.T) {
	mg, q := setupSqlite3Db()
	doTestBoolType(NewAssert(t), mg, q)
}

func TestSqlite3StringPk(t *testing.T) {
	mg, q := setupSqlite3Db()
	doTestStringPk(NewAssert(t), mg, q)
}

func TestSqlite3Count(t *testing.T) {
	registerSqlite3Test()
	doTestCount(NewAssert(t))
}

func TestSqlite3QueryMap(t *testing.T) {
	mg, q := setupSqlite3Db()
	doTestQueryMap(NewAssert(t), mg, q)
}

func TestSqlite3BulkInsert(t *testing.T) {
	registerSqlite3Test()
	doTestBulkInsert(NewAssert(t))
}

func TestSqlite3QueryStruct(t *testing.T) {
	registerSqlite3Test()
	doTestQueryStruct(NewAssert(t))
}

func TestSqlite3CustomNameConvertion(t *testing.T) {
	registerSqlite3Test()
	ColumnNameToFieldName = noConvert
	FieldNameToColumnName = noConvert
	TableNameToStructName = noConvert
	StructNameToTableName = noConvert
	doTestForeignKey(NewAssert(t))
	ColumnNameToFieldName = snakeToUpperCamel
	FieldNameToColumnName = toSnake
	TableNameToStructName = snakeToUpperCamel
	StructNameToTableName = toSnake
}

func TestSqlite3ConnectionLimit(t *testing.T) {
	registerSqlite3Test()
	doTestConnectionLimit(NewAssert(t))
}

func TestSqlite3Iterate(t *testing.T) {
	registerSqlite3Test()
	doTestIterate(NewAssert(t))
}

func TestSqlite3AddColumnSQL(t *testing.T) {
	doTestAddColumSQL(NewAssert(t), sqlite3Syntax)
}

func TestSqlite3CreateTableSQL(t *testing.T) {
	doTestCreateTableSQL(NewAssert(t), sqlite3Syntax)
}

func TestSqlite3CreateIndexSQL(t *testing.T) {
	doTestCreateIndexSQL(NewAssert(t), sqlite3Syntax)
}

func TestSqlite3InsertSQL(t *testing.T) {
	doTestInsertSQL(NewAssert(t), sqlite3Syntax)
}

func TestSqlite3UpdateSQL(t *testing.T) {
	doTestUpdateSQL(NewAssert(t), sqlite3Syntax)
}

func TestSqlite3DeleteSQL(t *testing.T) {
	doTestDeleteSQL(NewAssert(t), sqlite3Syntax)
}

func TestSqlite3SelectionSQL(t *testing.T) {
	doTestSelectionSQL(NewAssert(t), sqlite3Syntax)
}

func TestSqlite3QuerySQL(t *testing.T) {
	doTestQuerySQL(NewAssert(t), sqlite3Syntax)
}

func TestSqlite3DropTableSQL(t *testing.T) {
	doTestDropTableSQL(NewAssert(t), sqlite3Syntax)
}

func TestSqlite3SaveNullable(t *testing.T) {
	mg, q := setupSqlite3Db()
	doTestSaveNullable(NewAssert(t), mg, q)
}

func BenchmarkSqlite3Find(b *testing.B) {
	registerSqlite3Test()
	doBenchmarkFind(b, b.N)
}

func BenchmarkSqlite3DbQuery(b *testing.B) {
	registerSqlite3Test()
	doBenchmarkDbQuery(b, b.N)
}

func BenchmarkSqlite3StmtQuery(b *testing.B) {
	registerSqlite3Test()
	doBenchmarkStmtQuery(b, b.N)
}

func BenchmarkSqlite3Transaction(b *testing.B) {
	registerSqlite3Test()
	doBenchmarkTransaction(b, b.N)
}
