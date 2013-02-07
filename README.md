Qbs
===

Qbs stands for Query By Struct. A Go ORM.

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

##Basic Example

    func FindAuthorName(){
        //create Qbs instance
        db, _ = sql.Open("mymysql", "qbs_test/qbs_test/")
        q := qbs.New(db, qbs.NewMysql())
        defer q.Db.Close()

        //define struct
        type User struct {
            Id   qbs.Id
            Name string
        }
        type Post struct {
            Id       qbs.Id
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

##Limitation

* Every table works with Qbs must have an integer primary key.

* Every table name and culumn name in the database must be lower case, must not have any trailing "_" or any preceding "___"

* Define fields in camelcase to follow Go's nameing convention, Qbs will trasnlate them to snakecase in sql statement.

##Field tag syntax

Define not null column:

    `sql:"notnull"`

Define column size:

    `sql:"size:255"`

Define column default value:

    `sql:"default:'abc'"`

Define column index:

    `sql:"index"`

Define unique index:

    `sql:"unique"`

Define multiple attributes with comma separator

    `sql:"size:100,default:'abc'"`

Define foreign key:
	
	type User struct{
		Id qbs.Id
		Name string `sql:"size:255"`
	}

    type Post struct{
    	Id qbs.Id
    	AuthorId int64 `sql:"fk:Author"`
    	Author *User
    	Content string
    }

Define Join without foreign key constraint:

    `sql:"join:Author"`

If a struct field's type is int64 and its suffix is "Id"(converted to "_id" in database), And the rest of the name can be found in the struct field,
and that field is a pointer of struct type, then it become a implicit join, so in a find query, the previous example's `sql:"join:Author"` tag can be omitted.
It will perform a join query automatically.

    type Post struct{
    	Id qbs.Id
    	AuthorId int64
    	Author *User
    	Content string
    }

Define Updated and Created field:

	Updated time.Time `sql:"updated"`
	Created time.Time `sql:"created"`

If the field name is "Updated" and its type is "time.Time", then the field became the updated field automatically.
Its value will get updated when update. If the field name is "Created" and its type is "time.Time" it's value will be set when insert.
So the previous example's tag can be omitted.


##Contributors
[Erik Aigner](https://github.com/eaigner)
Qbs was originally a fork from [hood](https://github.com/eaigner/hood) by [Erik Aigner](https://github.com/eaigner), 
but I changed more than 80% of the code, then it ended up become a totally different ORM.
