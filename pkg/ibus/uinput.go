package ibus

import (
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// Linux input/uinput constants (linux/input.h, linux/uinput.h).
const (
	evSyn uint16 = 0x00
	evKey uint16 = 0x01

	keyBackspace uint16 = 14
	keyReturn    uint16 = 28

	uiDevCreate  uintptr = 0x5501
	uiDevDestroy uintptr = 0x5502
	uiSetEvBit   uintptr = 0x40045564
	uiSetKeyBit  uintptr = 0x40045565
	uiDevSetup   uintptr = 0x405c5503

	busUSB uint16 = 0x03
)

type uinputID struct {
	Bustype uint16
	Vendor  uint16
	Product uint16
	Version uint16
}

type uinputSetup struct {
	ID   uinputID
	Name [80]byte
	FF   uint32
}

// inputEvent mirrors struct input_event. Using syscall.Timeval keeps the
// layout correct on both 64-bit (16-byte timeval) and 32-bit (8-byte).
type inputEvent struct {
	Time  syscall.Timeval
	Type  uint16
	Code  uint16
	Value int32
}

// keyTapper injects real kernel key events. Unlike IBus ForwardKeyEvent —
// which Chromium/Docs/terminals silently ignore — these are indistinguishable
// from a physical keyboard, so every client honors them.
type keyTapper interface {
	Tap(key uint16) error
	Close() error
}

type virtualKeyboard struct {
	fd int
	mu sync.Mutex
}

func ioctl(fd int, req uintptr, arg uintptr) error {
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), req, arg); errno != 0 {
		return errno
	}
	return nil
}

func openVirtualKeyboard(path string) (*virtualKeyboard, error) {
	fd, err := syscall.Open(path, syscall.O_WRONLY|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	k := &virtualKeyboard{fd: fd}
	fail := func(what string, err error) (*virtualKeyboard, error) {
		_ = k.Close()
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	if err := ioctl(fd, uiSetEvBit, uintptr(evKey)); err != nil {
		return fail("UI_SET_EVBIT", err)
	}
	for _, key := range []uint16{keyReturn, keyBackspace} {
		if err := ioctl(fd, uiSetKeyBit, uintptr(key)); err != nil {
			return fail("UI_SET_KEYBIT", err)
		}
	}
	setup := uinputSetup{ID: uinputID{Bustype: busUSB, Vendor: 0x1d6b, Product: 0x1001, Version: 1}}
	copy(setup.Name[:], "NepKey Virtual Keyboard")
	// Kernels before 4.5 lack UI_DEV_SETUP; the legacy path still creates
	// a usable device, so a failure here is not fatal.
	_ = ioctl(fd, uiDevSetup, uintptr(unsafe.Pointer(&setup)))
	if err := ioctl(fd, uiDevCreate, 0); err != nil {
		return fail("UI_DEV_CREATE", err)
	}
	return k, nil
}

func (k *virtualKeyboard) write(typ, code uint16, value int32) error {
	ev := inputEvent{Type: typ, Code: code, Value: value}
	buf := unsafe.Slice((*byte)(unsafe.Pointer(&ev)), int(unsafe.Sizeof(ev)))
	_, err := syscall.Write(k.fd, buf)
	return err
}

func (k *virtualKeyboard) Tap(key uint16) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.fd < 0 {
		return os.ErrClosed
	}
	if err := k.write(evKey, key, 1); err != nil {
		return err
	}
	if err := k.write(evSyn, 0, 0); err != nil {
		return err
	}
	// A short hold makes the press read as a real keystroke to compositors
	// that debounce or coalesce events.
	time.Sleep(8 * time.Millisecond)
	if err := k.write(evKey, key, 0); err != nil {
		return err
	}
	return k.write(evSyn, 0, 0)
}

func (k *virtualKeyboard) Close() error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.fd < 0 {
		return nil
	}
	_ = ioctl(k.fd, uiDevDestroy, 0)
	err := syscall.Close(k.fd)
	k.fd = -1
	return err
}

// The virtual keyboard is process-wide: the IBus factory creates one engine
// per input context, and each must reuse the same device instead of piling
// up /dev/uinput nodes.
var (
	vkOnce sync.Once
	vkDev  *virtualKeyboard
	vkErr  error
)

// sharedVirtualKeyboard opens the device once per process. A nil tapper with
// a nil error means "not available" — callers then keep the legacy behavior.
func sharedVirtualKeyboard(path string) keyTapper {
	vkOnce.Do(func() {
		vkDev, vkErr = openVirtualKeyboard(path)
	})
	if vkErr != nil || vkDev == nil {
		return nil
	}
	return vkDev
}
