package storage

import (
	"bytes"
	"errors"
	"github.com/aviddiviner/inc/util"
	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
	"io"
	"net/http"
)

const c_CONTENT_TYPE = "binary/octet-stream"
const c_DEFAULT_ACL = s3.Private

// S3Connection stores data remotely to a private S3 bucket.
type S3Connection struct {
	client *s3.S3
	bucket *s3.Bucket
}

// NewS3Connection connects to S3 in the given region, using the AWS credentials
// provided. If accessKey / secretKey are blank, will try to either read them
// from ENV[AWS_CREDENTIAL_FILE] (defaults to $HOME/.aws/credentials) or use the
// values in ENV[AWS_ACCESS_KEY] and ENV[AWS_SECRET_KEY].
func NewS3Connection(region, bucket, accessKey, secretKey string) (*S3Connection, error) {
	auth, err := aws.GetAuth(accessKey, secretKey)
	if err != nil {
		return nil, err
	}
	client := s3.New(auth, aws.Regions[region])
	return &S3Connection{client, client.Bucket(bucket)}, nil
}

func (c *S3Connection) Exists() (bool, error) {
	resp, err := c.client.ListBuckets()
	if err != nil {
		return false, err
	}
	for _, bucket := range resp.Buckets {
		if c.bucket.Name == bucket.Name {
			return true, nil
		}
	}
	return false, nil
}

func (c *S3Connection) Create() error {
	return c.bucket.PutBucket(c_DEFAULT_ACL)
}

type HeadError struct {
	Response *http.Response
	Err      error
}

func (e *HeadError) Error() string {
	return e.Response.Proto + " " + e.Response.Status + ": " + e.Err.Error()
}

func (c *S3Connection) Size(key string) (int, error) {
	resp, err := c.bucket.Head(key)
	defer resp.Body.Close()

	if err != nil {
		return 0, err
	}
	if resp.StatusCode != http.StatusOK {
		return 0, &HeadError{resp, errors.New("failed HEAD request")}
	}
	if resp.ContentLength < 0 {
		return 0, &HeadError{resp, errors.New("unknown content length")}
	}
	return int(resp.ContentLength), nil
}

func (c *S3Connection) GetReader(key string) (io.Reader, error) {
	rc, err := c.bucket.GetReader(key)
	if err != nil {
		if rc != nil {
			rc.Close()
		}
		return nil, err
	}
	return &util.AutoCloseReader{rc}, nil
}

func (c *S3Connection) PutReader(key string, r io.Reader) (length int, err error) {
	// TODO: S3 multipart uploads should be a better way.
	var buf bytes.Buffer
	n, err := io.Copy(&buf, r) // we need the length :(
	if err != nil {
		return
	}
	err = c.bucket.PutReader(key, &buf, n, c_CONTENT_TYPE, c_DEFAULT_ACL)
	if err != nil {
		return
	}
	length = int(n)
	return
}

func (c *S3Connection) IsNotExist(err error) bool {
	switch e := err.(type) {
	case *s3.Error:
		switch e.Code {
		case "NoSuchKey":
			return true
		}
	}
	return false
}
