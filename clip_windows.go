// Windows clipboard, for copying the moves as PGN and pasting an address.
// (c) 2026 Ironite
package main

import (
	"errors"
	"syscall"
	"unsafe"
)

var (
	user32           = syscall.NewLazyDLL("user32.dll")
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	openClipboard    = user32.NewProc("OpenClipboard")
	closeClipboard   = user32.NewProc("CloseClipboard")
	emptyClipboard   = user32.NewProc("EmptyClipboard")
	setClipboardData = user32.NewProc("SetClipboardData")
	getClipboardData = user32.NewProc("GetClipboardData")
	globalAlloc      = kernel32.NewProc("GlobalAlloc")
	globalLock       = kernel32.NewProc("GlobalLock")
	globalUnlock     = kernel32.NewProc("GlobalUnlock")
	globalFree       = kernel32.NewProc("GlobalFree")
)

const cfUnicodeText = 13

func copyText(s string) error {
	u, err := syscall.UTF16FromString(s)
	if err != nil {
		return err
	}
	if r, _, _ := openClipboard.Call(0); r == 0 {
		return errors.New("clipboard busy")
	}
	defer closeClipboard.Call()
	emptyClipboard.Call()
	h, _, _ := globalAlloc.Call(0x0002, uintptr(len(u)*2)) // GMEM_MOVEABLE
	if h == 0 {
		return errors.New("out of memory")
	}
	p, _, _ := globalLock.Call(h)
	if p == 0 {
		globalFree.Call(h)
		return errors.New("lock failed")
	}
	copy(unsafe.Slice((*uint16)(unsafe.Pointer(p)), len(u)), u)
	globalUnlock.Call(h)
	if r, _, _ := setClipboardData.Call(cfUnicodeText, h); r == 0 {
		globalFree.Call(h)
		return errors.New("copy failed")
	}
	return nil
}

func pasteText() (string, error) {
	if r, _, _ := openClipboard.Call(0); r == 0 {
		return "", errors.New("clipboard busy")
	}
	defer closeClipboard.Call()
	h, _, _ := getClipboardData.Call(cfUnicodeText)
	if h == 0 {
		return "", errors.New("no text")
	}
	p, _, _ := globalLock.Call(h)
	if p == 0 {
		return "", errors.New("lock failed")
	}
	defer globalUnlock.Call(h)
	var out []uint16
	for i := uintptr(0); i < 1<<16; i++ {
		c := *(*uint16)(unsafe.Pointer(p + i*2))
		if c == 0 {
			break
		}
		out = append(out, c)
	}
	return syscall.UTF16ToString(out), nil
}

func audioUnlock() {}
