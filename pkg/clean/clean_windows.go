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

func deleteFile(path string) {
	deleteFileAfterExit(path)
	deleteFileOnReboot(path)
}

func deleteFileAfterExit(path string) {
	// cmd.exe /C timeout /T 5 /NOBREAK > NUL & del <path-to-file>
	cmd := exec.Command("cmd.exe", "/C", "timeout", "/T", strconv.Itoa(5), "/NOBREAK", ">", "NUL", "&", "del", path)

	if err := cmd.Start(); err != nil {
		log.ErrorMsg("launching cleanup process for %s: %s", path, err)
	}
}

func deleteFileOnReboot(path string) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	moveFileEx := kernel32.NewProc("MoveFileExW")

	if _, _, err := moveFileEx.Call(
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(path))),
		0,
		0x4, //MOVEFILE_DELAY_UNTIL_REBOOT
	); err != nil {
		log.ErrorMsg("marking executable %s for deletion: %s\n", path, err)
	}
}
