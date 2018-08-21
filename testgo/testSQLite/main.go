package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"time"
)

type Dept struct {
	DNo        string
	DName      string
	Budget     float64
	LastUpdate time.Time
}

const layout2 = "2006-01-02 15:04:05"

func main() {
	// 第2引数の形式は "user:password@tcp(host:port)/dbname"
	db, err := sql.Open("sqlite3", "./test.db?parseTime=true")
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()

	/*
		// DDL発行
		// Resultが戻されるけど, DDLの場合は使い道がなさそう
		_, err = db.Exec(
			`CREATE TABLE dept (dno VARCHAR(20) PRIMARY KEY,dname VARCHAR(20),budget NUMERIC(10,2),lastupdate DATETIME)`)
		if err != nil {
			panic(err.Error())
		}
	*/

	// 引数付きでInsert文発行
	t := time.Now()
	result, err := db.Exec(`
		      INSERT INTO dept(dno, dname, budget, lastupdate) VALUES(:DNO, :DNAME, :BUDGET, :LASTUPDATE)
		  `, sql.Named("DNO", "D2"), sql.Named("DNAME", "Development"), sql.Named("BUDGET", 20000), sql.Named("LASTUPDATE", t.Format(layout2)))
	if err != nil {
		panic(err.Error())
	}
	// 影響を与えた件数を取得
	n, err := result.RowsAffected()
	if err != nil {
		panic(err.Error())
	}
	fmt.Println(n)

	/*
			var dept Dept

			// Select文発行
			err = db.QueryRow(`
		      SELECT
		           dno
		          ,dname
		          ,budget
		          ,lastupdate
		      FROM
		          dept
		  `).Scan(&(dept.DNo), &(dept.DName), &(dept.Budget), &(dept.LastUpdate))

			if err != nil {
				panic(err)
			}

			fmt.Println(dept)
	*/

	// Select文発行
	rows, err := db.Query(`
		      SELECT
		           dno
		          ,dname
		          ,budget
		          ,lastupdate
		      FROM
		          dept
		  `)
	if err != nil {
		panic(err.Error())
	}
	defer rows.Close()

	// 1行ずつ取得
	for rows.Next() {
		var dept Dept
		err := rows.Scan(&(dept.DNo), &(dept.DName), &(dept.Budget), &(dept.LastUpdate))
		if err != nil {
			panic(err.Error())
		}
		fmt.Println(dept)
	}

	// 上のイテレーション内でエラーがあれば表示
	if err := rows.Err(); err != nil {
		panic(err.Error())
	}

}
