// +build !windows

package rl

import (
	"fmt"
	"github.com/mattn/go-runewidth"
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
	ch       chan rune
	size     int
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

	go func() {
		for {
			var buf [16]byte
			n, err := syscall.Read(int(c.in), buf[:])
			if err != nil || c.ch == nil {
				break
			}
			if n == 0 {
				continue
			}
			if buf[n-1] == '\n' {
				n--
			}
			for _, r := range []rune(string(buf[:n])) {
				c.ch <- r
			}
		}
	}()

	c.prompt = prompt
	c.input = []rune{}
	c.ch = make(chan rune)

	return c, nil
}

func (c *ctx) tearDown() {
	syscall.Syscall6(syscall.SYS_IOCTL, c.in, uintptr(TCSETS), uintptr(unsafe.Pointer(&c.st)), 0, 0, 0)
	if c.ch != nil {
		close(c.ch)
	}
}

func (c *ctx) redraw(dirty bool) error {
	os.Stdout.WriteString("\x1b[>5h")
	os.Stdout.WriteString("\r")
	if dirty {
		os.Stdout.WriteString("\x1b[2K")
		os.Stdout.WriteString(c.prompt + string(c.input))
		os.Stdout.WriteString("\r")
	}
	x := runewidth.StringWidth(c.prompt) + runewidth.StringWidth(string(c.input[:c.cursor_x]))
	os.Stdout.WriteString(fmt.Sprintf("\x1b[%dC", x))
	os.Stdout.WriteString("\x1b[>5l")
	return nil
}
