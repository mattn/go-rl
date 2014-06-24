package rl

import (
	"os"
	"os/signal"
)

func ReadLine(prompt string) (string, error) {
	c, err := NewRl(prompt)
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
loop:
	for !quit {
		err = c.redraw(dirty)
		if err != nil {
			return "", err
		}
		dirty = false

		rs, err := c.readRunes()
		if err != nil {
			break
		}
		for _, r := range rs {
			switch r {
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
				tmp = append(tmp, r)
				c.input = append(tmp, c.input[c.cursor_x:len(c.input)]...)
				c.cursor_x++
				dirty = true
			}
		}
	}
	os.Stdout.WriteString("\n")

	return string(c.input), nil
}
