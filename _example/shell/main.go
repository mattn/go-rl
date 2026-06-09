package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mattn/go-rl"
	"github.com/mattn/go-shellwords"
)

func completionStart(rs []rune, cursor int) int {
	for start := cursor; start >= 0; start-- {
		if start == 0 || rs[start-1] == ' ' && (start == 1 || rs[start-2] != '\\') {
			return start
		}
	}
	return -1
}

func completePath(line string, cursor int) (int, []string) {
	rs := []rune(line)
	start := completionStart(rs, cursor)
	if start < 0 {
		return -1, nil
	}

	fragment := strings.ReplaceAll(string(rs[start:cursor]), `\ `, ` `)
	files, _ := filepath.Glob(filepath.FromSlash(fragment) + "*")
	if len(files) == 0 {
		return cursor, []string{}
	}

	candidates := make([]string, len(files))
	for i, path := range files {
		display := filepath.ToSlash(path)
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			display += "/"
		}
		candidates[i] = strings.ReplaceAll(display, ` `, `\ `)
	}
	return start, candidates
}

func main() {
	r := rl.NewRl()
	r.CompleteFunc = func(line string, pos int) (int, []string) {
		start, candidates := completePath(line, pos)
		if len(candidates) > 1 {
			fmt.Printf("\n%v\n", candidates)
		}
		return start, candidates
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
			continue
		}

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
