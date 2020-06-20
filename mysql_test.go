package runc_mysql

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func TestLoadFile(t *testing.T) {
	m := New()
	err := m.Load()
	if err != nil {
		t.Error(err)
	}

	err = m.Start()
	if err != nil {
		t.Error(err)
	}

	tryCount := 0
	var db *sql.DB
	for range time.Tick(time.Second) {
		tryCount += 1
		db, err = sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/test")
		if err == nil {
			break
		} else if tryCount > 45 {
			t.Error("timeout waiting sql ready")
			break
		}
	}

	_, err = db.Exec("create table test(`id` int(10) unsigned NOT NULL AUTO_INCREMENT)")
	if err != nil {
		t.Error(err)
	}

	err = m.ShutDown(true)
	if err != nil {
		t.Error(err)
	}
}
