package main

import (
	"fmt"
	"github.com/mattn/go-rl"
	"github.com/mattn/go-shellwords"
	"os"
	"os/exec"
	"path/filepath"
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
				files, _ := filepath.Glob(filepath.FromSlash(v) + "*")
				if len(files) > 0 {
					for i, v := range files {
						files[i] = strings.Replace(filepath.ToSlash(v), ` `, `\ `, -1)
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
			fmt.Fprintln(os.Stderr, err)
			break
		}
		words, err := shellwords.Parse(string(b))
		if len(words) == 0 {
			continue
		}

		for i, v := range words {
			words[i] = filepath.ToSlash(v)
		}

		if words[0] == "exit" {
			break
		}

		if len(words) >= 2 && words[0] == "cd" {
			err = os.Chdir(filepath.Clean(strings.Join(words[1:], " ")))
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		} else {
			cmd := exec.Command(words[0], words[1:]...)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Run()
			if err != nil && cmd.Process == nil {
				fmt.Fprintln(os.Stderr, err)
			}
		}
	}
}
