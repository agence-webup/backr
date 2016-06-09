package execution

import (
	"fmt"
	"os"
	"os/exec"
)

// Pliz executes a pliz backup inside the specified directory
type Pliz struct {
}

// GetOutputFileExtension implements Executor interface by returning
func (pliz Pliz) GetOutputFileExtension() string {
	return "tar.gz"
}

// Execute implements Executor interface
func (pliz Pliz) Execute(workingDir string, output string) error {

	cmd := exec.Command("pliz", "backup", "-q", "--files", "--db", "-o", output)
	cmd.Dir = workingDir
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin

	fmt.Println("pliz", "backup", "-q", "--files", "--db", "-o", output)
	fmt.Println("into", workingDir)

	return cmd.Run()
}
