package gptscript

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func appendExtraFiles(cmd *exec.Cmd, extraFiles ...*os.File) {
	additionalInheritedHandles := make([]syscall.Handle, 0, len(extraFiles))
	for _, f := range extraFiles {
		additionalInheritedHandles = append(additionalInheritedHandles, syscall.Handle(f.Fd()))
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{AdditionalInheritedHandles: additionalInheritedHandles}

	cmd.Args = append(cmd.Args[:1], append([]string{fmt.Sprintf("--events-stream-to=fd://%d", extraFiles[len(extraFiles)-1].Fd())}, cmd.Args[1:]...)...)
}
