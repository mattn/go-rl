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
	in       uintptr
	out      uintptr
	st       syscall.Termios
	input    []rune
	cursor_x int
	prompt   string
	size     int
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

	return c, nil
}

func (c *ctx) tearDown() {
	syscall.Syscall6(syscall.SYS_IOCTL, c.in, uintptr(TCSETS), uintptr(unsafe.Pointer(&c.st)), 0, 0, 0)
}

func (c *ctx) redraw(dirty bool) error {
	var buf bytes.Buffer
	buf.WriteString("\x1b[>5h")
	buf.WriteString("\r")
	if dirty {
		buf.WriteString("\x1b[2K")
		buf.WriteString(c.prompt + string(c.input))
		buf.WriteString("\r")
	}
	x := runewidth.StringWidth(c.prompt) + runewidth.StringWidth(string(c.input[:c.cursor_x]))
	buf.WriteString(fmt.Sprintf("\x1b[%dC", x))
	buf.WriteString("\x1b[>5l")
	io.Copy(os.Stdout, &buf)
	os.Stdout.Sync()
	return nil
}
