package setup

import (
	"io/fs"
	"os"
	"os/exec"
)

type CommandDetector interface {
	LookPath(name string) (string, bool)
}

type ExecCommandDetector struct{}

func (ExecCommandDetector) LookPath(name string) (string, bool) {
	path, err := exec.LookPath(name)
	return path, err == nil
}

type FileSystem interface {
	ReadFile(path string) ([]byte, error)
	Stat(path string) (fs.FileInfo, error)
}

type WritableFileSystem interface {
	FileSystem
	WriteFile(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
}

type SetupEnv struct {
	ProjectDir string
	HomeDir    string
	GOOS       string
	Commands   CommandDetector
	FS         FileSystem
}

type OSFileSystem struct{}

func (OSFileSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (OSFileSystem) WriteFile(path string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(path, data, perm)
}

func (OSFileSystem) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (OSFileSystem) Stat(path string) (fs.FileInfo, error) {
	return os.Stat(path)
}

func withDefaults(env SetupEnv) SetupEnv {
	if env.FS == nil {
		env.FS = OSFileSystem{}
	}
	return env
}
