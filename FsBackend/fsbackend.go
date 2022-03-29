package FsBackend

import (
	"bytes"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
)

type FsBackend struct {
	BasePath string
}

func (b *FsBackend) GetFile(filename string) ([]byte, error) {
	f, err := os.Open(filepath.Join(b.BasePath, filename))
	defer f.Close()
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(f)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (b *FsBackend) PutFile(filename string, content *bytes.Buffer) error {
	return ioutil.WriteFile(filepath.Join(b.BasePath, filename), content.Bytes(), 0755)
}

func (b *FsBackend) FileExists(filename string) bool {
	_, err := os.Stat(filepath.Join(b.BasePath, filename))
	return err == nil
}

func (b *FsBackend) MkdirAll(dirname string) error {
	return os.MkdirAll(filepath.Join(b.BasePath, dirname), 0755)
}

func (b *FsBackend) GetDirectories(dirname string) ([]string, error) {
	files, err := ioutil.ReadDir(filepath.Join(b.BasePath, dirname))
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

func (b *FsBackend) GetFiles(dirname string) ([]string, error) {
	files, err := ioutil.ReadDir(filepath.Join(b.BasePath, dirname))
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

func (b *FsBackend) GetFilesRecursive(dirname string) ([]string, error) {
	var results []string

	err := filepath.Walk(filepath.Join(b.BasePath, dirname), func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			info.Name()
			results = append(results, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}
