package qbs

import (
	"database/sql"
	"fmt"
	"runtime"
)

func doBenchmarkFind(b benchmarker, n int) {
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
	for i := 0; i < n; i++ {
		ba := new(basic)
		ba.Id = 1
		q, _ = GetQbs()
		err := q.Find(ba)
		if err != nil {
			panic(err)
		}
		q.Close()
	}
	b.StopTimer()
	runtime.GC()
	stats := new(runtime.MemStats)
	runtime.ReadMemStats(stats)
	fmt.Printf("alloc:%d, total:%d\n", stats.Alloc, stats.TotalAlloc)
}

func doBenchmarkQueryStruct(b benchmarker, n int) {
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
	for i := 0; i < n; i++ {
		ba := new(basic)
		q, _ = GetQbs()
		err := q.QueryStruct(ba, "SELECT * FROM basic WHERE id = ?", 1)
		if err != nil {
			panic(err)
		}
		q.Close()
	}
	b.StopTimer()
	runtime.GC()
	stats := new(runtime.MemStats)
	runtime.ReadMemStats(stats)
	fmt.Printf("alloc:%d, total:%d\n", stats.Alloc, stats.TotalAlloc)
}

func doBenchmarkStmtQuery(b benchmarker, n int) {
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
	db, _ := sql.Open(driver, driverSource)
	query := q.Dialect.substituteMarkers("SELECT * FROM basic WHERE id = ?")
	stmt, _ := db.Prepare(query)
	for i := 0; i < n; i++ {
		ba := new(basic)
		rows, err := stmt.Query(1)
		if err != nil {
			panic(err)
		}
		rows.Next()
		err = rows.Scan(&ba.Id, &ba.Name, &ba.State)
		if err != nil {
			panic(err)
		}
		rows.Close()
	}
	stmt.Close()
	db.Close()
	b.StopTimer()
	runtime.GC()
	stats := new(runtime.MemStats)
	runtime.ReadMemStats(stats)
	fmt.Printf("alloc:%d, total:%d\n", stats.Alloc, stats.TotalAlloc)
}

func doBenchmarkDbQuery(b benchmarker, n int) {
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
	db, _ := sql.Open(driver, driverSource)
	query := q.Dialect.substituteMarkers("SELECT * FROM basic WHERE id = ?")
	for i := 0; i < n; i++ {
		ba := new(basic)
		rows, err := db.Query(query, 1)
		if err != nil {
			panic(err)
		}
		rows.Next()
		err = rows.Scan(&ba.Id, &ba.Name, &ba.State)
		if err != nil {
			panic(err)
		}
		rows.Close()
	}
	db.Close()
	b.StopTimer()
	runtime.GC()
	stats := new(runtime.MemStats)
	runtime.ReadMemStats(stats)
	fmt.Printf("alloc:%d, total:%d\n", stats.Alloc, stats.TotalAlloc)
}

func doBenchmarkTransaction(b benchmarker, n int) {
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
	for i := 0; i < n; i++ {
		ba := new(basic)
		ba.Id = 1
		q, _ = GetQbs()
		q.Begin()
		q.Find(ba)
		q.Commit()
		q.Close()
	}
	b.StopTimer()
	runtime.GC()
	stats := new(runtime.MemStats)
	runtime.ReadMemStats(stats)
	fmt.Printf("alloc:%d, total:%d\n", stats.Alloc, stats.TotalAlloc)
}
