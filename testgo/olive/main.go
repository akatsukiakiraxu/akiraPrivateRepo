package main

import (
	_ "fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
	"time"
)

func main() {
	log.Println("main")
	os.Remove("./olive.db")
	var err error
	daoUnit := NewDaoUnit()
	defer daoUnit.Close()
	daoUnit.CreateTable()
	daoUnit.InitRecordState()
	daoUnit.InitUserTable()

	daoUnit.AddUser("test1", "123456")

	t := time.Now()
	var re RecordState
	re.Pallet = 12345
	re.Counter = 54321
	daoUnit.UpdateRecordState(re)

	var wa WaveData
	wa.Channel = "CH01"
	wa.Pallet = "12345"
	wa.RecordDate = time.Now()
	wa.FilePath = "/CH01/12345/CH01_12345_54321.bin"
	wa.Size = 32456
	daoUnit.InsertWaveData(wa)

	wa.Channel = "CH01"
	wa.Pallet = "12345"
	wa.RecordDate = time.Now()
	wa.FilePath = "/CH01/12345/CH01_12345_54322.bin"
	wa.Size = 32456
	daoUnit.InsertWaveData(wa)

	daoUnit.ShowAll()

	err = daoUnit.Login("aaa", "bbb")
	log.Println(err)
	err = daoUnit.Login("test1", "123456")
	log.Println(err)

	re = daoUnit.GetRecordState()
	log.Println(re)

	var dateSection DateSection
	waveDataList, err := daoUnit.GetWaveData("CH01", "*", dateSection)
	log.Println(waveDataList)

	waveDataList, err = daoUnit.GetWaveData("CH02", "*", dateSection)
	log.Println(waveDataList)

	waveDataList, err = daoUnit.GetWaveData("*", "12345", dateSection)
	log.Println(waveDataList)

	waveDataList, err = daoUnit.GetWaveData("", "", dateSection)
	log.Println(waveDataList)

	dateSection.Start = t.Format(timelayout)
	dateSection.End = t.Add(time.Minute).Format(timelayout)
	waveDataList, err = daoUnit.GetWaveData("CH01", "12345", dateSection)
	log.Println(waveDataList)

	dateSection.Start = t.Add(-time.Minute).Format(timelayout)
	waveDataList, err = daoUnit.GetWaveData("", "*", dateSection)
	log.Println(waveDataList)

	dateSection.End = t.Add(-time.Minute).Format(timelayout)
	waveDataList, err = daoUnit.GetWaveData("*", "*", dateSection)
	log.Println(waveDataList)

	dateSection.Start = t.Add(time.Minute).Format(timelayout)
	dateSection.End = t.Add(time.Minute * 2).Format(timelayout)
	waveDataList, err = daoUnit.GetWaveData("*", "", dateSection)
	log.Println(waveDataList)

	waveDataList, err = daoUnit.GetWaveData("CH1", "", dateSection)
	waveDataList, err = daoUnit.GetWaveData("", "12345", dateSection)

	dateSection.Start = ""
	dateSection.End = t.Format(timelayout)
	waveDataList, err = daoUnit.GetWaveData("", "", dateSection)

	dateSection.Start = t.Format(timelayout)
	dateSection.End = ""
	waveDataList, err = daoUnit.GetWaveData("", "", dateSection)

	dateSection.Start = ""
	dateSection.End = ""
	waveDataList, err = daoUnit.GetWaveData("", "", dateSection)
}
