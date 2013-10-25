package qbs

import (
	_ "github.com/coocood/mysql"
	"testing"
	"time"
)

var mysqlSyntax = dialectSyntax{
	NewMysql(),
	"CREATE TABLE IF NOT EXISTS `without_pk` ( `first` longtext, `last` longtext, `amount` bigint )",
	"CREATE TABLE `with_pk` ( `primary` bigint PRIMARY KEY AUTO_INCREMENT, `first` longtext, `last` longtext, `amount` bigint )",
	"INSERT INTO `sql_gen_model` (`prim`, `first`, `last`, `amount`) VALUES (?, ?, ?, ?)",
	"UPDATE `sql_gen_model` SET `first` = ?, `last` = ?, `amount` = ? WHERE `prim` = ?",
	"DELETE FROM `sql_gen_model` WHERE `prim` = ?",
	"SELECT `post`.`id`, `post`.`author_id`, `post`.`content`, `author`.`id` AS author___id, `author`.`name` AS author___name FROM `post` LEFT JOIN `user` AS `author` ON `post`.`author_id` = `author`.`id`",
	"SELECT `name`, `grade`, `score` FROM `student` WHERE (grade IN (?, ?, ?)) AND ((score <= ?) OR (score >= ?)) ORDER BY `name`, `grade` DESC LIMIT ? OFFSET ?",
	"DROP TABLE IF EXISTS `drop_table`",
	"ALTER TABLE `a` ADD COLUMN `newc` varchar(100)",
	"CREATE UNIQUE INDEX `iname` ON `itable` (`a`, `b`, `c`)",
	"CREATE INDEX `iname2` ON `itable2` (`d`, `e`)",
}

func setupMysqlDb() (*Migration, *Qbs) {
	registerMysqlTest()
	mg, _ := GetMigration()
	q, _ := GetQbs()
	return mg, q
}

func registerMysqlTest() {
	dsn := new(DataSourceName)
	dsn.DbName = testDbName
	dsn.Username = "root"
	dsn.Dialect = NewMysql()
	dsn.Append("parseTime", "true").Append("loc", "Local")
	RegisterWithDataSourceName(dsn)
}
			
type FakeInt    int
type FakeInt16	int16
type FakeBool	bool
type FakeFloat	float32
type FakeTime	time.Time
type FakeString	string

type typeTestTable struct {
	Bool		bool		`qbs:""`
	
	Int8 	    int8		`qbs:""`
	Int16   	int16		`qbs:""`
	Int32   	int32		`qbs:""`
	UInt8		uint8		`qbs:""`
	UInt16		uint16		`qbs:""`
	UInt32		uint32		`qbs:""`
	
	Int 	    int		    `qbs:""`
	UInt 	    uint	    `qbs:""`
	Int64	 	int64		`qbs:""`
	UInt64	 	uint64		`qbs:""`
	
	Float32		float32
	Float64		float64
	
	Varchar 	string		`qbs:"size:128"`
	LongText 	string		`qbs:"size:65536"`
	
	Time    	time.Time
	
	Slice		[]byte
	
	DerivedInt		FakeInt			`qbs:"coltype:int"`
	DerivedInt16	FakeInt16		`qbs:"coltype:bigint"`
	DerivedBool		FakeBool		`qbs:"coltype:boolean"`
	DerivedFloat	FakeFloat		`qbs:"coltype:double"`
	DerivedTime		FakeTime		`qbs:"coltype:timestamp"`
	DerivedVarChar	FakeTime		`qbs:"coltype:text,size:128"`
	DerivedLongText	FakeTime		`qbs:"coltype:text,size:65536"`
}

var mysqlSqlTypeResults []string = []string{
	"boolean",
	"int",
	"int",
	"int",
	"int",
	"int",
	"int",
	"bigint",
	"bigint",
	"bigint",
	"bigint",
	"double",
	"double",
	"varchar(128)",
	"longtext",
	"timestamp",
	"longblob",
	"bigint",
	"int",
	"boolean",
	"double",
	"timestamp",
	"varchar(128)",
	"longtext",
}

func TestMysqlSqlType(t *testing.T) {
	assert := NewAssert(t)
	
	d := NewMysql()
	testModel := structPtrToModel(new(typeTestTable), false, nil)
	for index, column := range testModel.fields {
		if storedResult := mysqlSqlTypeResults[index]; storedResult != "-" {
			result := d.sqlType(*column)
			assert.Equal(storedResult, result)
		}
	}
}

func TestMysqlTransaction(t *testing.T) {
	registerMysqlTest()
	doTestTransaction(t)
}

func TestMysqlSaveAndDelete(t *testing.T) {
	mg, q := setupMysqlDb()
	doTestSaveAndDelete(t, mg, q)
}

func TestMysqlSaveAgain(t *testing.T) {
	mg, q := setupMysqlDb()
	doTestSaveAgain(t, mg, q)
}

func TestMysqlForeignKey(t *testing.T) {
	registerMysqlTest()
	doTestForeignKey(t)
}

func TestMysqlFind(t *testing.T) {
	registerMysqlTest()
	doTestFind(t)
}

func TestMysqlCreateTable(t *testing.T) {
	mg, _ := setupMysqlDb()
	doTestCreateTable(t, mg)
}

func TestMysqlUpdate(t *testing.T) {
	mg, q := setupMysqlDb()
	doTestUpdate(t, mg, q)
}

func TestMysqlValidation(t *testing.T) {
	mg, q := setupMysqlDb()
	doTestValidation(t, mg, q)
}

func TestMysqlBoolType(t *testing.T) {
	mg, q := setupMysqlDb()
	doTestBoolType(t, mg, q)
}

func TestMysqlStringPk(t *testing.T) {
	mg, q := setupMysqlDb()
	doTestStringPk(t, mg, q)
}

func TestMysqlCount(t *testing.T) {
	registerMysqlTest()
	doTestCount(t)
}

func TestMysqlQueryMap(t *testing.T) {
	mg, q := setupMysqlDb()
	doTestQueryMap(t, mg, q)
}

func TestMysqlBulkInsert(t *testing.T) {
	registerMysqlTest()
	doTestBulkInsert(t)
}

func TestMysqlQueryStruct(t *testing.T) {
	registerMysqlTest()
	doTestQueryStruct(t)
}

func TestMysqlCustomNameConvertion(t *testing.T) {
	registerMysqlTest()
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

func TestMysqlConnectionLimit(t *testing.T) {
	registerMysqlTest()
	doTestConnectionLimit(t)
}

func TestMysqlIterate(t *testing.T) {
	registerMysqlTest()
	doTestIterate(t)
}

func TestMysqlAddColumnSQL(t *testing.T) {
	doTestAddColumSQL(t, mysqlSyntax)
}

func TestMysqlCreateTableSQL(t *testing.T) {
	doTestCreateTableSQL(t, mysqlSyntax)
}

func TestMysqlCreateIndexSQL(t *testing.T) {
	doTestCreateIndexSQL(t, mysqlSyntax)
}

func TestMysqlInsertSQL(t *testing.T) {
	doTestInsertSQL(t, mysqlSyntax)
}

func TestMysqlUpdateSQL(t *testing.T) {
	doTestUpdateSQL(t, mysqlSyntax)
}

func TestMysqlDeleteSQL(t *testing.T) {
	doTestDeleteSQL(t, mysqlSyntax)
}

func TestMysqlSelectionSQL(t *testing.T) {
	doTestSelectionSQL(t, mysqlSyntax)
}

func TestMysqlQuerySQL(t *testing.T) {
	doTestQuerySQL(t, mysqlSyntax)
}
func TestMysqlDropTableSQL(t *testing.T) {
	doTestDropTableSQL(t, mysqlSyntax)
}

func TestMysqlDataSourceName(t *testing.T) {
	dsn := new(DataSourceName)
	dsn.DbName = "abc"
	dsn.Username = "john"
	dsn.Dialect = NewMysql()
	assert := NewAssert(t)
	assert.Equal("john@/abc", dsn)
	dsn.Password = "123"
	assert.Equal("john:123@/abc", dsn)
	dsn.Host = "192.168.1.3"
	assert.Equal("john:123@tcp(192.168.1.3)/abc", dsn)
	dsn.UnixSocket = true
	assert.Equal("john:123@unix(192.168.1.3)/abc", dsn)
	dsn.Append("charset", "utf8")
	dsn.Append("parseTime", "true")
	assert.Equal("john:123@unix(192.168.1.3)/abc?charset=utf8&parseTime=true", dsn)
	dsn.Port = "3336"
	assert.Equal("john:123@unix(192.168.1.3:3336)/abc?charset=utf8&parseTime=true", dsn)
}

func BenchmarkMysqlFind(b *testing.B) {
	registerMysqlTest()
	doBenchmarkFind(b)
}

func BenchmarkMysqlQueryStruct(b *testing.B) {
	registerMysqlTest()
	doBenchmarkQueryStruct(b)
}

func BenchmarkMysqlDbQuery(b *testing.B) {
	registerMysqlTest()
	doBenchmarkDbQuery(b)
}

func BenchmarkMysqlStmtQuery(b *testing.B) {
	registerMysqlTest()
	doBenchmarkStmtQuery(b)
}

func BenchmarkMysqlTransaction(b *testing.B) {
	registerMysqlTest()
	doBenchmarkTransaction(b)
}
