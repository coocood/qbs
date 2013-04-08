
package example

import (
	"github.com/coocood/qbs"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
)

type User struct {
	Id   int64
	Name string `qbs:"size:50,index"`
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
