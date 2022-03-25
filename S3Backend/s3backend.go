package S3Backend

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3Backend struct {
	Client   *minio.Client
	Bucket   string
	BasePath string
}

func NewS3Backend(path string, timeout int) (*S3Backend, error) {
	url_parsed, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	var secure bool
	if url_parsed.Scheme == "http" {
		secure = false
	} else if url_parsed.Scheme == "https" {
		secure = true
	} else {
		return nil, fmt.Errorf("invalid scheme: %s. valid schemes are: http, https", url_parsed.Scheme)
	}
	host := url_parsed.Host
	pathComponents := strings.Split(url_parsed.Path, "/")
	if len(pathComponents) == 1 {
		return nil, fmt.Errorf("Invalid path (maybe you forgot add the bucket name to the url?)")
	}
	bucket := pathComponents[1]

	accessKey := os.Getenv(host + "_" + bucket + "_ACCESS_KEY_ID")
	secretKey := os.Getenv(host + "_" + bucket + "_SECRET_ACCESS_KEY")
	transport, err := minio.DefaultTransport(secure)
	if err != nil {
		return nil, err
	}
	transport.ResponseHeaderTimeout = time.Duration(timeout) * time.Second
	minioClient, err := minio.New(host, &minio.Options{
		Creds:     credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure:    secure,
		Transport: transport,
	})

	if err != nil {
		return nil, err
	}

	return &S3Backend{
		Client:   minioClient,
		Bucket:   bucket,
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
