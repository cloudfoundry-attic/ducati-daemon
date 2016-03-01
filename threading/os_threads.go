package threading

import "runtime"

//go:generate counterfeiter -o ../fakes/os_thread_locker.go --fake-name OSThreadLocker . OSThreadLocker
type OSThreadLocker interface {
	LockOSThread()
	UnlockOSThread()
}

type OSLocker struct{}

func (l *OSLocker) LockOSThread() {
	runtime.LockOSThread()
}

func (l *OSLocker) UnlockOSThread() {
	runtime.UnlockOSThread()
}
