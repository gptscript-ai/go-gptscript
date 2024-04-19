//go:build !windows

package gptscript

import (
	"fmt"
	"os"
	"os/exec"
)

func appendExtraFiles(cmd *exec.Cmd, extraFiles ...*os.File) {
	cmd.ExtraFiles = append(cmd.ExtraFiles, extraFiles...)
	cmd.Args = append(cmd.Args[:1], append([]string{"--events-stream-to", fmt.Sprintf("fd://%d", len(cmd.ExtraFiles)+2)}, cmd.Args[1:]...)...)
}
