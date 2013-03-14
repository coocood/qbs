package qbs

import (
	"github.com/coocood/assrt"
	"testing"
)

type dialectSyntax struct {
	dialect                         Dialect
	createTableWithoutPkIfExistsSql string
	createTableWithPkSql            string
	insertSql                       string
	updateSql                       string
	deleteSql                       string
	selectionSql                    string
	querySql                        string
	dropTableIfExistsSql            string
	addColumnSql                    string
	createUniqueIndexSql            string
	createIndexSql                  string
}

var allDialectSyntax = []dialectSyntax{
	dialectSyntax{
		NewPostgres(),
		`CREATE TABLE IF NOT EXISTS "without_pk" ( "first" text, "last" text, "amount" integer )`,
		`CREATE TABLE "with_pk" ( "primary" bigserial PRIMARY KEY, "first" text, "last" text, "amount" integer )`,
		`INSERT INTO "sql_gen_model" ("prim", "first", "last", "amount") VALUES ($1, $2, $3, $4) RETURNING "prim"`,
		`UPDATE "sql_gen_model" SET "first" = $1, "last" = $2, "amount" = $3 WHERE "prim" = $4`,
		`DELETE FROM "sql_gen_model" WHERE "prim" = $1`,
		`SELECT "post"."id", "post"."author_id", "post"."content", "author"."id" AS author___id, "author"."name" AS author___name FROM "post" LEFT JOIN "user" AS "author" ON "post"."author_id" = "author"."id"`,
		`SELECT "name", "grade", "score" FROM "student" WHERE (grade IN ($1, $2, $3)) AND ((score <= $4) OR (score >= $5)) ORDER BY "name" , "grade" DESC LIMIT $6 OFFSET $7`,
		`DROP TABLE IF EXISTS "drop_table"`,
		`ALTER TABLE "a" ADD COLUMN "c" varchar(100)`,
		`CREATE UNIQUE INDEX "iname" ON "itable" ("a", "b", "c")`,
		`CREATE INDEX "iname2" ON "itable2" ("d", "e")`,
	},
	dialectSyntax{
		NewMysql(),
		"CREATE TABLE IF NOT EXISTS `without_pk` ( `first` longtext, `last` longtext, `amount` int )",
		"CREATE TABLE `with_pk` ( `primary` bigint PRIMARY KEY AUTO_INCREMENT, `first` longtext, `last` longtext, `amount` int )",
		"INSERT INTO `sql_gen_model` (`prim`, `first`, `last`, `amount`) VALUES (?, ?, ?, ?)",
		"UPDATE `sql_gen_model` SET `first` = ?, `last` = ?, `amount` = ? WHERE `prim` = ?",
		"DELETE FROM `sql_gen_model` WHERE `prim` = ?",
		"SELECT `post`.`id`, `post`.`author_id`, `post`.`content`, `author`.`id` AS author___id, `author`.`name` AS author___name FROM `post` LEFT JOIN `user` AS `author` ON `post`.`author_id` = `author`.`id`",
		"SELECT `name`, `grade`, `score` FROM `student` WHERE (grade IN (?, ?, ?)) AND ((score <= ?) OR (score >= ?)) ORDER BY `name` , `grade` DESC LIMIT ? OFFSET ?",
		"DROP TABLE IF EXISTS `drop_table`",
		"ALTER TABLE `a` ADD COLUMN `c` varchar(100)",
		"CREATE UNIQUE INDEX `iname` ON `itable` (`a`, `b`, `c`)",
		"CREATE INDEX `iname2` ON `itable2` (`d`, `e`)",
	},
	dialectSyntax{
		NewSqlite3(),
		"CREATE TABLE IF NOT EXISTS `without_pk` ( `first` text, `last` text, `amount` integer )",
		"CREATE TABLE `with_pk` ( `primary` integer PRIMARY KEY AUTOINCREMENT NOT NULL, `first` text, `last` text, `amount` integer )",
		"INSERT INTO `sql_gen_model` (`prim`, `first`, `last`, `amount`) VALUES (?, ?, ?, ?)",
		"UPDATE `sql_gen_model` SET `first` = ?, `last` = ?, `amount` = ? WHERE `prim` = ?",
		"DELETE FROM `sql_gen_model` WHERE `prim` = ?",
		"SELECT `post`.`id`, `post`.`author_id`, `post`.`content`, `author`.`id` AS author___id, `author`.`name` AS author___name FROM `post` LEFT JOIN `user` AS `author` ON `post`.`author_id` = `author`.`id`",
		"SELECT `name`, `grade`, `score` FROM `student` WHERE (grade IN (?, ?, ?)) AND ((score <= ?) OR (score >= ?)) ORDER BY `name` , `grade` DESC LIMIT ? OFFSET ?",
		"DROP TABLE IF EXISTS `drop_table`",
		"ALTER TABLE `a` ADD COLUMN `c` text",
		"CREATE UNIQUE INDEX `iname` ON `itable` (`a`, `b`, `c`)",
		"CREATE INDEX `iname2` ON `itable2` (`d`, `e`)",
	},
}

type sqlGenModel struct {
	Prim   int64 `qbs:"pk"`
	First  string
	Last   string
	Amount int
}

var sqlGenSampleData = &sqlGenModel{3, "FirstName", "LastName", 6}

func TestAddColumSQL(t *testing.T) {
	for _, info := range allDialectSyntax {
		doTestAddColumSQL(assrt.NewAssert(t), info)
	}
}

func doTestAddColumSQL(assert *assrt.Assert, info dialectSyntax) {
	assert.Logf("Dialect %T\n", info.dialect)
	sql := info.dialect.addColumnSql("a", "c", "", 100)
	assert.Equal(info.addColumnSql, sql)
}
func TestCreateTableSql(t *testing.T) {
	for _, info := range allDialectSyntax {
		doTestCreateTableSql(assrt.NewAssert(t), info)
	}
}

func doTestCreateTableSql(assert *assrt.Assert, info dialectSyntax) {
	assert.Logf("Dialect %T\n", info.dialect)
	type withoutPk struct {
		First  string
		Last   string
		Amount int
	}
	table := &withoutPk{"a", "b", 5}
	model := structPtrToModel(table, true, nil)
	sql := info.dialect.createTableSql(model, true)
	assert.Equal(info.createTableWithoutPkIfExistsSql, sql)
	type withPk struct {
		Primary int64 `qbs:"pk"`
		First   string
		Last    string
		Amount  int
	}
	table2 := &withPk{First: "a", Last: "b", Amount: 5}
	model = structPtrToModel(table2, true, nil)
	sql = info.dialect.createTableSql(model, false)
	assert.Equal(info.createTableWithPkSql, sql)
}

func TestCreateIndexSql(t *testing.T) {
	for _, info := range allDialectSyntax {
		doTestCreateIndexSql(assrt.NewAssert(t), info)
	}
}

func doTestCreateIndexSql(assert *assrt.Assert, info dialectSyntax) {
	assert.Logf("Dialect %T\n", info.dialect)
	sql := info.dialect.createIndexSql("iname", "itable", true, "a", "b", "c")
	assert.Equal(info.createUniqueIndexSql, sql)
	sql = info.dialect.createIndexSql("iname2", "itable2", false, "d", "e")
	assert.Equal(info.createIndexSql, sql)
}

func TestInsertSQL(t *testing.T) {
	for _, info := range allDialectSyntax {
		doTestInsertSQL(assrt.NewAssert(t), info)
	}
}

func doTestInsertSQL(assert *assrt.Assert, info dialectSyntax) {
	assert.Logf("Dialect %T\n", info.dialect)
	model := structPtrToModel(sqlGenSampleData, true, nil)
	criteria := &criteria{model: model}
	criteria.mergePkCondition(info.dialect)
	sql, _ := info.dialect.insertSql(criteria)
	sql = info.dialect.substituteMarkers(sql)
	assert.Equal(info.insertSql, sql)
}

func TestUpdateSQL(t *testing.T) {
	for _, info := range allDialectSyntax {
		doTestUpdateSQL(assrt.NewAssert(t), info)
	}
}

func doTestUpdateSQL(assert *assrt.Assert, info dialectSyntax) {
	assert.Logf("Dialect %T\n", info.dialect)
	model := structPtrToModel(sqlGenSampleData, true, nil)
	criteria := &criteria{model: model}
	criteria.mergePkCondition(info.dialect)
	sql, _ := info.dialect.updateSql(criteria)
	sql = info.dialect.substituteMarkers(sql)
	assert.Equal(info.updateSql, sql)
}

func TestDeleteSQL(t *testing.T) {
	for _, info := range allDialectSyntax {
		doTestDeleteSQL(assrt.NewAssert(t), info)
	}
}

func doTestDeleteSQL(assert *assrt.Assert, info dialectSyntax) {
	assert.Logf("Dialect %T\n", info.dialect)
	model := structPtrToModel(sqlGenSampleData, true, nil)
	criteria := &criteria{model: model}
	criteria.mergePkCondition(info.dialect)
	sql, _ := info.dialect.deleteSql(criteria)
	sql = info.dialect.substituteMarkers(sql)
	assert.Equal(info.deleteSql, sql)
}

func TestSelectionSQL(t *testing.T) {
	for _, info := range allDialectSyntax {
		doTestSelectionSQL(assrt.NewAssert(t), info)
	}
}

func doTestSelectionSQL(assert *assrt.Assert, info dialectSyntax) {
	assert.Logf("Dialect %T\n", info.dialect)
	type User struct {
		Id   int64
		Name string
	}
	type Post struct {
		Id       int64
		AuthorId int64 `qbs:"fk:Author"`
		Author   *User
		Content  string
	}
	model := structPtrToModel(new(Post), true, nil)
	criteria := new(criteria)
	criteria.model = model

	sql, _ := info.dialect.querySql(criteria)
	assert.Equal(info.selectionSql, sql)
}

func TestQuerySQL(t *testing.T) {
	for _, info := range allDialectSyntax {
		doTestQuerySQL(assrt.NewAssert(t), info)
	}
}

func doTestQuerySQL(assert *assrt.Assert, info dialectSyntax) {
	assert.Logf("Dialect %T\n", info.dialect)
	type Student struct {
		Name  string
		Grade int
		Score int
	}
	model := structPtrToModel(new(Student), true, nil)
	criteria := new(criteria)
	criteria.model = model
	condition := NewInCondition("grade", []interface{}{6, 7, 8})
	subCondition := NewCondition("score <= ?", 60).Or("score >= ?", 80)
	condition.AndCondition(subCondition)
	criteria.condition = condition
	criteria.orderBys = []order{order{info.dialect.quote("name"),false},order{info.dialect.quote("grade"),true}}
	criteria.offset = 3
	criteria.limit = 10
	sql, _ := info.dialect.querySql(criteria)
	sql = info.dialect.substituteMarkers(sql)
	assert.Equal(info.querySql, sql)
}

func TestDropTableSQL(t *testing.T){
	for _, info := range allDialectSyntax {
		doTestDropTableSQL(assrt.NewAssert(t), info)
	}
}

func doTestDropTableSQL(assert *assrt.Assert, info dialectSyntax) {
	assert.Logf("Dialect %T\n", info.dialect)
	sql := info.dialect.dropTableSql("drop_table")
	assert.Equal(info.dropTableIfExistsSql, sql)
}
