package client_test

import (
	"bytes"
	"context"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/rebost/client"
	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/mock"
	"github.com/xescugc/rebost/storing"
)

func TestGetFile(t *testing.T) {
	var (
		key = "fileName"
		// Kind of a big content just to test
		content = make([]byte, 6000)
		ctrl    = gomock.NewController(t)
	)
	st := mock.NewStoring(ctrl)
	defer ctrl.Finish()

	st.EXPECT().GetFile(gomock.Any(), key).Return(ClosingBuffer{Buffer: bytes.NewBuffer(content)}, nil)

	h := storing.MakeHandler(st)
	server := httptest.NewServer(h)
	c, err := client.New(server.URL)
	require.NoError(t, err)

	ior, err := c.GetFile(context.Background(), key)
	require.NoError(t, err)

	bu := new(bytes.Buffer)
	require.NoError(t, err)
	io.Copy(bu, ior)
	assert.Equal(t, content, bu.Bytes())
}

func TestHasFile(t *testing.T) {
	t.Run("True", func(t *testing.T) {
		var (
			key  = "fileName"
			ctrl = gomock.NewController(t)
		)
		st := mock.NewStoring(ctrl)
		defer ctrl.Finish()

		st.EXPECT().HasFile(gomock.Any(), key).Return(true, nil)

		h := storing.MakeHandler(st)
		server := httptest.NewServer(h)
		c, err := client.New(server.URL)
		require.NoError(t, err)

		ok, err := c.HasFile(context.Background(), key)
		require.NoError(t, err)
		assert.True(t, ok)
	})
	t.Run("False", func(t *testing.T) {
		var (
			key  = "fileName"
			ctrl = gomock.NewController(t)
		)
		st := mock.NewStoring(ctrl)
		defer ctrl.Finish()

		st.EXPECT().HasFile(gomock.Any(), key).Return(false, nil)

		h := storing.MakeHandler(st)
		server := httptest.NewServer(h)
		c, err := client.New(server.URL)
		require.NoError(t, err)

		ok, err := c.HasFile(context.Background(), key)
		require.NoError(t, err)
		assert.False(t, ok)
	})
}

func TestGetConfig(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		st := mock.NewStoring(ctrl)
		defer ctrl.Finish()
		ecfg := &config.Config{MemberlistName: "Pepito"}

		st.EXPECT().Config(gomock.Any()).Return(ecfg, nil)

		h := storing.MakeHandler(st)
		server := httptest.NewServer(h)
		c, err := client.New(server.URL)
		require.NoError(t, err)

		cfg, err := c.Config(context.Background())
		require.NoError(t, err)
		assert.Equal(t, ecfg, cfg)
	})
}

type ClosingBuffer struct {
	*bytes.Buffer
}

func (cb ClosingBuffer) Close() (err error) {
	//we don't actually have to do anything here, since the buffer is
	//just some data in memory
	//and the error is initialized to no-error
	return nil
}
