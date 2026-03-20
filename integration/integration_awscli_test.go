package integration_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	awsCLIAccessKey = "testkey"
	awsCLISecretKey = "testsecret"
)

func TestAWSCLI(t *testing.T) {
	awsBin, err := exec.LookPath("aws")
	if err != nil {
		t.Skip("aws CLI not installed")
	}

	_, u1, _, ca1 := newClient(t, "awscli1", firstNode)
	defer ca1()
	time.Sleep(500 * time.Millisecond)

	var (
		bucket   = "clibucket"
		key      = "hello.txt"
		content  = []byte("hello from AWS CLI")
		s3URI    = fmt.Sprintf("s3://%s/%s", bucket, key)
		tmpDir   = t.TempDir()
		upload   = filepath.Join(tmpDir, "upload.txt")
		download = filepath.Join(tmpDir, "download.txt")
	)

	require.NoError(t, os.WriteFile(upload, content, 0644))

	run := func(args ...string) (string, error) {
		all := append([]string{"s3"}, args...)
		all = append(all, "--endpoint-url", u1, "--no-sign-request")
		cmd := exec.Command(awsBin, all...)
		cmd.Env = append(os.Environ(), "AWS_DEFAULT_REGION=us-east-1")
		out, err := cmd.CombinedOutput()
		return string(out), err
	}

	t.Run("Upload", func(t *testing.T) {
		out, err := run("cp", upload, s3URI)
		require.NoError(t, err, out)
	})

	t.Run("Download", func(t *testing.T) {
		out, err := run("cp", s3URI, download)
		require.NoError(t, err, out)
		got, err := os.ReadFile(download)
		require.NoError(t, err)
		assert.Equal(t, content, got)
	})

	t.Run("Delete", func(t *testing.T) {
		out, err := run("rm", s3URI)
		require.NoError(t, err, out)
	})

	t.Run("DownloadAfterDelete_Fails", func(t *testing.T) {
		_, err := run("cp", s3URI, download)
		require.Error(t, err)
	})
}

func TestAWSCLIWithAuth(t *testing.T) {
	awsBin, err := exec.LookPath("aws")
	if err != nil {
		t.Skip("aws CLI not installed")
	}

	u1, ca1 := newClientWithAuth(t, "awscliauth1", firstNode, awsCLIAccessKey, awsCLISecretKey)
	defer ca1()
	time.Sleep(500 * time.Millisecond)

	var (
		bucket   = "authbucket"
		key      = "hello.txt"
		content  = []byte("hello from AWS CLI with auth")
		s3URI    = fmt.Sprintf("s3://%s/%s", bucket, key)
		tmpDir   = t.TempDir()
		upload   = filepath.Join(tmpDir, "upload.txt")
		download = filepath.Join(tmpDir, "download.txt")
	)

	require.NoError(t, os.WriteFile(upload, content, 0644))

	runWithAuth := func(accessKey, secretKey string, args ...string) (string, error) {
		all := append([]string{"s3"}, args...)
		all = append(all, "--endpoint-url", u1)
		cmd := exec.Command(awsBin, all...)
		cmd.Env = append(os.Environ(),
			"AWS_DEFAULT_REGION=us-east-1",
			"AWS_ACCESS_KEY_ID="+accessKey,
			"AWS_SECRET_ACCESS_KEY="+secretKey,
		)
		out, err := cmd.CombinedOutput()
		return string(out), err
	}

	run := func(args ...string) (string, error) {
		return runWithAuth(awsCLIAccessKey, awsCLISecretKey, args...)
	}

	t.Run("UploadWithoutCredentials_Fails", func(t *testing.T) {
		cmd := exec.Command(awsBin, "s3", "cp", upload, s3URI, "--endpoint-url", u1, "--no-sign-request")
		cmd.Env = append(os.Environ(), "AWS_DEFAULT_REGION=us-east-1")
		out, err := cmd.CombinedOutput()
		require.Error(t, err, string(out))
	})

	t.Run("UploadWithWrongCredentials_Fails", func(t *testing.T) {
		out, err := runWithAuth("wrongkey", "wrongsecret", "cp", upload, s3URI)
		require.Error(t, err, out)
	})

	t.Run("Upload", func(t *testing.T) {
		out, err := run("cp", upload, s3URI)
		require.NoError(t, err, out)
	})

	t.Run("Download", func(t *testing.T) {
		out, err := run("cp", s3URI, download)
		require.NoError(t, err, out)
		got, err := os.ReadFile(download)
		require.NoError(t, err)
		assert.Equal(t, content, got)
	})

	t.Run("Delete", func(t *testing.T) {
		out, err := run("rm", s3URI)
		require.NoError(t, err, out)
	})

	t.Run("DownloadAfterDelete_Fails", func(t *testing.T) {
		_, err := run("cp", s3URI, download)
		require.Error(t, err)
	})
}
