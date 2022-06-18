package fs

import (
	"os"
	"path"
	"pomu/s3"
)

type FS interface {
	Write(relativePath string, data []byte) error
	Read(relativePath string) ([]byte, error)
}

type FilesystemFS struct {
	rootPath string
}

type S3FS struct {
	s3 *s3.Client
}

// Read implements FS
func (*S3FS) Read(relativePath string) ([]byte, error) {
	panic("unimplemented")
}

// Write implements FS
func (*S3FS) Write(relativePath string, data []byte) error {
	panic("unimplemented")
}

func NewFilesystemFS(rootPath string) FS {
	return &FilesystemFS{
		rootPath,
	}
}

// Read implements FS
func (fs *FilesystemFS) Read(relativePath string) ([]byte, error) {
	path := path.Join(fs.rootPath, relativePath)

	return os.ReadFile(path)
}

// Write implements FS
func (fs *FilesystemFS) Write(relativePath string, data []byte) error {
	wholePath := path.Join(fs.rootPath, relativePath)
	basePath := path.Dir(wholePath)

	if err := os.MkdirAll(basePath, os.ModePerm); err != nil {
		return err
	}

	return os.WriteFile(wholePath, data, os.ModePerm)
}

var _ FS = (*FilesystemFS)(nil)
var _ FS = (*S3FS)(nil)
