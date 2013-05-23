package main

import (
	f "fmt"
	"github.com/lizijian/qbs"
	_ "github.com/mattn/go-sqlite3"
	"time"
)

type Task struct {
	Id          int64
	Name        string `qbs:"size:100"`
	Content     string `qbs:"size:256"`
	State       string `qbs:"size:10"`
	Created     time.Time
	InitiatorId int64 `qbs:"fk:User"`
	Initiator   *User
	Users       []*User `qbs:"m2m:TaskUser"`
}

type TaskUser struct {
	Id     int64
	UserId int64 `qbs:"fk:User"`
	User   *User
	TaskId int64 `qbs:"fk:Usergroup"`
	Task   *Task
}

type User struct {
	Id       int64
	Email    string `qbs:"index"`
	Password string `qbs:"size:100"`
	Username string `qbs:"size:100,index"`
}

func main() {
	qbs.Register("sqlite3", "test.db", "", qbs.NewSqlite3())
	q, err := qbs.GetQbs()
	if err != nil {
		panic(err)
	}
	q.Begin()
	defer q.Close()
	q.Log = true
	var ts []*Task
	if err = q.LoadM2mFields("Users").FindAll(&ts); err != nil {
		panic(err)
	}
	f.Println("tasks:", ts)
	for _, v := range ts {
		f.Println("tasks id ", v.Id, ":", v)
		for _, user := range v.Users {
			f.Println("task id=", v.Id, ", users ", user.Id, ":", user)
		}
	}
	// lazy load m2m
	ts = nil
	if err = q.FindAll(&ts); err != nil {
		panic(err)
	}
	// before lazy loading
	f.Println("tasks without load m2m:", ts)
	for _, v := range ts {
		f.Println("tasks id ", v.Id, ":", v)
		for _, user := range v.Users {
			f.Println("task id=", v.Id, ", users ", user.Id, ":", user)
		}
	}
	for _, v := range ts {
		// lazy load m2m with conditions
		// if err = q.Limit(1).Offset(1).Order("Id").LoadM2mFields("Users").LoadM2m(v); err != nil {
		// if err = q.OrderBy("user.id").LoadM2mFields("Users").LoadM2m(v); err != nil {
		if err = q.Condition(qbs.NewCondition("user.id = ?", 2)).OrderByDesc("user.id").LoadM2mFields("Users").LoadM2m(v); err != nil {
			panic(err)
		}
	}
	// display lazy loaded m2m
	f.Println("tasks lazy load m2m:", ts)
	for _, v := range ts {
		f.Println("tasks id ", v.Id, ":", v)
		for _, user := range v.Users {
			f.Println("task id=", v.Id, ", users ", user.Id, ":", user)
		}
	}
}
