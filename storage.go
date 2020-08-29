package main

import (
	"errors"
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
	ReadDir(dirname string) ([]string, error)
	GetBasePath() string
}

type FsBackend struct {
	BasePath string
}

func (*FsBackend) ReadDir(dirname string) ([]string, error) {
	files, err := ioutil.ReadDir(dirname)
	if err != nil {
		return nil, err
	}
	results := make([]string, len(files))
	for idx, file := range files {
		if !file.IsDir() {
			return nil, errors.New("not a file")
		}
		results[idx] = files[idx].Name()
	}
	return results, nil
}

func (b *FsBackend) GetBasePath() string {
	return b.BasePath
}
