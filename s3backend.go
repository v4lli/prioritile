package main

import (
	"context"
	"github.com/minio/minio-go/v7"
	"io"
	"log"
)

type S3Backend struct {
	Client   *minio.Client
	Bucket   string
	BasePath string
}

func (s *S3Backend) GetDirectories(dirname string) ([]string, error) {
	log.Printf("GetDirectories %s", dirname)

	var result []string
	for object := range s.Client.ListObjects(
		context.Background(),
		s.Bucket,
		minio.ListObjectsOptions{Prefix: dirname, Recursive: false},
	) {
		if object.Err != nil {
			panic(object.Err)
			return nil, object.Err
		}
		if object.Key[len(object.Key)-1] == '/' {
			// chop trailing slash and prefix
			result = append(result, object.Key[len(dirname):len(object.Key)-1])
		}
	}
	log.Println(result)
	return result, nil
}

func (s *S3Backend) GetFiles(dirname string) ([]string, error) {
	log.Printf("GetFiles %s", dirname)

	var result []string
	for object := range s.Client.ListObjects(
		context.Background(),
		s.Bucket,
		minio.ListObjectsOptions{Prefix: dirname, Recursive: false},
	) {
		if object.Err != nil {
			return nil, object.Err
		}
		if object.Key[len(object.Key)-1] != '/' {
			// chop trailing slash and prefix
			result = append(result, object.Key[len(dirname):])
		}
	}
	log.Println(result)
	return result, nil
}

func (s *S3Backend) MkdirAll(dirname string) error {
	// I think this is not necessary on S3
	return nil
}

func (s *S3Backend) GetFileReader(filename string) (io.Reader, error) {
	//log.Printf("GET %s\n", filename)
	r, err := s.Client.GetObject(context.Background(), s.Bucket, s.BasePath[:len(s.BasePath)-1]+filename, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (s *S3Backend) GetFileWriter(filename string) (*io.PipeWriter, error) {
	//log.Printf("PUT %s\n", filename)
	r, w := io.Pipe()
	go func() {
		_, err := s.Client.PutObject(context.Background(), s.Bucket, s.BasePath[:len(s.BasePath)-1]+filename,
			r, -1, minio.PutObjectOptions{})
		if err != nil {
			log.Println(err)
		}
	}()
	return w, nil
}

// XXX it's inconsistant that this auto-prepends the basePath
// XXX handle double slashes better
func (s *S3Backend) FileExists(filename string) bool {
	_, err := s.Client.StatObject(context.Background(), s.Bucket, s.BasePath[:len(s.BasePath)-1]+filename, minio.StatObjectOptions{})
	if err == nil {
		return true
	} else {
		return false
	}
}

func (s *S3Backend) GetBasePath() string {
	return s.BasePath
}
