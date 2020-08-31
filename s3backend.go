package main

import (
	"bytes"
	"context"
	"github.com/minio/minio-go/v7"
)

type S3Backend struct {
	Client   *minio.Client
	Bucket   string
	BasePath string
}

func (s *S3Backend) GetFile(filename string) ([]byte, error) {
	r, err := s.Client.GetObject(context.Background(), s.Bucket, s.BasePath+filename, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(r)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (s *S3Backend) PutFile(filename string, content *bytes.Buffer) error {
	_, err := s.Client.PutObject(context.Background(), s.Bucket, s.BasePath+filename, content, -1,
		minio.PutObjectOptions{})
	return err
}

func (s *S3Backend) GetDirectories(dirname string) ([]string, error) {
	prefix := s.BasePath + dirname
	var result []string
	for object := range s.Client.ListObjects(
		context.Background(),
		s.Bucket,
		minio.ListObjectsOptions{Prefix: prefix, Recursive: false},
	) {
		if object.Err != nil {
			return nil, object.Err
		}
		if object.Key[len(object.Key)-1] == '/' {
			// chop trailing slash and prefix
			result = append(result, object.Key[len(prefix):len(object.Key)-1])
		}
	}
	return result, nil
}

func (s *S3Backend) GetFiles(dirname string) ([]string, error) {
	prefix := s.BasePath + dirname
	var result []string
	for object := range s.Client.ListObjects(
		context.Background(),
		s.Bucket,
		minio.ListObjectsOptions{Prefix: prefix, Recursive: false},
	) {
		if object.Err != nil {
			return nil, object.Err
		}
		if object.Key[len(object.Key)-1] != '/' {
			// chop trailing slash and prefix
			result = append(result, object.Key[len(prefix):])
		}
	}
	return result, nil
}

func (s *S3Backend) MkdirAll(dirname string) error {
	// I think this is not necessary on S3
	return nil
}

func (s *S3Backend) FileExists(filename string) bool {
	_, err := s.Client.StatObject(context.Background(), s.Bucket, s.BasePath+filename,
		minio.StatObjectOptions{})
	return err == nil
}
