// _Interfaces_ are named collections of method
// signatures.

package main

import (
	"./bar"
	"./foo"
)

func main() {
	handle := foo.NewFoo()
	handle.Get()
	handle.Set()

	handle2 := bar.NewBar()
	handle2.Get()
	handle2.Set()
}
