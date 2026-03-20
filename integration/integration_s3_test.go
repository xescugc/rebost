package integration_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newS3Client(t *testing.T, nodeURL, accessKey, secretKey string) *s3.Client {
	t.Helper()
	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		),
		awsconfig.WithRegion("us-east-1"),
	)
	require.NoError(t, err)
	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(nodeURL)
		o.UsePathStyle = true
	})
}

func TestS3CRUD(t *testing.T) {
	var (
		bucket  = "testbucket"
		key     = "myfile.txt"
		content = []byte("hello from s3 client")
		ctx     = context.Background()
	)

	_, u1, _, ca1 := newClient(t, "s3n1", firstNode)
	defer ca1()

	time.Sleep(500 * time.Millisecond)

	s3cl := newS3Client(t, u1, "anykey", "anysecret")

	t.Run("CreateBucket", func(t *testing.T) {
		_, err := s3cl.CreateBucket(ctx, &s3.CreateBucketInput{
			Bucket: aws.String(bucket),
		})
		require.NoError(t, err)
	})

	t.Run("PutObject", func(t *testing.T) {
		_, err := s3cl.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
			Body:   bytes.NewReader(content),
		})
		require.NoError(t, err)
	})

	t.Run("HeadObject", func(t *testing.T) {
		_, err := s3cl.HeadObject(ctx, &s3.HeadObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
		require.NoError(t, err)
	})

	t.Run("GetObject", func(t *testing.T) {
		resp, err := s3cl.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
		require.NoError(t, err)
		defer resp.Body.Close()

		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		assert.Equal(t, content, buf.Bytes())
	})

	t.Run("DeleteObject", func(t *testing.T) {
		_, err := s3cl.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
		require.NoError(t, err)
	})

	t.Run("GetObjectAfterDelete", func(t *testing.T) {
		_, err := s3cl.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
		require.Error(t, err)
		var nsk *types.NoSuchKey
		assert.ErrorAs(t, err, &nsk)
	})

	t.Run("DeleteBucket", func(t *testing.T) {
		_, err := s3cl.DeleteBucket(ctx, &s3.DeleteBucketInput{
			Bucket: aws.String(bucket),
		})
		require.NoError(t, err)
	})
}

func TestS3ListNotImplemented(t *testing.T) {
	var (
		bucket = "testbucket"
		ctx    = context.Background()
	)

	_, u1, _, ca1 := newClient(t, "s3list1", firstNode)
	defer ca1()

	time.Sleep(500 * time.Millisecond)

	s3cl := newS3Client(t, u1, "anykey", "anysecret")

	_, err := s3cl.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "NotImplemented") || strings.Contains(err.Error(), "operation error"))
}

func TestS3MultipartNotImplemented(t *testing.T) {
	var (
		bucket = "testbucket"
		key    = "multipart.txt"
		ctx    = context.Background()
	)

	_, u1, _, ca1 := newClient(t, "s3mp1", firstNode)
	defer ca1()

	time.Sleep(500 * time.Millisecond)

	s3cl := newS3Client(t, u1, "anykey", "anysecret")

	_, err := s3cl.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "NotImplemented") || strings.Contains(err.Error(), "operation error"))
}

func TestS3Auth(t *testing.T) {
	var (
		bucket    = "authbucket"
		key       = "secret.txt"
		content   = []byte("authenticated content")
		accessKey = "testkey"
		secretKey = "testsecret"
		ctx       = context.Background()
	)

	u1, ca1 := newClientWithAuth(t, "s3auth1", firstNode, accessKey, secretKey)
	defer ca1()

	time.Sleep(500 * time.Millisecond)

	t.Run("RequestWithoutCredentials_Returns403", func(t *testing.T) {
		// Send a raw request with no Authorization header
		resp, err := http.Post(u1+"/"+bucket+"/"+key, "application/octet-stream", bytes.NewReader(content))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("RequestWithWrongCredentials_Returns403", func(t *testing.T) {
		wrongClient := newS3Client(t, u1, "wrongkey", "wrongsecret")
		_, err := wrongClient.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
			Body:   bytes.NewReader(content),
		})
		require.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "AccessDenied") || strings.Contains(err.Error(), "403"))
	})

	t.Run("RequestWithCorrectCredentials_Succeeds", func(t *testing.T) {
		authClient := newS3Client(t, u1, accessKey, secretKey)
		_, err := authClient.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
			Body:   bytes.NewReader(content),
		})
		require.NoError(t, err)
	})

	t.Run("InternalConfigRoute_AccessibleWithoutAuth", func(t *testing.T) {
		resp, err := http.Get(u1 + "/config")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestS3AuthWriteMode(t *testing.T) {
	var (
		bucket    = "writebucket"
		key       = "public.txt"
		content   = []byte("public read content")
		accessKey = "testkey"
		secretKey = "testsecret"
		ctx       = context.Background()
	)

	u1, ca1 := newClientWithAuthMode(t, "s3writemode1", firstNode, accessKey, secretKey, "write")
	defer ca1()

	time.Sleep(500 * time.Millisecond)

	authClient := newS3Client(t, u1, accessKey, secretKey)

	// Upload the file with credentials so reads can be tested
	t.Run("WriteWithoutCredentials_Fails", func(t *testing.T) {
		resp, err := http.Post(u1+"/"+bucket+"/"+key, "application/octet-stream", bytes.NewReader(content))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("WriteWithCorrectCredentials_Succeeds", func(t *testing.T) {
		_, err := authClient.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
			Body:   bytes.NewReader(content),
		})
		require.NoError(t, err)
	})

	t.Run("ReadWithoutCredentials_Succeeds", func(t *testing.T) {
		resp, err := http.Get(u1 + "/" + bucket + "/" + key)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		got, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, content, got)
	})

	t.Run("HeadWithoutCredentials_Succeeds", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodHead, u1+"/"+bucket+"/"+key, nil)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
