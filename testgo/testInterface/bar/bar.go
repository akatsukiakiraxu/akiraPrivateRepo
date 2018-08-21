package bar

import (
	"../it"
	"log"
)

type bar int

func (bar) Get() {
	log.Println("bar Get")
}

func (bar) Set() {
	log.Println("bar Set")
}

func NewBar() it.I {
	return bar(0)
}
