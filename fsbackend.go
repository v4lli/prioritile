package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"os"
)

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

func (b *FsBackend) GetFileWriter(filename string) (*io.PipeWriter, error) {
	r, w := io.Pipe()
	go func() {
		buf := new(bytes.Buffer)
		buf.ReadFrom(r)
		err := ioutil.WriteFile(b.BasePath+filename, buf.Bytes(), 0755)
		if err != nil {
			log.Println(err)
		}
		// XXX not sure if it's a good idea to rely on GC to call close on file...
	}()
	return w, nil
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
