package qbs

import "testing"

func doBenchmarkFind(b *testing.B) {
	b.StopTimer()
	bas := new(basic)
	bas.Name = "Basic"
	bas.State = 3
	mg, _ := GetMigration()
	mg.DropTable(bas)
	mg.CreateTableIfNotExists(bas)
	q, _ := GetQbs()
	q.Save(bas)
	closeMigrationAndQbs(mg, q)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		ba := new(basic)
		ba.Id = 1
		q, _ = GetQbs()
		q.Find(ba)
		q.Close()
	}
	ChangePoolSize(10)
}
