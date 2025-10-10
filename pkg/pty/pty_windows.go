//go:build windows
// +build windows

package pty

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32                          = windows.NewLazySystemDLL("kernel32.dll")
	createPseudoConsole               = kernel32.NewProc("CreatePseudoConsole")
	resizePseudoConsole               = kernel32.NewProc("ResizePseudoConsole")
	closePseudoConsole                = kernel32.NewProc("ClosePseudoConsole")
	initializeProcThreadAttributeList = kernel32.NewProc("InitializeProcThreadAttributeList")
	updateProcThreadAttribute         = kernel32.NewProc("UpdateProcThreadAttribute")
)

// compare github.com/UserExistsError/conpty
// compare https://devblogs.microsoft.com/commandline/windows-command-line-introducing-the-windows-pseudo-console-conpty/

// ConPTY represents a Windows Console Pseudo-Terminal (ConPTY) which provides
// PTY-like functionality on Windows systems. It manages the console handles,
// pipes for I/O, and the associated process information.
type ConPTY struct {
	hPC windows.Handle

	ptyIn  windows.Handle
	ptyOut windows.Handle
	cmdIn  windows.Handle
	cmdOut windows.Handle

	siEx startupInfoEx
	pi   *windows.ProcessInformation
}

type startupInfoEx struct {
	startupInfo   windows.StartupInfo
	attributeList []byte
}

func (cpty *ConPTY) Read(data []byte) (int, error) {
	var n uint32 = 0
	err := windows.ReadFile(cpty.cmdOut, data, &n, nil)
	return int(n), err
}

func (cpty *ConPTY) Write(data []byte) (int, error) {
	var n uint32 = 0
	err := windows.WriteFile(cpty.cmdIn, data, &n, nil)
	return int(n), err
}

// Close closes all handles associated with the ConPTY.
// Note: It does not call ClosePseudoConsole as that terminates the goncat process.
func (cpty *ConPTY) Close() error {
	if err := cpty.closeHandles(); err != nil {
		return err
	}

	// MS says we should do this but it terminates goncat so for now we skip it
	//if cpty.hPC != windows.InvalidHandle {
	//	closePseudoConsole.Call(uintptr(cpty.hPC)) // no errors?!?
	//}

	return nil
}

func (cpty *ConPTY) closeHandles() error {
	var errs []error

	for _, h := range []windows.Handle{
		cpty.cmdIn,
		cpty.cmdOut,
		cpty.ptyIn,
		cpty.ptyOut,
	} {
		if err := closeHandle(h); err != nil {
			errs = append(errs, err)
		}
	}

	if cpty.pi != nil {
		if err := closeHandle(cpty.pi.Process); err != nil {
			errs = append(errs, fmt.Errorf("closing handle to process: %s", err))
		}

		if err := closeHandle(cpty.pi.Thread); err != nil {
			errs = append(errs, fmt.Errorf("closing handle to thread: %s", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("closing handles: %d errors (last = %s)", len(errs), errs[len(errs)-1])
	}

	return nil
}

func closeHandle(h windows.Handle) error {
	if h != windows.InvalidHandle {
		if err := windows.CloseHandle(h); err != nil {
			return err
		}
	}
	return nil
}

// #####################################################
// ############ Setup of ConPTY ########################
// #####################################################

// Create creates a new ConPTY instance on Windows systems.
// It sets up the pseudo-console with pipes for I/O and initializes
// the process thread attribute list for process creation.
func Create() (*ConPTY, error) {
	cpty := &ConPTY{}

	if err := checkConPTYSupport(); err != nil {
		return cpty.err(fmt.Errorf("checkConPTYSupport(): %s", err))
	}

	if err := windows.CreatePipe(&cpty.ptyIn, &cpty.cmdIn, nil, 0); err != nil {
		return cpty.err(fmt.Errorf("CreatePipe(ptyIn, cmdIn, nil, 0): %s", err))
	}
	if err := windows.CreatePipe(&cpty.cmdOut, &cpty.ptyOut, nil, 0); err != nil {
		return cpty.err(fmt.Errorf("CreatePipe(ptyOut, cmdOut, nil, 0): %s", err))
	}

	if err := cpty.createPseudoConsole(); err != nil {
		return cpty.err(fmt.Errorf("CreatePseudoConsole(): %s", err))
	}

	if err := cpty.initializeProcThreadAttributeList(); err != nil {
		return cpty.err(fmt.Errorf("InitializeProcThreadAttributeList(): %s", err))
	}

	if err := cpty.updateProcThreadAttribute(); err != nil {
		return cpty.err(fmt.Errorf("UpdateProcThreadAttribute(): %s", err))
	}

	return cpty, nil
}

func (cpty *ConPTY) err(err error) (*ConPTY, error) {
	cpty.closeHandles()
	return nil, err
}

func checkConPTYSupport() error {
	for _, proc := range []*windows.LazyProc{
		createPseudoConsole,
		resizePseudoConsole,
		closePseudoConsole,
		initializeProcThreadAttributeList,
		updateProcThreadAttribute,
	} {
		if proc.Find() != nil {
			return fmt.Errorf("procedure %s missing", proc.Name)
		}
	}

	return nil
}

func (cpty *ConPTY) createPseudoConsole() error {
	size := TerminalSize{
		Rows: 40,
		Cols: 80,
	}

	ret, _, err := createPseudoConsole.Call(
		size.serialize(),
		uintptr(cpty.ptyIn),
		uintptr(cpty.ptyOut),
		0,
		uintptr(unsafe.Pointer(&cpty.hPC)))
	if ret != uintptr(0) {
		return fmt.Errorf("CreatePseudoConsole() failed with status 0x%x err=%s", ret, err)
	}
	return nil
}

func (cpty *ConPTY) initializeProcThreadAttributeList() error {
	var size uintptr
	ret, _, err := initializeProcThreadAttributeList.Call(0, 1, 0, uintptr(unsafe.Pointer(&size)))
	if ret != 0 {
		return fmt.Errorf("1st call to initializeProcThreadAttributeList() failed: ret = 0x%x err=%s", ret, err)
	}

	cpty.siEx.startupInfo.Cb = uint32(unsafe.Sizeof(windows.StartupInfo{}) + unsafe.Sizeof(&cpty.siEx.attributeList[0]))
	cpty.siEx.startupInfo.Flags |= windows.STARTF_USESTDHANDLES
	cpty.siEx.attributeList = make([]byte, size, size)
	ret, _, err = initializeProcThreadAttributeList.Call(
		uintptr(unsafe.Pointer(&cpty.siEx.attributeList[0])),
		1,
		0,
		uintptr(unsafe.Pointer(&size)))
	if ret != 1 {
		return fmt.Errorf("2nd call to initializeProcThreadAttributeList() failed: ret = 0x%x err=%s", ret, err)
	}

	return nil
}

func (cpty *ConPTY) updateProcThreadAttribute() error {
	ret, _, err := updateProcThreadAttribute.Call(
		uintptr(unsafe.Pointer(&cpty.siEx.attributeList[0])),
		0,
		uintptr(0x20016), //_PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE,
		uintptr(cpty.hPC),
		unsafe.Sizeof(cpty.hPC),
		0,
		0)
	if ret != 1 {
		return fmt.Errorf("UpdateProcThreadAttribute() failed: ret = 0x%x err=%s", ret, err)
	}

	return nil
}

// #####################################################
// ############ Execute program ########################
// #####################################################

// Execute starts a program in the ConPTY environment.
// The program parameter should be a command line string to execute.
func (cpty *ConPTY) Execute(program string) error {
	cmd, err := windows.UTF16PtrFromString(program)
	if err != nil {
		return fmt.Errorf("UTF16PtrFromString(%s): %s", program, err)
	}

	var pi windows.ProcessInformation
	err = windows.CreateProcess(
		nil,
		cmd,
		nil,
		nil,
		false,
		windows.EXTENDED_STARTUPINFO_PRESENT,
		nil,
		nil,
		&cpty.siEx.startupInfo,
		&pi)
	if err != nil {
		return fmt.Errorf("CreateProcess(%s): %s", program, err)
	}

	cpty.pi = &pi

	return nil
}

// Wait blocks until the process running in the ConPTY exits.
// It returns an error containing the exit code when the process terminates.
func (cpty *ConPTY) Wait() error {
	var exitCode uint32

	for {
		ret, _ := windows.WaitForSingleObject(cpty.pi.Process, 1000)
		if ret != uint32(windows.WAIT_TIMEOUT) {
			err := windows.GetExitCodeProcess(cpty.pi.Process, &exitCode)
			if err != nil {
				return fmt.Errorf("getting exit code of process: %s", err)
			}

			return fmt.Errorf("exit code %d", exitCode)
		}
	}
}

// KillProcess terminates the process running in the ConPTY.
// It does nothing if no process is running.
func (cpty *ConPTY) KillProcess() {
	if cpty.pi != nil {
		windows.TerminateProcess(cpty.pi.Process, 0)
	}
}

// SetTerminalSize resizes the ConPTY to the specified dimensions.
func (cpty *ConPTY) SetTerminalSize(size TerminalSize) error {
	ret, _, _ := resizePseudoConsole.Call(uintptr(cpty.hPC), size.serialize())
	if ret != uintptr(0) {
		return fmt.Errorf("ResizePseudoConsole failed with status 0x%x", ret)
	}

	return nil
}

// ###################################################
// ############ Terminal size ########################
// ###################################################

func (size *TerminalSize) serialize() uintptr {
	return uintptr((int32(size.Rows) << 16) | int32(size.Cols))
}

// GetTerminalSize retrieves the current terminal size from the console.
// It returns the terminal dimensions or an error if the console information cannot be retrieved.
func GetTerminalSize() (size TerminalSize, err error) {
	var csbi windows.ConsoleScreenBufferInfo
	hConsole, err := windows.GetStdHandle(windows.STD_OUTPUT_HANDLE)
	if err != nil {
		return size, err
	}

	err = windows.GetConsoleScreenBufferInfo(hConsole, &csbi)
	if err != nil {
		return size, err
	}

	return TerminalSize{
		Rows: int(csbi.Size.Y),
		Cols: int(csbi.Size.X),
	}, nil
}
