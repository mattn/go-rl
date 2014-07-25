package rl

import (
	"os"
	"os/signal"
)

type Rl struct {
	Prompt       string
	PasswordRune rune
	CompleteFunc func(string, int) (int, []string)
	completePos int
	completeIndex int
	completeCandidate []string
}

func NewRl() *Rl {
	return &Rl{Prompt:"> ", PasswordRune:'*', completeIndex: -1}
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
				if r.completeIndex == -1 && r.CompleteFunc != nil {
					r.completePos, r.completeCandidate = r.CompleteFunc(string(c.input), c.cursor_x)
					if r.completePos >= 0 {
						r.completeIndex = 0
					}
				}
				if r.completeIndex >= 0 {
					var item string
					if r.completeIndex >= len(r.completeCandidate) {
						r.completeIndex = 0
					} else {
						item = r.completeCandidate[r.completeIndex]
						r.completeIndex++
					}
					tmp := []rune{}
					tmp = append(tmp, c.input[0:r.completePos]...)
					tmp = append(tmp, []rune(item)...)
					c.input = tmp
					dirty = true
					c.cursor_x = r.completePos + len(item)
				}
			case 10: // LF
				break loop
			case 11: // CTRL-K
				c.input = c.input[:c.cursor_x]
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
			if rc != 9 && dirty {
				r.completeIndex = -1
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
