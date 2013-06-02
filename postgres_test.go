package qbs

import (
	"fmt"
	_ "github.com/lib/pq"
	"testing"
	"time"
)

const (
	pgDriver    = "postgres"
	pgDrvFormat = "user=%v dbname=%v sslmode=disable"
)

var pgSyntax = dialectSyntax{
	NewPostgres(),
	`CREATE TABLE IF NOT EXISTS "without_pk" ( "first" text, "last" text, "amount" integer )`,
	`CREATE TABLE "with_pk" ( "primary" bigserial PRIMARY KEY, "first" text, "last" text, "amount" integer )`,
	`INSERT INTO "sql_gen_model" ("prim", "first", "last", "amount") VALUES ($1, $2, $3, $4) RETURNING "prim"`,
	`UPDATE "sql_gen_model" SET "first" = $1, "last" = $2, "amount" = $3 WHERE "prim" = $4`,
	`DELETE FROM "sql_gen_model" WHERE "prim" = $1`,
	`SELECT "post"."id", "post"."author_id", "post"."content", "author"."id" AS author___id, "author"."name" AS author___name FROM "post" LEFT JOIN "user" AS "author" ON "post"."author_id" = "author"."id"`,
	`SELECT "name", "grade", "score" FROM "student" WHERE (grade IN ($1, $2, $3)) AND ((score <= $4) OR (score >= $5)) ORDER BY "name", "grade" DESC LIMIT $6 OFFSET $7`,
	`DROP TABLE IF EXISTS "drop_table"`,
	`ALTER TABLE "a" ADD COLUMN "c" varchar(100)`,
	`CREATE UNIQUE INDEX "iname" ON "itable" ("a", "b", "c")`,
	`CREATE INDEX "iname2" ON "itable2" ("d", "e")`,
}

func registerPgTest() {
	Register(pgDriver, fmt.Sprintf(pgDrvFormat, "postgres", testDbName), testDbName, NewPostgres())
}

func setupPgDb() (*Migration, *Qbs) {
	registerPgTest()
	mg, _ := GetMigration()
	q, _ := GetQbs()
	return mg, q
}

func TestSqlTypeForPgDialect(t *testing.T) {
	assert := NewAssert(t)
	d := NewPostgres()
	assert.Equal("boolean", d.sqlType(true, 0))
	var indirect interface{} = true
	assert.Equal("boolean", d.sqlType(indirect, 0))
	assert.Equal("integer", d.sqlType(uint32(2), 0))
	assert.Equal("bigint", d.sqlType(int64(1), 0))
	assert.Equal("double precision", d.sqlType(1.8, 0))
	assert.Equal("bytea", d.sqlType([]byte("asdf"), 0))
	assert.Equal("text", d.sqlType("astring", 0))
	assert.Equal("varchar(255)", d.sqlType("a", 255))
	assert.Equal("varchar(128)", d.sqlType("b", 128))
	assert.Equal("timestamp with time zone", d.sqlType(time.Now(), 0))
}

func TestPgTransaction(t *testing.T) {
	registerPgTest()
	doTestTransaction(t)
}

func TestPgSaveAndDelete(t *testing.T) {
	mg, q := setupPgDb()
	doTestSaveAndDelete(t, mg, q)
}

func TestPgSaveAgain(t *testing.T) {
	mg, q := setupPgDb()
	doTestSaveAgain(t, mg, q)
}

func TestPgForeignKey(t *testing.T) {
	registerPgTest()
	doTestForeignKey(t)
}

func TestPgFind(t *testing.T) {
	registerPgTest()
	doTestFind(t)
}

func TestPgCreateTable(t *testing.T) {
	mg, _ := setupPgDb()
	doTestCreateTable(t, mg)
}

func TestPgUpdate(t *testing.T) {
	mg, q := setupPgDb()
	doTestUpdate(t, mg, q)
}

func TestPgValidation(t *testing.T) {
	mg, q := setupPgDb()
	doTestValidation(t, mg, q)
}

func TestPgBoolType(t *testing.T) {
	mg, q := setupPgDb()
	doTestBoolType(t, mg, q)
}

func TestPgStringPk(t *testing.T) {
	mg, q := setupPgDb()
	doTestStringPk(t, mg, q)
}

func TestPgCount(t *testing.T) {
	registerPgTest()
	doTestCount(t)
}

func TestPgQueryMap(t *testing.T) {
	mg, q := setupPgDb()
	doTestQueryMap(t, mg, q)
}

func TestPgBulkInsert(t *testing.T) {
	registerPgTest()
	doTestBulkInsert(t)
}

func TestPgQueryStruct(t *testing.T) {
	registerPgTest()
	doTestQueryStruct(t)
}

func TestPgConnectionLimit(t *testing.T) {
	registerPgTest()
	doTestConnectionLimit(t)
}

func TestPgIterate(t *testing.T) {
	registerPgTest()
	doTestIterate(t)
}

func TestPgAddColumnSQL(t *testing.T) {
	doTestAddColumSQL(t, pgSyntax)
}

func TestPgCreateTableSQL(t *testing.T) {
	doTestCreateTableSQL(t, pgSyntax)
}

func TestPgCreateIndexSQL(t *testing.T) {
	doTestCreateIndexSQL(t, pgSyntax)
}

func TestPgInsertSQL(t *testing.T) {
	doTestInsertSQL(t, pgSyntax)
}

func TestPgUpdateSQL(t *testing.T) {
	doTestUpdateSQL(t, pgSyntax)
}

func TestPgDeleteSQL(t *testing.T) {
	doTestDeleteSQL(t, pgSyntax)
}

func TestPgSelectionSQL(t *testing.T) {
	doTestSelectionSQL(t, pgSyntax)
}

func TestPgQuerySQL(t *testing.T) {
	doTestQuerySQL(t, pgSyntax)
}

func TestPgDropTableSQL(t *testing.T) {
	doTestDropTableSQL(t, pgSyntax)
}

func BenchmarkPgFind(b *testing.B) {
	registerPgTest()
	doBenchmarkFind(b)
}

func BenchmarkPgDbQuery(b *testing.B) {
	registerPgTest()
	doBenchmarkDbQuery(b)
}

func BenchmarkPgStmtQuery(b *testing.B) {
	registerPgTest()
	doBenchmarkStmtQuery(b)
}

func BenchmarkPgTransaction(b *testing.B) {
	registerPgTest()
	doBenchmarkTransaction(b)
}
