package rl

import (
	"syscall"
)

const TCGETS = syscall.TIOCGETA
const TCSETS = syscall.TIOCSETA
