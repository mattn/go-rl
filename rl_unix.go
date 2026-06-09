//go:build !windows
// +build !windows

package rl

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
	"golang.org/x/sys/unix"
)

type ctx struct {
	in       uintptr
	out      uintptr
	st       unix.Termios
	input    []rune
	last     []rune
	prompt   string
	cursor_x int
	old_row  int
	old_crow int
	size     int
	pending  []byte
}

func (c *ctx) readRunes() ([]rune, error) {
	var buf [16]byte
	n, err := unix.Read(int(c.in), buf[:])
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return []rune{}, nil
	}

	rs, pending := decodeRunes(append(c.pending, buf[:n]...))
	c.pending = pending
	return rs, nil
}

func decodeRunes(buf []byte) ([]rune, []byte) {
	if len(buf) == 0 {
		return []rune{}, nil
	}

	var rs []rune
	i := 0
	for i < len(buf) {
		if buf[i] == '\n' {
			i++
			continue
		}
		r, size := utf8.DecodeRune(buf[i:])
		if r == utf8.RuneError && size == 1 {
			if !utf8.FullRune(buf[i:]) {
				return rs, append([]byte(nil), buf[i:]...)
			}
		}
		rs = append(rs, r)
		i += size
	}

	return rs, nil
}

func ioctlGetTermios(fd uintptr, req uint, st *unix.Termios) error {
	termios, err := unix.IoctlGetTermios(int(fd), req)
	if err != nil {
		return err
	}
	*st = *termios
	return nil
}

func ioctlSetTermios(fd uintptr, req uint, st *unix.Termios) error {
	return unix.IoctlSetTermios(int(fd), req, st)
}

func newCtx(prompt string) (*ctx, error) {
	c := new(ctx)

	c.in = os.Stdin.Fd()

	var st unix.Termios
	if err := ioctlGetTermios(c.in, uint(TCGETS), &st); err != nil {
		return nil, err
	}

	c.st = st

	st.Iflag &^= unix.ISTRIP | unix.INLCR | unix.ICRNL | unix.IGNCR | unix.IXON | unix.IXOFF
	st.Lflag &^= unix.ECHO | unix.ICANON | unix.ISIG
	if err := ioctlSetTermios(c.in, uint(TCSETS), &st); err != nil {
		return nil, err
	}

	c.prompt = prompt
	c.input = []rune{}

	ws, err := unix.IoctlGetWinsize(int(c.in), unix.TIOCGWINSZ)
	if err != nil {
		return nil, err
	}
	c.size = int(ws.Col)
	return c, nil
}

func (c *ctx) tearDown() {
	ioctlSetTermios(c.in, uint(TCSETS), &c.st)
}

func (c *ctx) redraw(dirty bool, passwordChar rune) error {
	var buf bytes.Buffer

	//buf.WriteString("\x1b[>5h")

	buf.WriteString("\x1b[0G")
	if dirty {
		buf.WriteString("\x1b[0K")
	}
	for i := 0; i < c.old_row-c.old_crow; i++ {
		buf.WriteString("\x1b[B")
	}
	for i := 0; i < c.old_row; i++ {
		if dirty {
			buf.WriteString("\x1b[2K")
		}
		buf.WriteString("\x1b[A")
	}

	var rs []rune
	if passwordChar != 0 {
		for i := 0; i < len(c.input); i++ {
			rs = append(rs, passwordChar)
		}
	} else {
		rs = c.input
	}

	ccol, crow, col, row := -1, 0, 0, 0
	plen := len([]rune(c.prompt))
	for i, r := range []rune(c.prompt + string(rs)) {
		if i == plen+c.cursor_x {
			ccol = col
			crow = row
		}
		rw := runewidth.RuneWidth(r)
		if col+rw > c.size {
			col = 0
			row++
			if dirty {
				buf.WriteString("\n\r\x1b[0K")
			}
		}
		if dirty {
			buf.WriteString(string(r))
		}
		col += rw
	}
	if dirty {
		buf.WriteString("\x1b[0G")
		for i := 0; i < row; i++ {
			buf.WriteString("\x1b[A")
		}
	}
	if ccol == -1 {
		ccol = col
		crow = row
	}
	for i := 0; i < crow; i++ {
		buf.WriteString("\x1b[B")
	}
	buf.WriteString(fmt.Sprintf("\x1b[%dG", ccol+1))

	//buf.WriteString("\x1b[>5l")
	io.Copy(os.Stdout, &buf)
	os.Stdout.Sync()

	c.old_row = row
	c.old_crow = crow

	return nil
}
