//go:build windows

package power

import (
	"fmt"
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
	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	procSetThreadExecutionState  = kernel32.NewProc("SetThreadExecutionState")
)

type Manager struct {
	mu          sync.Mutex
	isInhibited bool
}

func NewManager() *Manager {
	return &Manager{}
}

// InhibitSleep tells Windows to stay awake and prevent system sleep
func (m *Manager) InhibitSleep(requireAwayMode bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	flags := uintptr(ES_CONTINUOUS | ES_SYSTEM_REQUIRED)
	if requireAwayMode {
		flags |= uintptr(ES_AWAYMODE_REQUIRED)
	}

	ret, _, err := procSetThreadExecutionState.Call(flags)
	if ret == 0 {
		return fmt.Errorf("SetThreadExecutionState failed: %v", err)
	}

	m.isInhibited = true
	return nil
}

// AllowSleep releases system sleep prevention, allowing Windows to sleep according to power plan
func (m *Manager) AllowSleep() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Calling SetThreadExecutionState with only ES_CONTINUOUS clears other flags
	ret, _, err := procSetThreadExecutionState.Call(uintptr(ES_CONTINUOUS))
	if ret == 0 {
		return fmt.Errorf("SetThreadExecutionState(ES_CONTINUOUS) failed: %v", err)
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
