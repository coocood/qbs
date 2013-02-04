Qbs
===

Qbs stands for Query By Struct. A Go ORM.


##Limitation

*Every table must have an primary key Id field which has the int64 underlying type.
*Table name or column name should be in camel style in go code and will be converted to lower case snake style in database 
*Join type is left join, if you need an inner join, that can be achieved by adding a where condition
*Currently join only support one level, deeper Join is not supported yet, but it's possible in the future update.
*Foreign keys are cascade on delete by default, other options has not been supported yet.

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

    `sql:"size:100,default:'abc'"

Define foreign key:
	
	type User struct{
		Id Id
		Name string `sql:"size:255"`
	}

    type Post struct{
    	Id Id
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
    	Id Id
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
