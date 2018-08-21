package foo

import (
	"../it"
	"log"
)

type foo int

func (foo) Get() {
	log.Println("foo Get")
}

func (foo) Set() {
	log.Println("foo Set")
}
func NewFoo() it.I {
	return foo(0)
}
