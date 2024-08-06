package oss

import (
	"bytes"
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/iimeta/fastapi-sdk/logger"
)

type Oss struct {
	Endpoint  string
	Region    string
	AccessKey string
	SecretKey string
	Bucket    string
	Domain    string
}

func (oss *Oss) Upload(fileData []byte, fileKey string) (string, error) {
	sess := session.Must(session.NewSession(&aws.Config{
		Region:      aws.String(oss.Region),
		Endpoint:    aws.String(oss.Endpoint),
		Credentials: credentials.NewStaticCredentials(oss.AccessKey, oss.SecretKey, ""),
	}))

	s3Client := s3.New(sess)
	_, err := s3Client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(oss.Bucket),
		Key:    aws.String(fileKey),
		Body:   bytes.NewReader(fileData),
	})
	if err != nil {
		logger.Errorf(context.Background(), "upload failed: %s", err)
		return "", err
	}

	var url string
	if oss.Domain == "" {
		url = fmt.Sprintf("%s/%s/%s", s3Client.Endpoint, oss.Bucket, fileKey)
	} else {
		url = fmt.Sprintf("%s/%s", oss.Domain, fileKey)
	}
	return url, nil
}
