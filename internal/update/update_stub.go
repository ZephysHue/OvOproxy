//go:build !windows

package update

import "syscall"

func getDetachAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}
