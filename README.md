Qbs
===

Qbs stands for Query By Struct. A Go ORM.

##ChangeLog

* 2013.03.14: index name has changed to `{table name}_{column name}`.
    - For existing application with existing database, update to this change may lead to creating redundant index, you may need to drop duplicated index manually.

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

##Install

    go get github.com/coocood/qbs

## API Documentation

See [GoDoc](http://godoc.org/github.com/coocood/qbs) for automatic
documentation.

##Warning
 
* New version may break backwards compatibility.
* Once you installed it for the first time by "go get", do not "go get" again for your existing application.
* You should copy local source code when you need to compile your application on another mechine.
* Or you can simply fork this repo, so you won't get any suprise.
* When new version break backwards compatiblity, a branch with the number of the date will be created to keep the legacy code.

##Examples

    func FindAuthorName(){
        //create Qbs instance
        db, _ = sql.Open("mymysql", "qbs_test/qbs_test/")
        q := qbs.New(db, qbs.NewMysql())
        defer q.Db.Close()

        //define struct
        type User struct {
            Id   int64
            Name string
        }
        type Post struct {
            Id       int64
            Title    string
            AuthorId int64
            Author   *User
        }

        //find the post with Id 5.
        aPost := new(Post)
        aPost.Id = 5

        //assume table "user", "post" already exists in database, and post table have row which id is 5 and author's name is "john"
        q.Find(aPost)

        // result would be "john"
        fmt.Println(aPost.Author.Name)


    }

More advanced examples can be found in test files.

A complete application can be found in a CMS system [toropress](https://github.com/insionng/toropress), specifically in [models.go](https://github.com/insionng/toropress/blob/master/models/models.go) file

##Restriction

* Every table name and culumn name in the database must be lower case, must not have any trailing "_" or any preceding "___"

* Define fields in camelcase to follow Go's nameing convention, Qbs will trasnlate them to snakecase in sql statement.

##Field tag syntax

###Ignore field:

    `qbs:"-"`

###Define primary key:

- Primary key must be of type in64 or string, string type primary key must define column size.
- If field name is "Id" and type is "int64" the field becomes a implicit primary key.


    `qbs:"pk"`


###Define not null column:

    `qbs:"notnull"`

###Define column size:

    `qbs:"size:255"`

###Define column default value:

    `qbs:"default:'abc'"`

###Define column index:

    `qbs:"index"`

###Define unique index:

    `qbs:"unique"`

###Define multiple attributes with comma separator

    `qbs:"size:100,default:'abc'"`

###Define foreign key:
	
	type User struct{
		Id int64
		Name string `qbs:"size:255"`
	}

    type Post struct{
    	Id int64
    	AuthorId int64 `qbs:"fk:Author"`
    	Author *User
    	Content string
    }

###Define Join without foreign key constraint:

    `qbs:"join:Author"`

- If a struct field's type is int64 and its suffix is "Id"(converted to "_id" in database), And the rest of the name can be found in the struct field,
and that field is a pointer of struct type, then it become a implicit join, so in a find query, the previous example's `qbs:"join:Author"` tag can be omitted.
It will perform a join query automatically.

    type Post struct{
    	Id int64
    	AuthorId int64
    	Author *User
    	Content string
    }

###Define Updated and Created field:

	Updated time.Time `qbs:"updated"`
	Created time.Time `qbs:"created"`

- If the field name is "Updated" and its type is "time.Time", then the field became the updated field automatically.
Its value will get updated when update. If the field name is "Created" and its type is "time.Time" it's value will be set when insert.
So the previous example's tag can be omitted.


##Contributors
[Erik Aigner](https://github.com/eaigner)
Qbs was originally a fork from [hood](https://github.com/eaigner/hood) by [Erik Aigner](https://github.com/eaigner), 
but I changed more than 80% of the code, then it ended up become a totally different ORM.

[NuVivo314](https://github.com/NuVivo314),  [Jason McVetta](https://github.com/jmcvetta)
