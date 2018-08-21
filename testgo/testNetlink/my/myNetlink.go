package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"syscall"
	//    "net/url"
	"time"
)

const NETLINK_KOBJECT_UEVENT = 15
const UEVENT_BUFFER_SIZE = 2048

func parseUEventBuffer(arr []byte) (devName, devType, act string) {
	j := 0
	for i := 0; i < len(arr)+1; i++ {
		if i == len(arr) || arr[i] == 0 {
			str := string(arr[j:i])
			a := strings.Split(str, "=")
			if len(a) == 2 {
				log.Println(a[0], "=", a[1])
				switch a[0] {
				case "DEVNAME":
					devName = a[1]
				case "DEVTYPE":
					devType = a[1]
				case "ACTION":
					act = a[1]
				}
			}
			j = i + 1
		}
	}
	return
}

func storageWatcher() {
	fd, err := syscall.Socket(
		syscall.AF_NETLINK, syscall.SOCK_RAW,
		NETLINK_KOBJECT_UEVENT,
	)
	if err != nil {
		log.Println(err)
		return
	}

	nl := syscall.SockaddrNetlink{
		Family: syscall.AF_NETLINK,
		Pid:    uint32(os.Getpid()),
		Groups: 1,
	}
	err = syscall.Bind(fd, &nl)
	if err != nil {
		log.Println(err)
		return
	}

	b := make([]byte, UEVENT_BUFFER_SIZE*2)
	for {
		syscall.Read(fd, b)
		devName, devType, act := parseUEventBuffer(b)
		if devType == "partition" {
			log.Println("devName=", devName, "act", act)
			if act == "add" {
				res, err := http.Post("http://localhost:2223/export/storage/test_install", "application/x-www-form-urlencoded", nil)
				if err == nil {
					message := make([]byte, 128)
					res.Body.Read(message)
					log.Println(string(message))
				} else {
					log.Println(err)
				}
			} else if act == "remove" {
				res, err := http.Post("http://localhost:2223/export/storage/test_uninstall", "application/x-www-form-urlencoded", nil)
				if err == nil {
					message := make([]byte, 128)
					res.Body.Read(message)
					log.Println(string(message))
				} else {
					log.Println(err)
				}
			} else {

			}
		}
	}
}

func main() {
	go storageWatcher()
	log.Println("storageWatcher started")
	for {
		time.Sleep(time.Second)
	}
}
