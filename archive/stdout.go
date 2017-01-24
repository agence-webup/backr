package archive

import (
	"os"
	"os/exec"
)

// Stdout executes a custom command and get stdout output to save a backup archive
type Stdout struct {
	OutputFileExtension string
	Command             []string
}

// GetOutputFileExtension implements Executor interface by returning
func (s Stdout) GetOutputFileExtension() string {
	return s.OutputFileExtension
}

// Execute implements Executor interface
func (s Stdout) Execute(workingDir string, output string) error {

	var cmd *exec.Cmd
	if len(s.Command) > 1 {
		cmd = exec.Command(s.Command[0], s.Command[1:]...)
	} else {
		cmd = exec.Command(s.Command[0])
	}

	outputFile, err := os.Create(output)
	if err != nil {
		return err
	}

	cmd.Dir = workingDir
	cmd.Stdout = outputFile
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
