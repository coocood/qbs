
package example

import (
	"github.com/coocood/qbs"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"time"
)

type User struct {
	Id   int64
	Name string `qbs:"size:50,index"`
}

type Post struct {
	Id int64
	AuthorId int64
	Author *User
	Content string
	Created time.Time
	Updated time.Time
}

func OpenDb() (*sql.DB, error){
	db, err := sql.Open("mysql", "qbs_test@/qbs_test?charset=utf8&loc=Local")
	return db, err
}

func CreateUserTable() error{
	db, err := OpenDb()
	if err != nil {
		return err
	}
	migration := qbs.NewMigration(db,"qbs_test", qbs.NewMysql())
	defer migration.Close()
	return migration.CreateTableIfNotExists(new(User))
}

func GetQbs() (q *qbs.Qbs, err error){
	db := qbs.GetFreeDB()
	if db == nil{
		db, err = OpenDb()
		if err != nil {
			return nil,err
		}
	}
	q = qbs.New(db, qbs.NewMysql())
	return q, nil
}


func CreateUser(q *qbs.Qbs) (*User,error){
	user := new(User)
	user.Name = "Green"
	_, err := q.Save(user)
	return user,err
}

func FindUserById(q *qbs.Qbs, id int64) (*User, error) {
	user := new(User)
	user.Id = id
	err := q.Find(user)
	return user, err
}

func FindUserByName(q *qbs.Qbs, n string) (*User, error) {
	user := new(User)
	err := q.WhereEqual("name", n).Find(user)
	return user, err
}

func FindUserByCondition(q *qbs.Qbs) (*User, error) {
	user := new(User)
	condition1 := qbs.NewCondition("id > ?", 100).Or("id < ?", 50).OrEqual("id", 75)
	condition2 := qbs.NewCondition("name != ?", "Red").And("name != ?", "Black")
	condition1.AndCondition(condition2)
	err := q.Condition(condition1).Find(user)
	return user, err
}

func FindUsers(q *qbs.Qbs) ([]*User, error) {
	var users []*User
	err := q.Limit(10).Offset(10).FindAll(&users)
	return users, err
}

func UpdateOneUser(q *qbs.Qbs, id int64, name string) (affected int64, err error){
	user, err := FindUserById(q, id)
	if err != nil {
		return 0, err
	}
	user.Name = name
	return q.Save(user)
}


func UpdateMultipleUsers(q *qbs.Qbs)(affected int64, err error) {
	type User struct {
		Name string
	}
	user := new(User)
	user.Name = "Blue"
	return q.WhereEqual("name", "Green").Update(user)
}

func DeleteUser(q *qbs.Qbs, id int64)(affected int64, err error) {
	user := new(User)
	user.Id = id
	return q.Delete(user)
}

func FindPostsOmitContentAndCreated(q *qbs.Qbs) ([]*Post, error) {
	var posts []*Post
	err := q.OmitFields("Content","Created").Find(&posts)
	return posts, err
}

func FindPostsOmitJoin(q *qbs.Qbs) ([]*Post, error) {
	var posts []*Post
	err := q.OmitJoin().OmitFields("Content").Find(&posts)
	return posts, err
}
