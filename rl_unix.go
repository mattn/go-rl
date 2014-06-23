// +build !windows

package rl

import (
	"fmt"
	"github.com/mattn/go-runewidth"
	"io"
	"os"
	"syscall"
	"unsafe"
)

func readLine(prompt string) (string, error) {
	var old syscall.Termios

	fd := int(os.Stdin.Fd())
	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TCGETS), uintptr(unsafe.Pointer(&old)), 0, 0, 0); err != 0 {
		return "", err
	}

	defer func() {
		syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TCSETS), uintptr(unsafe.Pointer(&old)), 0, 0, 0)
	}()

	cur := old

	cur.Iflag &^= syscall.ISTRIP | syscall.INLCR | syscall.ICRNL | syscall.IGNCR | syscall.IXON | syscall.IXOFF
	cur.Lflag &^= syscall.ECHO | syscall.ICANON | syscall.ISIG
	//cur.Lflag &^= syscall.ECHO
	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TCSETS), uintptr(unsafe.Pointer(&cur)), 0, 0, 0); err != 0 {
		return "", err
	}

	var buf [16]byte
	var input []rune
	cursor_x := 0

loop:
	for {
		os.Stdout.WriteString("\r")
		os.Stdout.WriteString("\x1b[2K")
		os.Stdout.WriteString(prompt + string(input))
		os.Stdout.WriteString("\r")
		x := runewidth.StringWidth(prompt) + runewidth.StringWidth(string(input[:cursor_x]))
		os.Stdout.WriteString(fmt.Sprintf("\x1b[%dC", x))

		n, err := syscall.Read(fd, buf[:])
		if err != nil {
			return "", err
		}
		if n == 0 {
			if len(input) == 0 {
				return "", io.EOF
			}
			break
		}
		if buf[n-1] == '\n' {
			n--
		}
		for _, r := range []rune(string(buf[:n])) {
			switch r {
			case 0:
			case 1: // CTRL-A
				cursor_x = 0
			case 2: // CTRL-B
				if cursor_x > 0 {
					cursor_x--
				}
			case 5: // CTRL-E
				cursor_x = len(input)
			case 6: // CTRL-F
				if cursor_x < len(input) {
					cursor_x++
				}
			case 8: // BS
				if cursor_x > 0 {
					input = append(input[0:cursor_x-1], input[cursor_x:len(input)]...)
					cursor_x--
				}
			case 10: // LF
				break loop
			case 11: // CTRL-K
				input = input[:cursor_x]
			case 13: // CR
				break loop
			case 21: // CTRL-U
				input = input[cursor_x:]
				cursor_x = 0
			case 23: // CTRL-W
				for i := len(input) - 1; i >= 0; i-- {
					if i == 0 || input[i] == ' ' || input[i] == '\t' {
						input = append(input[:i], input[cursor_x:]...)
						cursor_x = i
						break
					}
				}
			default:
				tmp := []rune{}
				tmp = append(tmp, input[0:cursor_x]...)
				tmp = append(tmp, r)
				input = append(tmp, input[cursor_x:len(input)]...)
				cursor_x++
			}
		}
	}
	fmt.Println()

	return string(input), nil
}
