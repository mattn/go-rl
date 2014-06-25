// +build !windows

package rl

import (
	"bytes"
	"fmt"
	"github.com/mattn/go-runewidth"
	"io"
	"os"
	"syscall"
	"unsafe"
)

type ctx struct {
	in           uintptr
	out          uintptr
	st           syscall.Termios
	input        []rune
	prompt       string
	cursor_x     int
	old_row     int
	old_crow     int
	size         int
}

func (c *ctx) readRunes() ([]rune, error) {
	var buf [16]byte
	n, err := syscall.Read(int(c.in), buf[:])
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return []rune{}, nil
	}
	if buf[n-1] == '\n' {
		n--
	}
	return []rune(string(buf[:n])), nil
}

func NewRl(prompt string) (*ctx, error) {
	c := new(ctx)

	c.in = os.Stdin.Fd()

	var st syscall.Termios
	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, c.in, uintptr(TCGETS), uintptr(unsafe.Pointer(&st)), 0, 0, 0); err != 0 {
		return nil, err
	}

	c.st = st

	st.Iflag &^= syscall.ISTRIP | syscall.INLCR | syscall.ICRNL | syscall.IGNCR | syscall.IXON | syscall.IXOFF
	st.Lflag &^= syscall.ECHO | syscall.ICANON | syscall.ISIG
	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, c.in, uintptr(TCSETS), uintptr(unsafe.Pointer(&st)), 0, 0, 0); err != 0 {
		return nil, err
	}

	c.prompt = prompt
	c.input = []rune{}

	c.size = 80
	return c, nil
}

func (c *ctx) tearDown() {
	syscall.Syscall6(syscall.SYS_IOCTL, c.in, uintptr(TCSETS), uintptr(unsafe.Pointer(&c.st)), 0, 0, 0)
}

func (c *ctx) redraw(dirty bool) error {
	var buf bytes.Buffer

	//buf.WriteString("\x1b[>5h")

	buf.WriteString("\x1b[0G")
	if dirty {
		buf.WriteString("\x1b[0K")
	}
	for i := 0; i <  c.old_row - c.old_crow; i++ {
		buf.WriteString("\x1b[B")
	}
	for i := 0; i <  c.old_row; i++ {
		if dirty {
			buf.WriteString("\x1b[2K")
		}
		buf.WriteString("\x1b[A")
	}

	ccol, crow, col, row := -1, 0, 0, 0
	plen := len([]rune(c.prompt))
	for i, r := range []rune(c.prompt + string(c.input)) {
		if i == plen + c.cursor_x {
			ccol = col
			crow = row
		}
		rw := runewidth.RuneWidth(r)
		if col + rw > c.size {
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
		for i := 0; i <  row; i++ {
			buf.WriteString("\x1b[A")
		}
	}
	if ccol == -1 {
		ccol = col
		crow = row
	}
	for i := 0; i <  crow; i++ {
		buf.WriteString("\x1b[B")
	}
	buf.WriteString(fmt.Sprintf("\x1b[%dG", ccol + 1))

	//buf.WriteString("\x1b[>5l")
	io.Copy(os.Stdout, &buf)
	os.Stdout.Sync()

	c.old_row = row
	c.old_crow = crow

	return nil
}
