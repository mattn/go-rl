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
	cursor_x     int
	prompt       string
	old_cursor_x int
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

	buf.WriteString("\x1b[>5h")

	ccols, crows, cols, rows := 0, 0, 0, 0
	width := runewidth.StringWidth(c.prompt + string(c.input[:c.old_cursor_x]))
	if dirty {
		buf.WriteString("\x1b[0G")
		buf.WriteString("\x1b[2K")
		for i := 0; i <  width / c.size; i++ {
			buf.WriteString("\x1b[2K\x1b[A")
		}
		plen := len([]rune(c.prompt))
		for i, r := range []rune(c.prompt + string(c.input)) {
			if i == plen + c.cursor_x {
				ccols = cols
				crows = rows
			}
			rw := runewidth.RuneWidth(r)
			if cols + rw > c.size {
				cols = 0
				rows++
				buf.WriteString("\n")
			}
			buf.WriteString(string(r))
			cols += rw
		}
	} else {
		buf.WriteString("\x1b[0G")
		for i := 0; i <  width / c.size; i++ {
			buf.WriteString("\x1b[A")
		}
		plen := len([]rune(c.prompt))
		for i, r := range []rune(c.prompt + string(c.input)) {
			if i == plen + c.cursor_x {
				ccols = cols
				crows = rows
			}
			rw := runewidth.RuneWidth(r)
			if cols + rw > c.size {
				cols -= c.size
				rows++
			}
			cols += rw
		}
	}
	if ccols == 0 {
		ccols = cols
		crows = rows
	}
	for i := 0; i <  crows - rows; i++ {
		buf.WriteString("\x1b[A")
	}
	buf.WriteString(fmt.Sprintf("\x1b[%dG", ccols + 1))

	buf.WriteString("\x1b[>5l")
	io.Copy(os.Stdout, &buf)
	os.Stdout.Sync()

	c.old_cursor_x = c.cursor_x

	return nil
}
