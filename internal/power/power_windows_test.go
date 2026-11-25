//go:build windows

package power

import (
	"sync"
	"testing"
)

func TestInhibitAndAllowSleepAcrossGoroutines(t *testing.T) {
	m := NewManager()

	// Call InhibitSleep in one goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := m.InhibitSleep(true); err != nil {
			t.Errorf("InhibitSleep failed: %v", err)
		}
	}()
	wg.Wait()

	if !m.IsInhibited() {
		t.Fatalf("expected IsInhibited() to be true")
	}

	// Call AllowSleep from a different goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := m.AllowSleep(); err != nil {
			t.Errorf("AllowSleep failed: %v", err)
		}
	}()
	wg.Wait()

	if m.IsInhibited() {
		t.Fatalf("expected IsInhibited() to be false")
	}
}
