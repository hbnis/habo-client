//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

const (
	legacyStartupTaskName = "Habo Client"
	legacyStartupMarker   = "legacy-autostart-disabled-v1"
)

// Older Habo Client installers created a Task Scheduler entry that launched
// the client at Windows logon. New releases no longer do that. This migration
// silently removes the old task the first time the updated client runs.
//
// If the current launch is not elevated and Windows refuses the deletion, no
// marker is written. The old task itself launches the client with highest
// privileges, so the next logon gives the migration another chance and it can
// remove the task then without asking the user to reinstall anything.
func init() {
	cleanupLegacyStartupTask()
}

func cleanupLegacyStartupTask() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return
	}

	migrationDir := filepath.Join(configDir, "albiondata-client")
	markerPath := filepath.Join(migrationDir, legacyStartupMarker)
	if _, err := os.Stat(markerPath); err == nil {
		return
	}

	systemRoot := os.Getenv("SystemRoot")
	if systemRoot == "" {
		return
	}

	// Task Scheduler stores root-level tasks here. If it is already gone,
	// mark the migration complete so new installs do not run schtasks on every
	// launch.
	taskPath := filepath.Join(systemRoot, "System32", "Tasks", legacyStartupTaskName)
	if _, err := os.Stat(taskPath); os.IsNotExist(err) {
		markLegacyStartupMigrationDone(migrationDir, markerPath)
		return
	}

	cmd := exec.Command(
		filepath.Join(systemRoot, "System32", "schtasks.exe"),
		"/Delete",
		"/TN", legacyStartupTaskName,
		"/F",
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Run(); err != nil {
		// Do not mark this as complete. A later elevated launch can retry.
		return
	}

	markLegacyStartupMigrationDone(migrationDir, markerPath)
}

func markLegacyStartupMigrationDone(dir, markerPath string) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return
	}
	_ = os.WriteFile(markerPath, []byte("done\n"), 0o600)
}
