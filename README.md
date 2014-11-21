Qbs
===

Qbs stands for Query By Struct. A Go ORM. [中文版 README](https://github.com/coocood/qbs/blob/master/README_ZH.md)

[![Build Status](https://drone.io/github.com/coocood/qbs/status.png)](https://drone.io/github.com/coocood/qbs/latest)

##ChangeLog

* 2013.03.14: index name has changed to `{table name}_{column name}`.
    - For existing application with existing database, update to this change may lead to creating redundant index, you may need to drop duplicated index manually.
* 2013.03.14: make all internal structures unexported.
* 2013.05.22: fixed memory leak issue.
* 2013.05.23: Breaking change, improved performance up to 100%, removed deprecated methods, make Db and Tx field unexported.
* 2013.05.25: Added `QueryStruct` method to do raw SQL query and fill the result into struct or slice of struct.
 Added support for defining custom table/struct and column/field name convertion function, so you are not forced to use snake case in database.
* 2013.05.27: dropped go 1.0 support, only support go 1.1, changed to use build-in sql.DB connection pool.
* 2013.05.29: Added `Iterate` method for processing large data without loading all rows into memory.
* 2013.08.03: Fixed a bug that exceeds max_prepared_stmt_count

##Features

* Define table schema in struct type, create table if not exists.
* Detect table columns in database and alter table add new column automatically.
* Define selection clause in struct type, fields in the struct type become the columns to be selected.
* Define join query in struct type by add pointer fields which point to the parent table's struct.
* Do CRUD query by struct value.
* After a query, all the data you need will be filled into the struct value.
* Compose where clause by condition, which can easily handle complex precedence of "AND/OR" sub conditions.
* If Id value in the struct is provided, it will be added to the where clause.
* "Created" column will be set to current time when insert, "Updated" column will be set to current time when insert and update.
* Struct type can implement Validator interface to do validation before insert or update.
* Support MySQL, PosgreSQL and SQLite3.
* Support connection pool.

## Performance

`Qbs.Find` is about 60% faster on mysql, 130% faster on postgreSQL than raw `Db.Query`, about 20% slower than raw `Stmt.Query`. (benchmarked on windows).
The reason why it is faster than `Db.Query` is because all prepared Statements are cached in map.

##Install

Only support go 1.1+

Go get to get the most recent source code.

    go get github.com/coocood/qbs

New version may break backwards compatibility, so for production project, it's better to 
download the tagged version. The most recent release is [v0.2](https://github.com/coocood/qbs/tags).

tags with same minor version would be backward compatible, e.g `v0.1` and `v0.1.1`.

tags with different minor version would break compatibility, e.g `v0.1.1` and `v0.2`.

## API Documentation

See [Gowalker](http://gowalker.org/github.com/coocood/qbs) for complete documentation.

##Get Started

###First you need to register your database

* The `qbs.Register` function has two more arguments than `sql.Open`, they are database name and dilect instance.
* You only need to call it once at the start time..

        func RegisterDb(){
            qbs.Register("mysql","qbs_test@/qbs_test?charset=utf8&parseTime=true&loc=Local", "qbs_test", qbs.NewMysql())
        }

### Define a model `User`
- If the field name is `Id` and field type is `int64`, the field will be considered as the primary key of the table.
if you want define a primary key with name other than `Id`, you can set the tag `qbs:"pk"` to explictly mark the field as primary key.
- The tag of `Name` field `qbs:"size:32,index"` is used to define the column attributes when create the table, attributes are comma seperated, inside double quotes.
- The `size:32` tag on a string field will be translated to SQL `varchar(32)`, add `index` attribute to create a index on the column, add `unique` attribute to create a unique index on the column
- Some DB (MySQL) can not create a index on string column without `size` defined.

        type User struct {
            Id   int64
            Name string `qbs:"size:32,index"`
        }

- If you want to create multi column index, you should implement `Indexed` interface by define a `Indexes` method like the following.

        func (*User) Indexes(indexes *qbs.Indexes){
            //indexes.Add("column_a", "column_b") or indexes.AddUnique("column_a", "column_b")
        }

###Create a new table

- call `qbs.GetMigration` function to get a Migration instance, and then use it to create a table.
- When you create a table, if the table already exists, it will not recreate it, but looking for newly added columns or indexes in the model, and execute add column or add index operation.
- It is better to do create table task at the start time, because the Migration only do incremental operation, it is safe to keep the table creation code in production enviroment.
- `CreateTableIfNotExists` expect a struct pointer parameter.

        func CreateUserTable() error{
            migration, err := qbs.GetMigration()
            if err != nil {
                return err
            }
            defer migration.Close()
            return migration.CreateTableIfNotExists(new(User))
        }

### Get and use `*qbs.Qbs` instance：
- Suppose we are in a handle http function. call `qbs.GetQbs()` to get a instance.
- Be sure to close it by calling `defer q.Close()` after get it.
- qbs has connection pool, the default size is 100, you can call `qbs.ChangePoolSize()` to change the size.

        func GetUser(w http.ResponseWriter, r *http.Request){
        	q, err := qbs.GetQbs()
        	if err != nil {
        		fmt.Println(err)
        		w.WriteHeader(500)
        		return
        	}
        	defer q.Close()
        	u, err := FindUserById(q, 6)
        	data, _ := json.Marshal(u)
        	w.Write(data)
        }

### Inset a row：
- Call `Save` method to insert or update the row，if the primary key field `Id` has not been set, `Save` would execute insert stamtment.
- If `Id` is set to a positive integer, `Save` would query the count of the row to find out if the row already exists, if not then execute `INSERT` statement.
otherwise execute `UPDATE`.
- `Save` expects a struct pointer parameter.

        func CreateUser(q *qbs.Qbs) (*User,error){
            user := new(User)
            user.Name = "Green"
            _, err := q.Save(user)
            return user,err
        }

### Find：
- If you want to get a row by `Id`, just assign the `Id` value to the model instance.

        func FindUserById(q *qbs.Qbs, id int64) (*User, error) {
            user := new(User)
            user.Id = id
            err := q.Find(user)
            return user, err
        }

- Call `FindAll` to get multiple rows, it expects a pointer of slice, and the element of the slice must be a pointer of struct.

        func FindUsers(q *qbs.Qbs) ([]*User, error) {
        	var users []*User
        	err := q.Limit(10).Offset(10).FindAll(&users)
        	return users, err
        }

- If you want to add conditions other than `Id`, you should all `Where` method. `WhereEqual("name", name)` is equivalent to `Where（"name = ?", name)`, just a shorthand method.
- Only the last call to `Where`/`WhereEqual` counts, so it is only applicable to define simple condition.
- Notice that the column name passed to `WhereEqual` method is lower case, by default, all the camel case field name and struct name will be converted to snake case in database storage,
so whenever you pass a column name or table name parameter in string, it should be in snake case.
- You can change the convertion behavior by setting the 4 convertion function variable: `FieldNameToColumnName`,`StructNameToTableName`,`ColumnNameToFieldName`,`TableNameToStructName` to your own function.

        func FindUserByName(q *qbs.Qbs, n string) (*User, error) {
            user := new(User)
            err := q.WhereEqual("name", n).Find(user)
            return user, err
        }

- If you need to define more complex condition, you should call `Condition` method, it expects a `*Condition` parameter.
 you can get a new condition instance by calling `qbs.NewCondition`, `qbs.NewEqualCondition` or `qbs.NewInCondition` function.
- `*Condition` instance has `And`, `Or` ... methods, can be called sequentially to construct a complex condition.
- `Condition` method of Qbs instance should only be called once as well, it will replace previous condition defined by `Condition` or `Where` methods.

        func FindUserByCondition(q *qbs.Qbs) (*User, error) {
            user := new(User)
            condition1 := qbs.NewCondition("id > ?", 100).Or("id < ?", 50).OrEqual("id", 75)
            condition2 := qbs.NewCondition("name != ?", "Red").And("name != ?", "Black")
            condition1.AndCondition(condition2)
            err := q.Condition(condition1).Find(user)
            return user, err
        }

### Update a single row
- To update a single row, you should call `Find` first, then update the model, and `Save` it.

        func UpdateOneUser(q *qbs.Qbs, id int64, name string) (affected int64, error){
        	user, err := FindUserById(q, id)
        	if err != nil {
        		return 0, err
        	}
        	user.Name = name
        	return q.Save(user)
        }

### Update multiple row
- Call `Update` to update multiple rows at once, but you should call this method cautiously, if the the model struct contains all the columns, it will update every column, most of the time this is not what we want.
- The right way to do it is to define a temporary model struct in method or block, that only contains the column we want to update.

        func UpdateMultipleUsers(q *qbs.Qbs)(affected int64, error) {
        	type User struct {
        		Name string
        	}
        	user := new(User)
        	user.Name = "Blue"
        	return q.WhereEqual("name", "Green").Update(user)
        }

### Delete
- call `Delete` method to delete a row, there must be at least one condition defined, either by `Id` value, or by `Where`/`Condition`.

        func DeleteUser(q *qbs.Qbs, id int64)(affected int64, err error) {
        	user := new(User)
        	user.Id = id
        	return q.Delete(user)
        }

### Define another table for join query
- For join query to work, you should has a pair of fields to define the join relationship in the model struct.
- Here the model `Post` has a `AuthorId` int64 field, and has a `Author` field of type `*User`.
- The rule to define join relationship is like `{xxx}Id int64`, `{xxx} *{yyy}`.
- As the `Author` field is pointer type, it will be ignored when creating table.
- As `AuthorId` is a join column, a index of it will be created automatically when creating the table, so you don't have to add `qbs:"index"` tag on it.
- You can also set the join column explicitly by add a tag `qbs:"join:Author"` to it for arbitrary field Name. here `Author` is the struct pointer field of the parent table model.
- To define a foreign key constraint, you have to explicitly add a tag `qbs:"fk:Author"` to the foreign key column, and an index will be created as well when creating table.
- `Created time.Time` field will be set to the current time when insert a row,`Updated time.Time` field will be set to current time when update the row.
- You can explicitly set tag `qbs:"created"` or `qbs:"updated"` on `time.Time` field to get the functionality for arbitrary field name.

        type Post struct {
            Id int64
            AuthorId int64
            Author *User
            Content string
            Created time.Time
            Updated time.Time
        }

### Omit some column
- Sometimes we do not need to get every field of a model, especially for joined field (like `Author` field) or large field (like `Content` field).
- Omit them will get better performance.

        func FindPostsOmitContentAndCreated(q *qbs.Qbs) ([]*Post, error) {
        	var posts []*Post
        	err := q.OmitFields("Content","Created").Find(&posts)
        	return posts, err
        }

- With `OmitJoin`, you can omit every join fields, return only the columns in a single table, and it can be used along with `OmitFields`.

        func FindPostsOmitJoin(q *qbs.Qbs) ([]*Post, error) {
        	var posts []*Post
        	err := q.OmitJoin().OmitFields("Content").Find(&posts)
        	return posts, err
        }

##Projects use Qbs:

- a CMS system [toropress](https://github.com/insionng/toropress)
- Go documentation reference website [Gowalker](http://gowalker.org/)
- Nebri OS (https://nebrios.com/)

##Contributors
[Erik Aigner](https://github.com/eaigner)
Qbs was originally a fork from [hood](https://github.com/eaigner/hood) by [Erik Aigner](https://github.com/eaigner), 
but I changed more than 80% of the code, then it ended up become a totally different ORM.

[NuVivo314](https://github.com/NuVivo314),  [Jason McVetta](https://github.com/jmcvetta), [pix64](https://github.com/pix64), [vadimi](https://github.com/vadimi), [Ravi Teja](https://github.com/tejainece).
