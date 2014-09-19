package main

import (
	"fmt"
	"github.com/mattn/go-rl"
)

func main() {
	for {
		b, err := rl.ReadLine("> ")
		if err != nil {
			break
		}
		s := string(b)
		if s == "quit" {
			break
		}
		fmt.Println("Hello:", s)
	}
}
