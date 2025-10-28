//go:build windows
// +build windows

package clean

import (
	"dominicbreuker/goncat/pkg/log"
	"os/exec"
	"strconv"
	"syscall"
	"unsafe"
)

// deleteFile removes the file at the specified path using Windows-specific deletion methods.
// It attempts two approaches:
//  1. Delayed deletion using cmd.exe timeout command
//  2. Deletion on system reboot using MoveFileEx with MOVEFILE_DELAY_UNTIL_REBOOT flag
//
// Errors are logged but not returned.
func deleteFile(path string, logger *log.Logger) {
	deleteFileAfterExit(path, logger)
	deleteFileOnReboot(path, logger)
}

// deleteFileAfterExit schedules file deletion using cmd.exe timeout and del commands.
// It waits 5 seconds before attempting deletion to ensure the process has exited.
func deleteFileAfterExit(path string, logger *log.Logger) {
	// cmd.exe /C timeout /T 5 /NOBREAK > NUL & del <path-to-file>
	cmd := exec.Command("cmd.exe", "/C", "timeout", "/T", strconv.Itoa(5), "/NOBREAK", ">", "NUL", "&", "del", path)

	if err := cmd.Start(); err != nil {
		logger.ErrorMsg("launching cleanup process for %s: %s", path, err)
	}
}

// deleteFileOnReboot marks the file for deletion on next system reboot.
// It uses the Windows MoveFileEx API with the MOVEFILE_DELAY_UNTIL_REBOOT flag.
func deleteFileOnReboot(path string, logger *log.Logger) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	moveFileEx := kernel32.NewProc("MoveFileExW")

	if _, _, err := moveFileEx.Call(
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(path))),
		0,
		0x4, //MOVEFILE_DELAY_UNTIL_REBOOT
	); err != nil {
		logger.ErrorMsg("marking executable %s for deletion: %s\n", path, err)
	}
}
