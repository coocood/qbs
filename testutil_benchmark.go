package qbs

import (
	"database/sql"
	"testing"
)

func doBenchmarkFind(b *testing.B, setupDbFunc func() (*Migration, *Qbs), openDbFunc func() (*sql.DB, error), dialect Dialect) {
	b.StopTimer()
	mg, q := setupDbFunc()
	bas := new(basic)
	bas.Name = "Basic"
	bas.State = 3
	mg.DropTable(bas)
	mg.CreateTableIfNotExists(bas)
	q.Save(bas)
	closeMigrationAndQbs(mg, q)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		ba := new(basic)
		ba.Id = 1
		db := GetFreeDB()
		if db == nil {
			db, _ = openDbFunc()
		}
		q := New(db, dialect)
		q.Find(ba)
		q.Close()
	}
	ChangePoolSize(10)
}
