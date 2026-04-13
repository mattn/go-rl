package rl

import (
	"io"
	"os"
	"os/signal"
)

type Rl struct {
	Prompt       string
	PasswordRune rune
	EOFOnCtrlD   bool
	CompleteFunc func(string, int) (int, []string)
}

func commonPrefix(words []string) string {
	if len(words) == 0 {
		return ""
	}

	prefix := []rune(words[0])
	for _, word := range words[1:] {
		rs := []rune(word)
		n := len(prefix)
		if len(rs) < n {
			n = len(rs)
		}

		i := 0
		for i < n && prefix[i] == rs[i] {
			i++
		}
		prefix = prefix[:i]
		if len(prefix) == 0 {
			return ""
		}
	}

	return string(prefix)
}

func NewRl() *Rl {
	return &Rl{Prompt: "> ", PasswordRune: '*'}
}

func shouldReturnEOFOnCtrlD(input []rune, eofOnCtrlD bool) bool {
	return eofOnCtrlD || len(input) == 0
}

func applyCompletion(input []rune, completePos, cursor int, candidates []string) ([]rune, int, bool) {
	if len(candidates) == 0 || completePos < 0 || completePos > cursor || cursor > len(input) {
		return input, 0, false
	}

	item := commonPrefix(candidates)
	if item == "" {
		return input, 0, false
	}

	tmp := make([]rune, 0, completePos+len([]rune(item))+len(input)-cursor)
	tmp = append(tmp, input[:completePos]...)
	tmp = append(tmp, []rune(item)...)
	tmp = append(tmp, input[cursor:]...)
	return tmp, completePos + len([]rune(item)), true
}

func deleteWordBeforeCursor(input []rune, cursor int) ([]rune, int, bool) {
	if cursor <= 0 || cursor > len(input) {
		return input, cursor, false
	}

	start := cursor
	for start > 0 && (input[start-1] == ' ' || input[start-1] == '\t') {
		start--
	}
	for start > 0 && input[start-1] != ' ' && input[start-1] != '\t' {
		start--
	}
	if start == cursor {
		return input, cursor, false
	}

	out := append(input[:start], input[cursor:]...)
	return out, start, true
}

func insertRune(input []rune, cursor int, r rune) ([]rune, int) {
	tmp := make([]rune, 0, len(input)+1)
	tmp = append(tmp, input[:cursor]...)
	tmp = append(tmp, r)
	tmp = append(tmp, input[cursor:]...)
	return tmp, cursor + 1
}

func deleteRuneBeforeCursor(input []rune, cursor int) ([]rune, int, bool) {
	if cursor <= 0 || cursor > len(input) {
		return input, cursor, false
	}

	out := append(input[:cursor-1], input[cursor:]...)
	return out, cursor - 1, true
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
	defer signal.Stop(sc)
	go func() {
		<-sc
		c.input = nil
		quit = true
	}()

	passwordRune := rune(0)
	if passwordInput {
		passwordRune = r.PasswordRune
	}

	dirty := true
loop:
	for !quit {
		if err := c.redraw(dirty, passwordRune); err != nil {
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
				if !shouldReturnEOFOnCtrlD(c.input, r.EOFOnCtrlD) {
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
				var ok bool
				c.input, c.cursor_x, ok = deleteRuneBeforeCursor(c.input, c.cursor_x)
				if ok {
					dirty = true
				}
			case 9: // TAB
				if r.CompleteFunc == nil {
					continue
				}
				completePos, candidates := r.CompleteFunc(string(c.input), c.cursor_x)
				var ok bool
				c.input, c.cursor_x, ok = applyCompletion(c.input, completePos, c.cursor_x, candidates)
				if ok {
					dirty = true
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
				var ok bool
				c.input, c.cursor_x, ok = deleteWordBeforeCursor(c.input, c.cursor_x)
				if ok {
					dirty = true
				}
			default:
				c.input, c.cursor_x = insertRune(c.input, c.cursor_x, rc)
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
