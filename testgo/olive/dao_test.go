package main

import (
	_ "fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
	"path/filepath"
	"testing"
	_ "testing"
)

type AddUserTestCase struct {
	Name     string
	Password string
	Auth     string
}

func TestDaoUnit_AddUser(test *testing.T) {
	os.Remove("./olive.db")
	daoUnit := NewDaoUnit()
	defer daoUnit.Close()
	daoUnit.CreateTable()
	daoUnit.InitRecordState()
	cases := []struct {
		in       AddUserTestCase
		expected error
	}{
		{AddUserTestCase{Name: "admin", Password: "123456", Auth: "admin"}, nil},
		{AddUserTestCase{Name: "test1", Password: "123456", Auth: "user"}, nil},
	}
	for _, c := range cases {
		actual := daoUnit.AddUser(c.in.Name, c.in.Password, c.in.Auth)
		if actual != c.expected {
			test.Errorf("AddUser(%q, %q, %q) == %q, expect %q",
				c.in.Name, c.in.Password, c.in.Auth, actual, c.expected)
		}
	}

}
