package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"time"
)

type Dept struct {
	DNo        string
	DName      string
	Budget     float64
	LastUpdate time.Time
}

func main() {
	// 第2引数の形式は "user:password@tcp(host:port)/dbname"
	db, err := sql.Open("mysql", "fixstars:fixstars@/first_test?parseTime=true")
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

	/*
			// Insert文発行
			// Resultが戻される
			result, err := db.Exec(`
		      INSERT INTO dept(dno, dname, budget) VALUES('D1', 'Marketing', 10000)
		  `)
			if err != nil {
				panic(err.Error())
			}

			// AutoIncrementの型で使える
			// 最後に挿入したキーを返す(が, 今回は主キーをAutoIncrementにしていないので使えない.例を誤った感)
			id, err := result.LastInsertId()
			if err != nil {
				panic(err.Error())
			}
			fmt.Println(id)

			// 影響を与えた行数を返す
			n, err := result.RowsAffected()
			if err != nil {
				panic(err.Error())
			}
			fmt.Println(n)
	*/

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
	query := "SELECT dno, dname,budget,lastupdate FROM dept"
	rows, err := db.Query(query)
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
