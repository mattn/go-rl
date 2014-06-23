package main

import (
	"fmt"
	"github.com/mattn/go-rl"
)

func main() {
	s, err := rl.ReadLine("> ")
	fmt.Println(string(s), err)
}
