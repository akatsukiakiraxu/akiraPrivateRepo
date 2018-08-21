package main

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
	"log"
	"time"
)

type OliveUser struct {
	Id            string
	Username      string
	Password      string
	Authority     string
	LastLoginDate time.Time
	Folder        string
}
type WaveData struct {
	Channel    string
	Pallet     string
	FilePath   string
	RecordDate time.Time
	Size       uint64
}

type RecordState struct {
	id      uint32
	Pallet  uint32
	Counter uint32
}

type DaoUnit struct {
	db *sql.DB
}

type DateSection struct {
	Start string
	End   string
}

const timelayout = "2006-01-02 15:04:05"

func (dao *DaoUnit) CreateTable() error {
	// DDL発行
	_, err := dao.db.Exec("CREATE TABLE olive_user (id VARCHAR(38),username VARCHAR(20),password VARCHAR(128),authority VARCHAR(20), last_loging_date DATETIME, folder VARCHAR(128))")
	if err != nil {
		log.Fatal(err)
	}
	_, err = dao.db.Exec("CREATE TABLE wave_data (channel VARCHAR(10), pallet VARCHAR(10),file_path VARCHAR(128),record_date DATETIME, size INTEGER)")
	if err != nil {
		log.Fatal(err)
	}
	_, err = dao.db.Exec("CREATE TABLE record_state (id INTEGER, pallet INTEGER,counter INTEGER)")
	if err != nil {
		log.Fatal(err)
	}
	return err
}

func (dao *DaoUnit) InitRecordState() {

	tx, err := dao.db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	stmt, err := tx.Prepare("INSERT INTO record_state(id, pallet, counter) VALUES(?,?,?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	_, err = stmt.Exec(0, 0, 0)
	if err != nil {
		log.Fatal(err)
	}
	tx.Commit()
}

func (dao *DaoUnit) InitUserTable() {
	tx, err := dao.db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	stmt, err := tx.Prepare("INSERT INTO olive_user(id, username, password, authority, last_loging_date, folder) VALUES(?,?,?,?,?,?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	t := time.Now()
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("helloolive"), bcrypt.DefaultCost)
	_, err = stmt.Exec("Administrator", "oliveAdmin", string(hashedPassword), "admin", t.Format(timelayout), ("/opt/Administrator"))
	if err != nil {
		log.Fatal(err)
	}
	tx.Commit()
}

func (dao *DaoUnit) ShowAll() {
	// Select文発行
	rows, err := dao.db.Query("SELECT * FROM olive_user")
	if err != nil {
		panic(err.Error())
	}
	defer rows.Close()

	// 1行ずつ取得
	for rows.Next() {
		var user OliveUser
		err := rows.Scan(&(user.Id), &(user.Username), &(user.Password), &(user.Authority), &(user.LastLoginDate), &(user.Folder))
		if err != nil {
			panic(err.Error())
		}
		fmt.Println(user)
	}

	// Select文発行
	rows, err = dao.db.Query("SELECT * FROM record_state")
	if err != nil {
		panic(err.Error())
	}
	defer rows.Close()

	// 1行ずつ取得
	for rows.Next() {
		var recordState RecordState
		err := rows.Scan(&(recordState.id), &(recordState.Pallet), &(recordState.Counter))
		if err != nil {
			panic(err.Error())
		}
		fmt.Println(recordState)
	}

	// Select文発行
	rows, err = dao.db.Query("SELECT * FROM wave_data")
	if err != nil {
		panic(err.Error())
	}
	defer rows.Close()

	// 1行ずつ取得
	for rows.Next() {
		var wavedata WaveData
		err := rows.Scan(&(wavedata.Channel), &(wavedata.Pallet), &(wavedata.FilePath), &(wavedata.RecordDate), &(wavedata.Size))
		if err != nil {
			panic(err.Error())
		}
		fmt.Println(wavedata)
	}
	// 上のイテレーション内でエラーがあれば表示
	if err := rows.Err(); err != nil {
		panic(err.Error())
	}
}

func NewDaoUnit() *DaoUnit {
	s := DaoUnit{}
	var err error
	s.db, err = sql.Open("sqlite3", "./olive.db?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	return &s
}
func (dao *DaoUnit) Close() {
	dao.db.Close()
}

func (dao *DaoUnit) Login(username string, inputPassword string) error {
	row := dao.db.QueryRow("SELECT password FROM olive_user WHERE username=?", username)
	var password string
	var result error
	err := row.Scan(&password)
	switch {
	case err == sql.ErrNoRows:
		result = errors.New("user not found")
	case err != nil:
		result = err
	default:
		err = bcrypt.CompareHashAndPassword([]byte(password), []byte(inputPassword))
		if err != nil {
			result = errors.New("password authentication failed")
		}
	}
	return result
}

func isExistUser(db *sql.DB, username string) bool {
	var count int
	row := db.QueryRow("SELECT COUNT(*) as count FROM olive_user WHERE username=?", username)
	row.Scan(&count)
	if count > 0 {
		return true
	} else {
		return false
	}
}

func (dao *DaoUnit) GetRecordState() RecordState {
	row := dao.db.QueryRow("SELECT pallet, counter FROM record_state")
	var recordState RecordState
	row.Scan(&recordState.Pallet, &recordState.Counter)
	return recordState
}

func (dao *DaoUnit) GetWaveData(channel string, pallet string, date DateSection) ([]WaveData, error) {
	// Select文発行
	var query string
	isConditions := false
	var conditions string
	var err error
	if (channel == "" || channel == "*") && (pallet == "" || pallet == "*") && date.Start == "" && date.End == "" {
		query = "SELECT * FROM wave_data"
	} else {
		query = "SELECT * FROM wave_data WHERE "
		if channel != "" && channel != "*" {
			conditions += ("channel=\"" + channel + "\"")
			isConditions = true
		}
		if pallet != "" && pallet != "*" {
			if isConditions {
				conditions += " AND "
			}
			conditions += ("pallet=\"" + pallet + "\"")
			isConditions = true
		}
		if isConditions {
			conditions += " AND "
		}
		if date.Start == "" {
			conditions += ("record_date <= \"" + date.End + "\"")
		} else if date.End == "" {
			conditions += ("record_date >= \"" + date.Start + "\"")
		} else {
			conditions += ("record_date BETWEEN \"" + date.Start + "\"" + " AND \"" + date.End + "\"")
		}
		isConditions = true
	}
	var count int64
	getCountQuery := "SELECT COUNT(*) as count FROM wave_data"
	if isConditions {
		getCountQuery += (" WHERE " + conditions)
		query += conditions
	}
	log.Println(getCountQuery)
	log.Println(query)
	row := dao.db.QueryRow(getCountQuery)
	row.Scan(&count)
	log.Println(count)
	result := make([]WaveData, count)

	if count > 0 {
		var rows *sql.Rows
		rows, err = dao.db.Query(query)
		if err != nil {
			panic(err.Error())
		}
		defer rows.Close()
		// 1行ずつ取得
		count = 0
		for rows.Next() {
			var oneData WaveData
			err := rows.Scan(&(oneData.Channel), &(oneData.Pallet), &(oneData.FilePath), &(oneData.RecordDate), &(oneData.Size))
			if err != nil {
				panic(err.Error())
			}
			result[count] = oneData
			//fmt.Println(oneData)
			count++
		}
		// 上のイテレーション内でエラーがあれば表示
		if err := rows.Err(); err != nil {
			log.Fatalln(err.Error())
		}
	}
	return result, err
}

func (dao *DaoUnit) InsertWaveData(wavedata WaveData) error {
	tx, err := dao.db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	stmt, err := tx.Prepare("INSERT INTO wave_data(channel, pallet, file_path, record_date,size) VALUES(?,?,?,?,?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	_, err = stmt.Exec(wavedata.Channel, wavedata.Pallet, wavedata.FilePath, wavedata.RecordDate.Format(timelayout), wavedata.Size)
	if err != nil {
		log.Fatal(err)
	}
	tx.Commit()
	return err
}

func (dao *DaoUnit) AddUser(name string, pwd string) error {
	if isExistUser(dao.db, name) {
		return errors.New("user duplicated")
	}
	tx, err := dao.db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	stmt, err := tx.Prepare("INSERT INTO olive_user(id, username, password, authority, last_loging_date, folder) VALUES(?,?,?,?,?,?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	t := time.Now()
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	_, err = stmt.Exec(uuid.New(), name, string(hashedPassword), "user", t.Format(timelayout), ("/opt/" + name))
	if err != nil {
		log.Fatal(err)
	}
	tx.Commit()
	return err
}

func (dao *DaoUnit) UpdateRecordState(recordState RecordState) error {
	tx, err := dao.db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	stmt, err := tx.Prepare("UPDATE record_state SET pallet=?, counter=? WHERE id=0")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	_, err = stmt.Exec(recordState.Pallet, recordState.Counter)
	if err != nil {
		log.Fatal(err)
	}
	tx.Commit()
	return err
}
