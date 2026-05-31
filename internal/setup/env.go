package setup

import (
	"io/fs"
	"os"
)

type CommandDetector interface {
	LookPath(name string) (string, bool)
}

type FileSystem interface {
	ReadFile(path string) ([]byte, error)
	Stat(path string) (fs.FileInfo, error)
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

func (OSFileSystem) Stat(path string) (fs.FileInfo, error) {
	return os.Stat(path)
}
