//go:build windows

package singleinstance

import (
	"syscall"
	"unsafe"
)

var (
	kernel32     = syscall.NewLazyDLL("kernel32.dll")
	createMutexW = kernel32.NewProc("CreateMutexW")
	releaseMutex = kernel32.NewProc("ReleaseMutex")
	closeHandle  = kernel32.NewProc("CloseHandle")
)

const errorAlreadyExists syscall.Errno = 183

var mutexHandle uintptr

func Acquire(name string) (bool, error) {
	namePtr, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return false, err
	}

	handle, _, callErr := createMutexW.Call(
		0,
		1,
		uintptr(unsafe.Pointer(namePtr)),
	)
	if handle == 0 {
		return false, callErr
	}

	if callErr == errorAlreadyExists {
		closeHandle.Call(handle)
		return false, nil
	}

	mutexHandle = handle
	return true, nil
}

func Release() {
	if mutexHandle != 0 {
		releaseMutex.Call(mutexHandle)
		closeHandle.Call(mutexHandle)
		mutexHandle = 0
	}
}
