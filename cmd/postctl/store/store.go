package store

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pkg/errors"
)

type Store struct {
	bucketName string
	keyPrefix  string
	sess       *session.Session
}

func New(bucketName string, key string) *Store {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	return &Store{
		bucketName: bucketName,
		keyPrefix:  key,
		sess:       sess,
	}
}

func (s *Store) Upload(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", errors.Wrap(err, "store: open file")
	}
	defer file.Close()

	filename := filepath.Base(path)
	uploadpath := filepath.Join(s.keyPrefix, filename)

	t := contentType(file)

	uploader := s3manager.NewUploader(s.sess)
	out, err := uploader.Upload(&s3manager.UploadInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(uploadpath),
		Body:        file,
		ContentType: aws.String(t),
		ACL:         aws.String("public-read"),
	})
	if err != nil {
		return "", errors.Wrap(err, "upload media")
	}
	return out.Location, nil
}

func contentType(file *os.File) string {
	defer func() {
		_, _ = file.Seek(0, 0)
	}()

	fileData, err := ioutil.ReadAll(file)
	if err != nil {
		return "application/octet-stream"
	}

	return http.DetectContentType(fileData)
}
