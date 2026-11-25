//go:build windows

package power

import (
	"fmt"
	"runtime"
	"sync"
	"syscall"
)

const (
	ES_SYSTEM_REQUIRED   = 0x00000001
	ES_DISPLAY_REQUIRED  = 0x00000002
	ES_USER_PRESENT      = 0x00000004
	ES_AWAYMODE_REQUIRED = 0x00000040
	ES_CONTINUOUS        = 0x80000000
)

var (
	kernel32                    = syscall.NewLazyDLL("kernel32.dll")
	procSetThreadExecutionState = kernel32.NewProc("SetThreadExecutionState")
)

type powerCmd struct {
	inhibit         bool
	requireAwayMode bool
	reply           chan error
}

type Manager struct {
	mu          sync.Mutex
	isInhibited bool
	cmds        chan powerCmd
}

func NewManager() *Manager {
	m := &Manager{
		cmds: make(chan powerCmd),
	}
	go m.runLockedThread()
	return m
}

// runLockedThread pins a single OS thread for all SetThreadExecutionState calls.
// Windows tracks thread execution state per-OS-thread, so InhibitSleep and
// AllowSleep must always execute on the exact same OS thread.
func (m *Manager) runLockedThread() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	for cmd := range m.cmds {
		var flags uintptr
		if cmd.inhibit {
			flags = uintptr(ES_CONTINUOUS | ES_SYSTEM_REQUIRED)
			if cmd.requireAwayMode {
				flags |= uintptr(ES_AWAYMODE_REQUIRED)
			}
		} else {
			flags = uintptr(ES_CONTINUOUS)
		}

		ret, _, err := procSetThreadExecutionState.Call(flags)
		if ret == 0 {
			cmd.reply <- fmt.Errorf("SetThreadExecutionState(0x%X) failed: %v", flags, err)
		} else {
			cmd.reply <- nil
		}
	}
}

// InhibitSleep tells Windows to stay awake and prevent system sleep
func (m *Manager) InhibitSleep(requireAwayMode bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	reply := make(chan error, 1)
	m.cmds <- powerCmd{
		inhibit:         true,
		requireAwayMode: requireAwayMode,
		reply:           reply,
	}
	if err := <-reply; err != nil {
		return err
	}

	m.isInhibited = true
	return nil
}

// AllowSleep releases system sleep prevention, allowing Windows to sleep according to power plan
func (m *Manager) AllowSleep() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	reply := make(chan error, 1)
	m.cmds <- powerCmd{
		inhibit: false,
		reply:   reply,
	}
	if err := <-reply; err != nil {
		return err
	}

	m.isInhibited = false
	return nil
}

// IsInhibited returns whether sleep prevention is currently active
func (m *Manager) IsInhibited() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.isInhibited
}
