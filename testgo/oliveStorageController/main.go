package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
)

type storageInformation struct {
	Action          *string `json:"action"`
	Uuid            *string `json:"uuid"`
	MountPoint      *string `json:"mount_point"`
	PartEntryNumber *uint64 `json:"part_entry_number"`
}

type oliveStorageControl struct {
	StorageControl map[string]storageInformation `json:"storage_control"`
}

const storageFile = "storage.json"

func main() {
	action := flag.String("action", "", "")
	uuid := flag.String("uuid", "", "")
	mountPoint := flag.String("mount_point", "", "")
	partNum := flag.Uint64("part_entry_number", 0, "")
	flag.Parse()
	log.Println(*action, *uuid, *mountPoint, *partNum)

	if _, err := os.Stat(storageFile); os.IsNotExist(err) {
		if *action == "add" {
			s := new(oliveStorageControl)
			s.StorageControl = make(map[string]storageInformation)
			s.StorageControl[*uuid] = storageInformation{
				Action:          action,
				Uuid:            uuid,
				MountPoint:      mountPoint,
				PartEntryNumber: partNum,
			}
			jsonBuffer, _ := json.Marshal(s)
			err = ioutil.WriteFile(storageFile, jsonBuffer, 0644)
		} else {
			return
		}
	} else {
		b, err := ioutil.ReadFile(storageFile)
		if err != nil {
			return
		}
		s := new(oliveStorageControl)
		if err = json.Unmarshal(b, s); err != nil {
			return
		}
		if *action == "add" {
			if _, ok := s.StorageControl[*uuid]; ok {
				return
			} else {
				s.StorageControl[*uuid] = storageInformation{
					Action:          action,
					Uuid:            uuid,
					MountPoint:      mountPoint,
					PartEntryNumber: partNum,
				}
			}
		} else if *action == "remove" {
			if _, ok := s.StorageControl[*uuid]; ok {
				delete(s.StorageControl, *uuid)
			} else {
				return
			}
		} else {
			return
		}
		jsonBuffer, _ := json.Marshal(s)
		err = ioutil.WriteFile(storageFile, jsonBuffer, 0644)
	}
}
