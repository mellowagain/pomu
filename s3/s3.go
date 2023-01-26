package s3

import (
	"io"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type Client struct {
	bucket   string
	s3       *s3.S3
	uploader *s3manager.Uploader
}

func New(bucket string) (*Client, error) {
	s3Config := &aws.Config{
		Credentials:             credentials.NewStaticCredentials(os.Getenv("S3_KEY_ID"), os.Getenv("S3_APPLICATION_KEY"), ""),
		Endpoint:                aws.String(os.Getenv("S3_ENDPOINT")),
		Region:                  aws.String(os.Getenv("S3_REGION")),
		S3ForcePathStyle:        aws.Bool(true),
		MaxRetries:              aws.Int(10),
		EnforceShouldRetryCheck: aws.Bool(true),
	}
	newSession, err := session.NewSession(s3Config)

	if err != nil {
		return nil, err
	}

	uploader := s3manager.NewUploader(newSession)
	client := s3.New(newSession)
	return &Client{bucket: bucket, s3: client, uploader: uploader}, nil
}

func (client *Client) Upload(path string, reader io.Reader, contentType string) error {
	if len(contentType) <= 0 {
		contentType = "binary/octet-stream"
	}

	_, err := client.uploader.Upload(&s3manager.UploadInput{
		Body:               reader,
		Bucket:             aws.String(client.bucket),
		Key:                aws.String(path),
		ContentType:        aws.String(contentType),
		ContentDisposition: aws.String("inline"),
	}, func(u *s3manager.Uploader) {
		u.LeavePartsOnError = true
	})
	return err
}
