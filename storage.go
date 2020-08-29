package main

import (
	"io"
	"io/ioutil"
	"os"
)

func stringToBackend(input string) (StorageBackend, error) {
	//if input[0:4] == "s3://" {
	//	panic("S3 backend not implemented yet")
	//}

	basePath := input
	if basePath[len(basePath)-1] != '/' {
		basePath = basePath + "/"
	}

	_, err := os.Stat(basePath)
	if os.IsNotExist(err) {
		return nil, err
	}

	return &FsBackend{BasePath: basePath}, nil
}

type StorageBackend interface {
	GetDirectories(dirname string) ([]string, error)
	GetFiles(dirname string) ([]string, error)
	MkdirAll(dirname string) error
	GetFileReader(filename string) (io.Reader, error)
	GetFileWriter(filename string) (io.Writer, error)
	FileExists(filename string) bool
	GetBasePath() string
}

type FsBackend struct {
	BasePath string
}

func (b *FsBackend) GetFileReader(filename string) (io.Reader, error) {
	f, err := os.Open(b.BasePath + filename)
	if err != nil {
		return nil, err
	}
	// XXX not sure if it's a good idea to rely on GC to call close on file...
	return f, nil
}

func (b *FsBackend) GetFileWriter(filename string) (io.Writer, error) {
	f, err := os.Create(b.BasePath + filename)
	if err != nil {
		return nil, err
	}
	// XXX not sure if it's a good idea to rely on GC to call close on file...
	return f, nil
}

func (b *FsBackend) FileExists(filename string) bool {
	_, err := os.Stat(b.BasePath + filename)
	return err == nil
}

func (*FsBackend) MkdirAll(dirname string) error {
	return os.MkdirAll(dirname, 0755)
}

func (*FsBackend) GetDirectories(dirname string) ([]string, error) {
	files, err := ioutil.ReadDir(dirname)
	if err != nil {
		return nil, err
	}

	var results []string
	for idx, file := range files {
		if file.IsDir() {
			results = append(results, files[idx].Name())
		}
	}
	return results, nil
}

func (*FsBackend) GetFiles(dirname string) ([]string, error) {
	files, err := ioutil.ReadDir(dirname)
	if err != nil {
		return nil, err
	}

	var results []string
	for idx, file := range files {
		if !file.IsDir() {
			results = append(results, files[idx].Name())
		}
	}
	return results, nil
}

func (b *FsBackend) GetBasePath() string {
	return b.BasePath
}
