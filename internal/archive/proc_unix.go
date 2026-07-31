//go:build !windows

package archive

import "syscall"

// detachAttrs puts the session runner in its own session so it outlives the
// process that started it.
func detachAttrs() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}

func processAlive(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}
