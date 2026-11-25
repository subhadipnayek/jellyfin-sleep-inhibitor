//go:build windows

package service

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// Install registers the application as an auto-starting Windows Service
func Install() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to determine executable path: %w", err)
	}
	exePath, err = filepath.Abs(exePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute executable path: %w", err)
	}

	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to Windows Service Manager: %w (make sure to run as Administrator)", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(ServiceName)
	if err == nil {
		s.Close()
		return fmt.Errorf("service %s is already installed", ServiceName)
	}

	serviceConfig := mgr.Config{
		DisplayName:      ServiceDisplayName,
		Description:      ServiceDescription,
		StartType:        mgr.StartAutomatic,
		ServiceStartName: "", // LocalSystem
	}

	s, err = m.CreateService(ServiceName, exePath, serviceConfig)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}
	defer s.Close()

	fmt.Printf("[OK] Service '%s' successfully installed with Automatic startup.\n", ServiceName)
	fmt.Printf("Binary path: %s\n", exePath)
	return nil
}

// Uninstall stops and removes the Windows Service
func Uninstall() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to Service Manager: %w (make sure to run as Administrator)", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(ServiceName)
	if err != nil {
		return fmt.Errorf("service %s is not installed", ServiceName)
	}
	defer s.Close()

	// Try stopping it first
	status, err := s.Query()
	if err == nil && status.State != svc.Stopped {
		fmt.Println("Stopping running service...")
		_, _ = s.Control(svc.Stop)
		time.Sleep(1 * time.Second)
	}

	if err := s.Delete(); err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	fmt.Printf("[OK] Service '%s' uninstalled successfully.\n", ServiceName)
	return nil
}

// Start launches the installed Windows Service
func Start() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to Service Manager: %w (make sure to run as Administrator)", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(ServiceName)
	if err != nil {
		return fmt.Errorf("service %s is not installed: %w", ServiceName, err)
	}
	defer s.Close()

	if err := s.Start(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	fmt.Printf("[OK] Service '%s' started successfully.\n", ServiceName)
	return nil
}

// Stop halts the running Windows Service
func Stop() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to Service Manager: %w (make sure to run as Administrator)", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(ServiceName)
	if err != nil {
		return fmt.Errorf("service %s is not installed: %w", ServiceName, err)
	}
	defer s.Close()

	status, err := s.Control(svc.Stop)
	if err != nil {
		return fmt.Errorf("failed to send stop command: %w", err)
	}

	fmt.Printf("[OK] Service '%s' stop signal sent (current state: %d).\n", ServiceName, status.State)
	return nil
}

// Status queries the current service state
func Status() (string, error) {
	// Connect to SCM with read-only access so non-admin users can check status
	h, err := windows.OpenSCManager(nil, nil, windows.GENERIC_READ)
	if err != nil {
		return "", fmt.Errorf("failed to open Service Manager: %w (try running as Administrator)", err)
	}
	m := &mgr.Mgr{Handle: h}
	defer m.Disconnect()

	namePtr, err := windows.UTF16PtrFromString(ServiceName)
	if err != nil {
		return "", err
	}
	sh, err := windows.OpenService(m.Handle, namePtr, windows.SERVICE_QUERY_STATUS)
	if err != nil {
		if err == windows.ERROR_SERVICE_DOES_NOT_EXIST {
			return "Not Installed", nil
		}
		return "", fmt.Errorf("failed to open service: %w", err)
	}
	s := &mgr.Service{Name: ServiceName, Handle: sh}
	defer s.Close()

	status, err := s.Query()
	if err != nil {
		return "", fmt.Errorf("failed to query service status: %w", err)
	}

	switch status.State {
	case svc.Stopped:
		return "Stopped", nil
	case svc.StartPending:
		return "Start Pending", nil
	case svc.StopPending:
		return "Stop Pending", nil
	case svc.Running:
		return "Running", nil
	case svc.ContinuePending:
		return "Continue Pending", nil
	case svc.PausePending:
		return "Pause Pending", nil
	case svc.Paused:
		return "Paused", nil
	default:
		return fmt.Sprintf("Unknown state (%d)", status.State), nil
	}
}
