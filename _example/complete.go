package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mattn/go-rl"
)

func completionStart(rs []rune, cursor int) int {
	for start := cursor; start >= 0; start-- {
		if start == 0 || rs[start-1] == ' ' && (start == 1 || rs[start-2] != '\\') {
			return start
		}
	}
	return -1
}

func globPattern(fragment string) string {
	fragment = strings.ReplaceAll(fragment, `\ `, ` `)
	if runtime.GOOS == "windows" {
		fragment = strings.ReplaceAll(fragment, `/`, `\`)
	}
	return fragment + "*"
}

func formatCandidate(path string) string {
	display := path
	if runtime.GOOS == "windows" {
		display = strings.ReplaceAll(display, `\`, `/`)
	}
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		display += "/"
	}
	return strings.ReplaceAll(display, ` `, `\ `)
}

func completePath(line string, cursor int) (int, []string) {
	rs := []rune(line)
	start := completionStart(rs, cursor)
	if start < 0 {
		return -1, nil
	}

	files, _ := filepath.Glob(globPattern(string(rs[start:cursor])))
	if len(files) == 0 {
		return cursor, []string{}
	}

	candidates := make([]string, len(files))
	for i, path := range files {
		candidates[i] = formatCandidate(path)
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
			break
		}
		s := string(b)
		if s == "quit" {
			break
		}
		fmt.Println("Hello:", s)
	}
}
