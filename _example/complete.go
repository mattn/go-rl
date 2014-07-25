package main

import (
	"fmt"
	"github.com/mattn/go-rl"
)

func main() {
	r := rl.NewRl()
	r.CompleteFunc = func(line string, pos int) (int, []string) {
		rs := []rune(line)
		for pos > 0 {
			if rs[pos-1] == ' ' {
				println(1)
				return pos, []string{"foo", "bar"}
			}
			pos--
		}
		return -1, nil
	}
	s, err := r.ReadLine()
	fmt.Println(string(s), err)
}
