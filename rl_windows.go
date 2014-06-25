package rl

import (
	"github.com/mattn/go-runewidth"
	"os"
	"syscall"
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
	procGetConsoleCursorInfo        = kernel32.NewProc("GetConsoleCursorInfo")
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
	if len(rs) == 0 {
		return nil
	}
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

type ctx struct {
	in       uintptr
	out      uintptr
	st       uint32
	input    []rune
	prompt   string
	cursor_x int
	old_row     int
	old_crow     int
	size     int
	old_size  int
}

func (c *ctx) readRunes() ([]rune, error) {
	var ir inputRecord
	err := readConsoleInput(c.in, &ir)
	if err != nil {
		return nil, err
	}

	switch ir.eventType {
	case keyEvent:
		kr := (*keyEventRecord)(unsafe.Pointer(&ir.event))
		if kr.keyDown != 0 {
			return []rune{rune(kr.unicodeChar)}, nil
		}
	case windowBufferSizeEvent:
		sr := *(*windowBufferSizeRecord)(unsafe.Pointer(&ir.event))
		println(&sr)
	case mouseEvent:
		mr := *(*mouseEventRecord)(unsafe.Pointer(&ir.event))
		println(&mr)
	}

	return nil, nil
}

func NewRl(prompt string) (*ctx, error) {
	c := new(ctx)
	if isTty() {
		c.in = getStdHandle(syscall.STD_INPUT_HANDLE)
		c.out = getStdHandle(syscall.STD_OUTPUT_HANDLE)
	} else {
		conin, err := os.Open("CONIN$")
		if err != nil {
			return nil, err
		}
		c.in = conin.Fd()

		conout, err := os.Open("CONOUT$")
		if err != nil {
			return nil, err
		}
		c.out = conout.Fd()
	}

	var st uint32
	r1, _, err := procGetConsoleMode.Call(c.in, uintptr(unsafe.Pointer(&st)))
	if r1 == 0 {
		return nil, err
	}
	c.st = st

	st &^= (enableEchoInput | enableLineInput)
	r1, _, err = procSetConsoleMode.Call(c.in, uintptr(st))
	if r1 == 0 {
		return nil, err
	}

	var csbi consoleScreenBufferInfo

	r1, _, err = procGetConsoleScreenBufferInfo.Call(c.out, uintptr(unsafe.Pointer(&csbi)))
	if r1 == 0 {
		return nil, err
	}

	c.prompt = prompt
	c.input = []rune{}
	c.size = int(csbi.size.x) - 1
	c.old_size = c.size

	return c, nil
}

func (c *ctx) tearDown() {
	procSetConsoleMode.Call(c.in, uintptr(c.st))
}

func (c *ctx) redraw(dirty bool) error {
	var csbi consoleScreenBufferInfo

	var ci consoleCursorInfo
	r1, _, err := procGetConsoleCursorInfo.Call(c.out, uintptr(unsafe.Pointer(&ci)))
	if r1 == 0 {
		return err
	}
	ci.visible = 0
	r1, _, err = procSetConsoleCursorInfo.Call(c.out, uintptr(unsafe.Pointer(&ci)))
	if r1 == 0 {
		return err
	}
	defer func() {
		ci.visible = 1
		procSetConsoleCursorInfo.Call(c.out, uintptr(unsafe.Pointer(&ci)))
	}()

	r1, _, err = procGetConsoleScreenBufferInfo.Call(c.out, uintptr(unsafe.Pointer(&csbi)))
	if r1 == 0 {
		return err
	}

	var oldpos, cursor coord
	oldpos.x = 0
	oldpos.y = csbi.cursorPosition.y - short(c.old_crow)
	cursor.x = 0
	cursor.y = oldpos.y
	r1, _, err = procSetConsoleCursorPosition.Call(c.out, uintptr(*(*int32)(unsafe.Pointer(&cursor))))
	if r1 == 0 {
		return err
	}
	if dirty {
		var w uint32
		r1, _, err = procFillConsoleOutputCharacter.Call(c.out, uintptr(' '), uintptr(csbi.size.x), *(*uintptr)(unsafe.Pointer(&cursor)), uintptr(unsafe.Pointer(&w)))
		if r1 == 0 {
			return err
		}
	}
	cursor.y -= short(c.old_row - c.old_crow)
	if dirty {
		for i := 0; i <  c.old_row; i++ {
			var w uint32
			r1, _, err = procFillConsoleOutputCharacter.Call(c.out, uintptr(' '), uintptr(csbi.size.x), *(*uintptr)(unsafe.Pointer(&cursor)), uintptr(unsafe.Pointer(&w)))
			if r1 == 0 {
				return err
			}
			cursor.y++
		}
	}

	var ccol, crow, col, row int
	ccol = -1
	plen := len([]rune(c.prompt))
	for i, r := range []rune(c.prompt + string(c.input)) {
		if i == plen + c.cursor_x {
			ccol = col
			crow = row
		}
		rw := runewidth.RuneWidth(r)
		if col + rw >= c.size {
			col = 0
			row++
		}
		if dirty {
			cursor.x = oldpos.x + short(col)
			cursor.y = oldpos.y + short(row)
			var w uint32
			r1, _, err = procFillConsoleOutputCharacter.Call(c.out, uintptr(r), uintptr(rw), *(*uintptr)(unsafe.Pointer(&cursor)), uintptr(unsafe.Pointer(&w)))
			if r1 == 0 {
				return err
			}
		}
		col += rw
	}

	if ccol == -1 {
		ccol = col
		crow = row
	}
	cursor.x = oldpos.x + short(ccol)
	cursor.y = oldpos.y + short(crow)
	r1, _, err = procSetConsoleCursorPosition.Call(c.out, uintptr(*(*int32)(unsafe.Pointer(&cursor))))
	if r1 == 0 {
		return err
	}

	c.old_row = row
	c.old_crow = crow

	return nil
}
