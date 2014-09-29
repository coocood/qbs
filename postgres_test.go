package qbs

import (
	_ "github.com/lib/pq"
	"testing"
	//"time"
)

var pgSyntax = dialectSyntax{
	NewPostgres(),
	`CREATE TABLE IF NOT EXISTS "without_pk" ( "first" text, "last" text, "amount" bigint )`,
	`CREATE TABLE "with_pk" ( "primary" bigserial PRIMARY KEY, "first" text, "last" text, "amount" bigint )`,
	`INSERT INTO "sql_gen_model" ("prim", "first", "last", "amount") VALUES ($1, $2, $3, $4) RETURNING "prim"`,
	`UPDATE "sql_gen_model" SET "first" = $1, "last" = $2, "amount" = $3 WHERE "prim" = $4`,
	`DELETE FROM "sql_gen_model" WHERE "prim" = $1`,
	`SELECT "post"."id", "post"."author_id", "post"."content", "author"."id" AS author___id, "author"."name" AS author___name FROM "post" LEFT JOIN "user" AS "author" ON "post"."author_id" = "author"."id"`,
	`SELECT "name", "grade", "score" FROM "student" WHERE (grade IN ($1, $2, $3)) AND ((score <= $4) OR (score >= $5)) ORDER BY "name", "grade" DESC LIMIT $6 OFFSET $7`,
	`DROP TABLE IF EXISTS "drop_table"`,
	`ALTER TABLE "a" ADD COLUMN "newc" varchar(100)`,
	`CREATE UNIQUE INDEX "iname" ON "itable" ("a", "b", "c")`,
	`CREATE INDEX "iname2" ON "itable2" ("d", "e")`,
}

func registerPgTest() {
	RegisterWithDataSourceName(DefaultPostgresDataSourceName(testDbName))
}

func setupPgDb() (*Migration, *Qbs) {
	registerPgTest()
	mg, _ := GetMigration()
	q, _ := GetQbs()
	return mg, q
}

var postgresSqlTypeResults []string = []string{
	"boolean",
	"integer",
	"integer",
	"integer",
	"integer",
	"integer",
	"integer",
	"bigint",
	"bigint",
	"bigint",
	"bigint",
	"double precision",
	"double precision",
	"varchar(128)",
	"text",
	"timestamp with time zone",
	"bytea",
	"bigint",
	"integer",
	"boolean",
	"double precision",
	"timestamp with time zone",
	"varchar(128)",
	"text",
}

func TestSqlTypeForPgDialect(t *testing.T) {
	assert := NewAssert(t)
	d := NewPostgres()
	testModel := structPtrToModel(new(typeTestTable), false, nil)
	for index, column := range testModel.fields {
		if storedResult := postgresSqlTypeResults[index]; storedResult != "-" {
			result := d.sqlType(*column)
			assert.Equal(storedResult, result)
		}
	}
}

func TestPgTransaction(t *testing.T) {
	registerPgTest()
	doTestTransaction(NewAssert(t))
}

func TestPgSaveAndDelete(t *testing.T) {
	mg, q := setupPgDb()
	doTestSaveAndDelete(NewAssert(t), mg, q)
}

func TestPgSaveAgain(t *testing.T) {
	mg, q := setupPgDb()
	doTestSaveAgain(NewAssert(t), mg, q)
}

func TestPgForeignKey(t *testing.T) {
	registerPgTest()
	doTestForeignKey(NewAssert(t))
}

func TestPgFind(t *testing.T) {
	registerPgTest()
	doTestFind(NewAssert(t))
}

func TestPgCreateTable(t *testing.T) {
	mg, _ := setupPgDb()
	doTestCreateTable(NewAssert(t), mg)
}

func TestPgUpdate(t *testing.T) {
	mg, q := setupPgDb()
	doTestUpdate(NewAssert(t), mg, q)
}

func TestPgValidation(t *testing.T) {
	mg, q := setupPgDb()
	doTestValidation(NewAssert(t), mg, q)
}

func TestPgBoolType(t *testing.T) {
	mg, q := setupPgDb()
	doTestBoolType(NewAssert(t), mg, q)
}

func TestPgStringPk(t *testing.T) {
	mg, q := setupPgDb()
	doTestStringPk(NewAssert(t), mg, q)
}

func TestPgCount(t *testing.T) {
	registerPgTest()
	doTestCount(NewAssert(t))
}

func TestPgQueryMap(t *testing.T) {
	mg, q := setupPgDb()
	doTestQueryMap(NewAssert(t), mg, q)
}

func TestPgBulkInsert(t *testing.T) {
	registerPgTest()
	doTestBulkInsert(NewAssert(t))
}

func TestPgQueryStruct(t *testing.T) {
	registerPgTest()
	doTestQueryStruct(NewAssert(t))
}

func TestPgConnectionLimit(t *testing.T) {
	registerPgTest()
	doTestConnectionLimit(NewAssert(t))
}

func TestPgIterate(t *testing.T) {
	registerPgTest()
	doTestIterate(NewAssert(t))
}

func TestPgAddColumnSQL(t *testing.T) {
	doTestAddColumSQL(NewAssert(t), pgSyntax)
}

func TestPgCreateTableSQL(t *testing.T) {
	doTestCreateTableSQL(NewAssert(t), pgSyntax)
}

func TestPgCreateIndexSQL(t *testing.T) {
	doTestCreateIndexSQL(NewAssert(t), pgSyntax)
}

func TestPgInsertSQL(t *testing.T) {
	doTestInsertSQL(NewAssert(t), pgSyntax)
}

func TestPgUpdateSQL(t *testing.T) {
	doTestUpdateSQL(NewAssert(t), pgSyntax)
}

func TestPgDeleteSQL(t *testing.T) {
	doTestDeleteSQL(NewAssert(t), pgSyntax)
}

func TestPgSelectionSQL(t *testing.T) {
	doTestSelectionSQL(NewAssert(t), pgSyntax)
}

func TestPgQuerySQL(t *testing.T) {
	doTestQuerySQL(NewAssert(t), pgSyntax)
}

func TestPgDropTableSQL(t *testing.T) {
	doTestDropTableSQL(NewAssert(t), pgSyntax)
}

func TestPgSaveNullable(t *testing.T) {
	mg, q := setupPgDb()
	doTestSaveNullable(NewAssert(t), mg, q)
}

func TestPgDataSourceName(t *testing.T) {
	dsn := new(DataSourceName)
	dsn.DbName = "abc"
	dsn.Username = "john"
	dsn.Dialect = NewPostgres()
	assert := NewAssert(t)
	assert.Equal("user=john dbname=abc", dsn)
	dsn.Password = "123"
	assert.Equal("user=john password=123 dbname=abc", dsn)
	dsn.Host = "192.168.1.3"
	assert.Equal("user=john password=123 dbname=abc host=192.168.1.3", dsn)
	dsn.UnixSocket = true
	assert.Equal("user=john password=123 dbname=abc host=/192.168.1.3", dsn)
	dsn.Port = "9876"
	assert.Equal("user=john password=123 dbname=abc host=/192.168.1.3 port=9876", dsn)
}

func BenchmarkPgFind(b *testing.B) {
	registerPgTest()
	doBenchmarkFind(b, b.N)
}

func BenchmarkPgDbQuery(b *testing.B) {
	registerPgTest()
	doBenchmarkDbQuery(b, b.N)
}

func BenchmarkPgStmtQuery(b *testing.B) {
	registerPgTest()
	doBenchmarkStmtQuery(b, b.N)
}

func BenchmarkPgTransaction(b *testing.B) {
	registerPgTest()
	doBenchmarkTransaction(b, b.N)
}
