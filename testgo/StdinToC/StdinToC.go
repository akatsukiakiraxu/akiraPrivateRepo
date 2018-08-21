package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

func main() {
	subProcess := exec.Command("/home/share/temp/testCxx/StdinFromGo/StdinFromGo")
	stdin, err := subProcess.StdinPipe()
	if err != nil {
		fmt.Println(err)
	}
	defer stdin.Close()
	subProcess.Stdout = os.Stdout
	subProcess.Stderr = os.Stderr
	fmt.Println("START")
	if err = subProcess.Start(); err != nil {
		fmt.Println("An error occured: ", err)
	}
	io.WriteString(stdin, "hoge\n")
	fmt.Println("END")
}
