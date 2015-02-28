package main

import (
	"fmt"
	"github.com/mattn/go-rl"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	r := rl.NewRl()
	r.CompleteFunc = func(line string, pos int) (int, []string) {
		rs := []rune(line)
		start := pos
		for pos >= 0 {
			if pos == 0 || pos > 0 && rs[pos-1] == ' ' && (pos == 1 || rs[pos-2] != '\\') {
				v := strings.Replace(string(rs[pos:]), `\ `, ` `, -1)
				if runtime.GOOS == "windows" {
					v = strings.Replace(v, `/`, `\`, -1)
				}
				files, _ := filepath.Glob(v + "*")
				if len(files) > 0 {
					for i, v := range files {
						if runtime.GOOS == "windows" {
							v = strings.Replace(v, `\`, `/`, -1)
						}
						files[i] = strings.Replace(v, ` `, `\ `, -1)
					}
					if len(files) == 1 {
						files = []string{files[0] + "/"}
					} else {
						fmt.Printf("\n%v\n", files)
					}
					return pos, files
				} else {
					return start, []string{}
				}
			}
			pos--
		}
		return -1, nil
	}

	for {
		b, err := r.ReadLine()
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
