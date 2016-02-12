package ns

import (
	"fmt"

	"golang.org/x/sys/unix"
)

type ns struct{}

type handle struct {
	fd     uintptr
	closed bool
}

var LinuxNamespacer = &ns{}

func (*ns) GetFromPath(path string) (Handle, error) {
	fd, err := unix.Open(path, unix.O_RDONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("open failed: %s", err)
	}
	return &handle{
		fd: uintptr(fd),
	}, nil
}

func (*ns) Set(handle Handle) error {
	_, _, errno := unix.Syscall(unix.SYS_SETNS, handle.Fd(), uintptr(unix.CLONE_NEWNET), 0)
	if errno != 0 {
		return fmt.Errorf("failed to set namespace: %s", errno)
	}

	return nil
}

func (h *handle) Fd() uintptr {
	return h.fd
}

func (h *handle) Close() error {
	if err := unix.Close(int(h.fd)); err != nil {
		return fmt.Errorf("close failed: %s", err)
	}

	h.closed = true
	return nil
}

func (h *handle) IsOpen() bool {
	return !h.closed
}
