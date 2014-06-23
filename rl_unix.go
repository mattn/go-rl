// +build !windows

package rl

import (
	"errors"
	"syscall"
)

func readLine(prompt string) (string, error) {
	return "", errors.New("Not implemented")
}
