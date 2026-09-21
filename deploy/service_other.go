//go:build !windows

package main

import (
	"context"
	"fmt"
)

func isWindowsService() bool { return false }

func runWindowsService(run func(ctx context.Context) error) error {
	return fmt.Errorf("windows service mode is only available on Windows; run without -service")
}

func serviceNameOnOS() string { return "" }
