package s3

import (
	"io"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type Client struct {
	bucket string
	s3     *s3.S3
}

func New(bucket string) (*Client, error) {
	s3Config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(os.Getenv("S3_KEY_ID"), os.Getenv("S3_APPLICATION_KEY"), ""),
		Endpoint:         aws.String(os.Getenv("S3_ENDPOINT")),
		Region:           aws.String(os.Getenv("S3_REGION")),
		S3ForcePathStyle: aws.Bool(true),
	}
	newSession, err := session.NewSession(s3Config)

	if err != nil {
		return nil, err
	}

	client := s3.New(newSession)

	return &Client{bucket: bucket, s3: client}, nil
}

func (client *Client) Upload(path string, reader io.ReadSeeker) error {
	_, err := client.s3.PutObject(&s3.PutObjectInput{
		Body:   reader,
		Bucket: aws.String(client.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return err
	}
	return nil
}
