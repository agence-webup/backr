package backr

// Executor defines some methods necessary to execute a backup
type Executor interface {
	GetOutputFileExtension() string
	Execute(workingDir string, output string) error
}
