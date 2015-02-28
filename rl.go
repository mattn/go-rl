package rl

import (
	"io"
	"os"
	"os/signal"
)

type Rl struct {
	Prompt            string
	PasswordRune      rune
	CompleteFunc      func(string, int) (int, []string)
	completePos       int
	completeCandidate []string
}

// Count the number of common initial characters
func countCommonPrefixLength(words []string) int {
	pos := 0
outer:
	for ; ; pos++ {
		if pos >= len(words[0]) {
			break outer
		}
		ch := words[0][pos]
		for _, word := range words[1:] {
			if pos >= len(word) {
				break outer
			}
			if word[pos] != ch {
				break outer
			}
		}
	}
	return pos
}

func NewRl() *Rl {
	return &Rl{Prompt: "> ", PasswordRune: '*'}
}

func (r *Rl) readLine(passwordInput bool) (string, error) {
	c, err := newCtx(r.Prompt)
	if err != nil {
		return "", err
	}
	defer c.tearDown()

	quit := false
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt)
	go func() {
		<-sc
		c.input = nil
		quit = true
	}()

	dirty := true
	var passwordRune rune
	if passwordInput {
		passwordRune = r.PasswordRune
	}
loop:
	for !quit {
		err = c.redraw(dirty, passwordRune)
		if err != nil {
			return "", err
		}
		dirty = false

		rs, err := c.readRunes()
		if err != nil {
			break
		}
		for _, rc := range rs {
			switch rc {
			case 0:
			case 1: // CTRL-A
				c.cursor_x = 0
			case 2: // CTRL-B
				if c.cursor_x > 0 {
					c.cursor_x--
				}
			case 3: // BREAK
				return "", nil
			case 4: // CTRL-D
				if len(c.input) > 0 {
					continue
				}
				return "", io.EOF
			case 5: // CTRL-E
				c.cursor_x = len(c.input)
			case 6: // CTRL-F
				if c.cursor_x < len(c.input) {
					c.cursor_x++
				}
			case 8, 0x7F: // BS
				if c.cursor_x > 0 {
					c.input = append(c.input[0:c.cursor_x-1], c.input[c.cursor_x:len(c.input)]...)
					c.cursor_x--
					dirty = true
				}
			case 9: // TAB
				if r.CompleteFunc != nil {
					r.completePos, r.completeCandidate = r.CompleteFunc(string(c.input), c.cursor_x)
					if len(r.completeCandidate) > 0 {
						common := countCommonPrefixLength(r.completeCandidate)
						item := r.completeCandidate[0][0:common]
						tmp := []rune{}
						tmp = append(tmp, c.input[0:r.completePos]...)
						tmp = append(tmp, []rune(item)...)
						c.input = tmp
						dirty = true
						c.cursor_x = r.completePos + common
					}
				}
			case 10: // LF
				break loop
			case 11: // CTRL-K
				c.input = c.input[:c.cursor_x]
				dirty = true
			case 12: // CTRL-L
				dirty = true
			case 13: // CR
				break loop
			case 21: // CTRL-U
				c.input = c.input[c.cursor_x:]
				c.cursor_x = 0
				dirty = true
			case 23: // CTRL-W
				for i := len(c.input) - 1; i >= 0; i-- {
					if i == 0 || c.input[i] == ' ' || c.input[i] == '\t' {
						c.input = append(c.input[:i], c.input[c.cursor_x:]...)
						c.cursor_x = i
						dirty = true
						break
					}
				}
			default:
				tmp := []rune{}
				tmp = append(tmp, c.input[0:c.cursor_x]...)
				tmp = append(tmp, rc)
				c.input = append(tmp, c.input[c.cursor_x:len(c.input)]...)
				c.cursor_x++
				dirty = true
			}
		}
	}
	os.Stdout.WriteString("\n")

	return string(c.input), nil
}

func (r *Rl) ReadLine() (string, error) {
	return r.readLine(false)
}

func (r *Rl) ReadPassword() (string, error) {
	return r.readLine(true)
}

func ReadLine(prompt string) (string, error) {
	r := NewRl()
	r.Prompt = prompt
	return r.readLine(false)
}

func ReadPassword(prompt string) (string, error) {
	r := NewRl()
	r.Prompt = prompt
	r.PasswordRune = '*'
	return r.readLine(true)
}
