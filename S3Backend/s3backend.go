package S3Backend

import (
	"bytes"
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"os"
	"strings"
)

type S3Backend struct {
	Client   *minio.Client
	Bucket   string
	BasePath string
}

func NewS3Backend(path string) (*S3Backend, error) {
	pathComponents := strings.Split(path[5:], "/")

	minioClient, err := minio.New(pathComponents[0], &minio.Options{
		Creds:  credentials.NewStaticV4(os.Getenv(pathComponents[0]+"_ACCESS_KEY_ID"), os.Getenv(pathComponents[0]+"_SECRET_ACCESS_KEY"), ""),
		Secure: true,
	})

	if err != nil {
		return nil, err
	}

	return &S3Backend{
		Client:   minioClient,
		Bucket:   pathComponents[1],
		BasePath: strings.Join(pathComponents[2:], "/"),
	}, nil
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
	opts := minio.PutObjectOptions{}
	opts.UserMetadata = make(map[string]string)
	opts.UserMetadata["x-amz-acl"] = "public-read"
	_, err := s.Client.PutObject(context.Background(), s.Bucket, s.BasePath+filename, content, int64(content.Len()),
		opts)
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
			if len(prefix) != len(object.Key) {
				result = append(result, object.Key[len(prefix):len(object.Key)-1])
			}
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
			// chop prefix
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
