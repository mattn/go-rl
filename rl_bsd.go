//go:build darwin || dragonfly || freebsd || netbsd || openbsd
// +build darwin dragonfly freebsd netbsd openbsd

package rl

import "golang.org/x/sys/unix"

const TCGETS = unix.TIOCGETA
const TCSETS = unix.TIOCSETA
