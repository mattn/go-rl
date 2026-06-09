//go:build linux
// +build linux

package rl

import "golang.org/x/sys/unix"

const TCGETS = unix.TCGETS
const TCSETS = unix.TCSETS
