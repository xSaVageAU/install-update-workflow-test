//go:build windows

package uninstall

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

const (
	createNoWindow         = 0x08000000
	createBreakawayFromJob = 0x01000000
)

// removeBinary deletes the running executable.
//
// Windows cannot delete or overwrite a file while it's mapped into a
// running process, but (as in internal/update/apply.go) it can rename one.
// So this renames the binary aside, then starts a short-lived, detached
// helper process that waits a moment and deletes the renamed file once
// this process has exited and released its handle on it.
//
// The helper is started with CREATE_BREAKAWAY_FROM_JOB so it survives even
// if the parent process is running inside a job object that kills its
// process tree on exit (as some terminal hosts do); if the job doesn't
// permit breakaway, that flag is dropped and the helper is retried without
// it.
func removeBinary(path string) error {
	renamed := path + ".uninstall"
	os.Remove(renamed) // clean up any leftover from a previous attempt

	if err := os.Rename(path, renamed); err != nil {
		return fmt.Errorf("moving binary aside: %w", err)
	}

	script := fmt.Sprintf(`ping 127.0.0.1 -n 2 >nul & del "%s"`, renamed)
	if startDetached(script, createNoWindow|createBreakawayFromJob) == nil {
		return nil
	}
	if err := startDetached(script, createNoWindow); err != nil {
		return fmt.Errorf("moved binary to %s but could not schedule its cleanup: %w (delete it manually)", renamed, err)
	}
	return nil
}

func startDetached(script string, flags uint32) error {
	cmd := exec.Command("cmd")
	// Go's os/exec quotes each argument individually for CreateProcess,
	// which mangles the embedded quotes the del command needs around a
	// path that might contain spaces. CmdLine bypasses that and is used
	// as the literal command line instead of being built from Args.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: flags,
		CmdLine:       "cmd /C " + script,
	}
	return cmd.Start()
}
