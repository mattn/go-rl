package rl

import (
	"fmt"
	"github.com/mattn/go-runewidth"
	"os"
	"syscall"
	"unicode"
	"unicode/utf16"
	"unsafe"
)

const (
	enableLineInput       = 2
	enableEchoInput       = 4
	enableProcessedInput  = 1
	enableWindowInput     = 8
	enableMouseInput      = 16
	enableInsertMode      = 32
	enableQuickEditMode   = 64
	enableExtendedFlags   = 128
	enableAutoPosition    = 256
	enableProcessedOutput = 1
	enableWrapAtEolOutput = 2

	keyEvent              = 0x1
	mouseEvent            = 0x2
	windowBufferSizeEvent = 0x4
)

var kernel32 = syscall.NewLazyDLL("kernel32.dll")

var (
	procSetStdHandle                = kernel32.NewProc("SetStdHandle")
	procGetStdHandle                = kernel32.NewProc("GetStdHandle")
	procSetConsoleScreenBufferSize  = kernel32.NewProc("SetConsoleScreenBufferSize")
	procCreateConsoleScreenBuffer   = kernel32.NewProc("CreateConsoleScreenBuffer")
	procGetConsoleScreenBufferInfo  = kernel32.NewProc("GetConsoleScreenBufferInfo")
	procWriteConsoleOutputCharacter = kernel32.NewProc("WriteConsoleOutputCharacterW")
	procWriteConsoleOutputAttribute = kernel32.NewProc("WriteConsoleOutputAttribute")
	procSetConsoleCursorInfo        = kernel32.NewProc("SetConsoleCursorInfo")
	procSetConsoleCursorPosition    = kernel32.NewProc("SetConsoleCursorPosition")
	procReadConsoleInput            = kernel32.NewProc("ReadConsoleInputW")
	procGetConsoleMode              = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode              = kernel32.NewProc("SetConsoleMode")
	procFillConsoleOutputCharacter  = kernel32.NewProc("FillConsoleOutputCharacterW")
	procFillConsoleOutputAttribute  = kernel32.NewProc("FillConsoleOutputAttribute")
)

type wchar uint16
type short int16
type dword uint32
type word uint16

type coord struct {
	x short
	y short
}

type smallRect struct {
	left   short
	top    short
	right  short
	bottom short
}
type consoleScreenBufferInfo struct {
	size              coord
	cursorPosition    coord
	attributes        word
	window            smallRect
	maximumWindowSize coord
}

type consoleCursorInfo struct {
	size    dword
	visible int32
}

type inputRecord struct {
	eventType word
	_         [2]byte
	event     [16]byte
}

type keyEventRecord struct {
	keyDown         int32
	repeatCount     word
	virtualKeyCode  word
	virtualScanCode word
	unicodeChar     wchar
	controlKeyState dword
}

type windowBufferSizeRecord struct {
	size coord
}

type mouseEventRecord struct {
	mousePos        coord
	buttonState     dword
	controlKeyState dword
	eventFlags      dword
}

func isTty() bool {
	var st uint32
	r1, _, err := procGetConsoleMode.Call(uintptr(os.Stdin.Fd()), uintptr(unsafe.Pointer(&st)))
	return r1 != 0 && err != nil
}

func getStdHandle(stdhandle int32) uintptr {
	r1, _, _ := procGetStdHandle.Call(uintptr(stdhandle))
	return r1
}

func setStdHandle(stdhandle int32, handle uintptr) error {
	r1, _, err := procSetStdHandle.Call(uintptr(stdhandle), handle)
	if r1 == 0 {
		return err
	}
	return nil
}

func writeConsole(fd uintptr, rs []rune) error {
	wchars := utf16.Encode(rs)
	var w uint32
	return syscall.WriteConsole(syscall.Handle(fd), &wchars[0], uint32(len(wchars)), &w, nil)
}

func readConsoleInput(fd uintptr, record *inputRecord) (err error) {
	var w uint32
	r1, _, err := procReadConsoleInput.Call(fd, uintptr(unsafe.Pointer(record)), 1, uintptr(unsafe.Pointer(&w)))
	if r1 == 0 {
		return err
	}
	return nil
}

func readLine(prompt string) (string, error) {
	var in, out uintptr

	if isTty() {
		in = getStdHandle(syscall.STD_INPUT_HANDLE)
		out = getStdHandle(syscall.STD_OUTPUT_HANDLE)
	} else {
		conin, err := os.Open("CONIN$")
		if err != nil {
			return "", err
		}
		in = conin.Fd()

		conout, err := os.Open("CONOUT$")
		if err != nil {
			return "", err
		}
		out = conout.Fd()
	}

	var st uint32
	r1, _, err := procGetConsoleMode.Call(in, uintptr(unsafe.Pointer(&st)))
	if r1 == 0 {
		return "", err
	}
	old := st

	st &^= (enableEchoInput | enableLineInput)
	r1, _, err = procSetConsoleMode.Call(in, uintptr(st))
	if r1 == 0 {
		return "", err
	}

	defer func() {
		procSetConsoleMode.Call(in, uintptr(old))
	}()

	var input []rune
	var csbi consoleScreenBufferInfo

	r1, _, err = procGetConsoleScreenBufferInfo.Call(out, uintptr(unsafe.Pointer(&csbi)))
	if r1 == 0 {
		return "", err
	}
	cursor := csbi.cursorPosition
	cursor_x := 0

	rc := make(chan rune)
	go func() {
		for {
			var ir inputRecord
			err := readConsoleInput(in, &ir)
			if err != nil {
				break
			}

			switch ir.eventType {
			case keyEvent:
				kr := (*keyEventRecord)(unsafe.Pointer(&ir.event))
				if kr.keyDown != 0 {
					rc <- rune(kr.unicodeChar)
				}
			case windowBufferSizeEvent:
				sr := *(*windowBufferSizeRecord)(unsafe.Pointer(&ir.event))
				println(&sr)
			case mouseEvent:
				mr := *(*mouseEventRecord)(unsafe.Pointer(&ir.event))
				println(&mr)
			}
		}
	}()

loop:
	for {
		var w uint32
		r1, _, err = procGetConsoleScreenBufferInfo.Call(out, uintptr(unsafe.Pointer(&csbi)))
		if r1 == 0 {
			return "", err
		}
		cursor = csbi.cursorPosition
		cursor.x = 0
		r1, _, err = procFillConsoleOutputCharacter.Call(out, uintptr(' '), uintptr(csbi.size.x), *(*uintptr)(unsafe.Pointer(&cursor)), uintptr(unsafe.Pointer(&w)))
		if r1 == 0 {
			return "", err
		}
		r1, _, err = procSetConsoleCursorPosition.Call(out, uintptr(*(*int32)(unsafe.Pointer(&cursor))))
		if r1 == 0 {
			return "", err
		}
		writeConsole(out, []rune(prompt+string(input)))
		cursor.x = short(runewidth.StringWidth(prompt)) + short(runewidth.StringWidth(string(input[:cursor_x])))
		r1, _, err = procSetConsoleCursorPosition.Call(out, uintptr(*(*int32)(unsafe.Pointer(&cursor))))
		if r1 == 0 {
			return "", err
		}

		select {
		case r := <-rc:
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
					if i == 0 || unicode.IsPrint(input[i]) {
						input = append(input[:i], input[cursor_x:]...)
						cursor_x = i
						break
					}
				}
			default:
				if unicode.IsPrint(r) {
					tmp := []rune{}
					tmp = append(tmp, input[0:cursor_x]...)
					tmp = append(tmp, r)
					input = append(tmp, input[cursor_x:len(input)]...)
					cursor_x++
				}
			}
		}
	}
	fmt.Println()

	return string(input), nil
}
