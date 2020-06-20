package main

import (
	"database/sql"
	"log"
	"os/user"
	"time"

	_ "github.com/go-sql-driver/mysql"
	mysql "github.com/u2takey/runc-mysql"
)

var testSql = `
CREATE TABLE IF NOT EXISTS test(
   id INT AUTO_INCREMENT,
   name VARCHAR(64) NOT NULL,
   PRIMARY KEY ( id )
)ENGINE=InnoDB DEFAULT CHARSET=utf8;
`

func main() {
	m := mysql.New()
	var err error
	//err = m.Load()
	//if err != nil {
	//	t.Error(err)
	//}
	u, err := user.Current()
	if err != nil {
		panic(err)
	}
	log.Println(u)
	m.SetDir("/tmp/mysql")
	log.Println("staring")
	go func() {
		err = m.Start()
		if err != nil {
			log.Fatal(err)
		}
	}()

	tryCount := 0
	var db *sql.DB
	for range time.Tick(time.Second) {
		tryCount += 1
		if tryCount > 45 {
			log.Fatal("timeout waiting sql ready")
		}
		db, err = sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/test")
		if err == nil {
			_, err = db.Exec("show databases")
			if err == nil {
				break
			}
		}
	}

	_, err = db.Exec(testSql)
	if err != nil {
		log.Println(err)
	}

	err = m.ShutDown(true)
	if err != nil {
		log.Fatal(err)
	}
}
