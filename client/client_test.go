package client_test

import (
	"bytes"
	"context"
	"errors"
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

func TestCreateFile(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			ctrl        = gomock.NewController(t)
			st          = mock.NewStoring(ctrl)
			content     = make([]byte, 6000)
			iorcContent = io.NopCloser(bytes.NewBuffer(content))
			key         = "filename"
			rep         = 10
		)
		defer ctrl.Finish()

		st.EXPECT().CreateFile(gomock.Any(), key, gomock.Any(), rep).Do(func(_ context.Context, _ string, b io.ReadCloser, _ int) {
			c, err := io.ReadAll(b)
			require.NoError(t, err)
			assert.Equal(t, content, c)
		}).Return(nil)

		h := storing.MakeHandler(st)
		server := httptest.NewServer(h)
		c, err := client.New(server.URL)
		require.NoError(t, err)

		err = c.CreateFile(context.Background(), key, iorcContent, rep)
		require.NoError(t, err)
	})
	t.Run("Error", func(t *testing.T) {
		var (
			ctrl        = gomock.NewController(t)
			st          = mock.NewStoring(ctrl)
			content     = make([]byte, 6000)
			iorcContent = io.NopCloser(bytes.NewBuffer(content))
			key         = "filename"
			rep         = 10
		)
		defer ctrl.Finish()

		st.EXPECT().CreateFile(gomock.Any(), key, gomock.Any(), rep).Do(func(_ context.Context, _ string, b io.ReadCloser, _ int) {
			c, err := io.ReadAll(b)
			require.NoError(t, err)
			assert.Equal(t, content, c)
		}).Return(errors.New("some error"))

		h := storing.MakeHandler(st)
		server := httptest.NewServer(h)
		c, err := client.New(server.URL)
		require.NoError(t, err)

		err = c.CreateFile(context.Background(), key, iorcContent, rep)
		assert.EqualError(t, err, "some error")
	})
}

func TestGetFile(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			key = "fileName"
			// Kind of a big content just to test
			content = make([]byte, 6000)
			ctrl    = gomock.NewController(t)
		)
		st := mock.NewStoring(ctrl)
		defer ctrl.Finish()

		st.EXPECT().GetFile(gomock.Any(), key).Return(io.NopCloser(bytes.NewBuffer(content)), nil)

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
	})
	t.Run("Error", func(t *testing.T) {
		var (
			key  = "fileName"
			ctrl = gomock.NewController(t)
		)
		st := mock.NewStoring(ctrl)
		defer ctrl.Finish()

		st.EXPECT().GetFile(gomock.Any(), key).Return(nil, errors.New("some error"))

		h := storing.MakeHandler(st)
		server := httptest.NewServer(h)
		c, err := client.New(server.URL)
		require.NoError(t, err)

		ior, err := c.GetFile(context.Background(), key)
		require.Nil(t, ior)
		assert.EqualError(t, err, "some error")
	})
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
	t.Run("Error", func(t *testing.T) {
		var (
			key  = "fileName"
			ctrl = gomock.NewController(t)
		)
		st := mock.NewStoring(ctrl)
		defer ctrl.Finish()

		st.EXPECT().HasFile(gomock.Any(), key).Return(false, errors.New("some error"))

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
		ecfg := &config.Config{Name: "Pepito"}

		st.EXPECT().Config(gomock.Any()).Return(ecfg, nil)

		h := storing.MakeHandler(st)
		server := httptest.NewServer(h)
		c, err := client.New(server.URL)
		require.NoError(t, err)

		cfg, err := c.Config(context.Background())
		require.NoError(t, err)
		assert.Equal(t, ecfg, cfg)
	})
	t.Run("Error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		st := mock.NewStoring(ctrl)
		defer ctrl.Finish()

		st.EXPECT().Config(gomock.Any()).Return(nil, errors.New("some error"))

		h := storing.MakeHandler(st)
		server := httptest.NewServer(h)
		c, err := client.New(server.URL)
		require.NoError(t, err)

		cfg, err := c.Config(context.Background())
		require.Nil(t, cfg)
		assert.EqualError(t, err, "some error")
	})
}

func TestDeleteFile(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		st := mock.NewStoring(ctrl)
		key := "filename"
		defer ctrl.Finish()

		st.EXPECT().DeleteFile(gomock.Any(), key).Return(nil)

		h := storing.MakeHandler(st)
		server := httptest.NewServer(h)
		c, err := client.New(server.URL)
		require.NoError(t, err)

		err = c.DeleteFile(context.Background(), key)
		require.NoError(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		st := mock.NewStoring(ctrl)
		key := "filename"
		defer ctrl.Finish()

		st.EXPECT().DeleteFile(gomock.Any(), key).Return(errors.New("some error"))

		h := storing.MakeHandler(st)
		server := httptest.NewServer(h)
		c, err := client.New(server.URL)
		require.NoError(t, err)

		err = c.DeleteFile(context.Background(), key)
		assert.EqualError(t, err, "some error")
	})
}

func TestCreateReplica(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			ctrl        = gomock.NewController(t)
			st          = mock.NewStoring(ctrl)
			content     = make([]byte, 6000)
			iorcContent = io.NopCloser(bytes.NewBuffer(content))
			key         = "filename"
			volID       = "volID"
		)
		defer ctrl.Finish()

		st.EXPECT().CreateReplica(gomock.Any(), key, gomock.Any()).Do(func(_ context.Context, _ string, b io.ReadCloser) {
			c, err := io.ReadAll(b)
			require.NoError(t, err)
			assert.Equal(t, content, c)
		}).Return(volID, nil)

		h := storing.MakeHandler(st)
		server := httptest.NewServer(h)
		c, err := client.New(server.URL)
		require.NoError(t, err)

		vID, err := c.CreateReplica(context.Background(), key, iorcContent)
		require.NoError(t, err)
		assert.Equal(t, volID, vID)
	})
	t.Run("Error", func(t *testing.T) {
		var (
			ctrl        = gomock.NewController(t)
			st          = mock.NewStoring(ctrl)
			content     = make([]byte, 6000)
			iorcContent = io.NopCloser(bytes.NewBuffer(content))
			key         = "filename"
		)
		defer ctrl.Finish()

		st.EXPECT().CreateReplica(gomock.Any(), key, gomock.Any()).Do(func(_ context.Context, _ string, b io.ReadCloser) {
			c, err := io.ReadAll(b)
			require.NoError(t, err)
			assert.Equal(t, content, c)
		}).Return("", errors.New("some-error"))

		h := storing.MakeHandler(st)
		server := httptest.NewServer(h)
		c, err := client.New(server.URL)
		require.NoError(t, err)

		vID, err := c.CreateReplica(context.Background(), key, iorcContent)
		assert.EqualError(t, err, "some-error")
		assert.Equal(t, "", vID)
	})
}

func TestUpdateFileReplica(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			ctrl = gomock.NewController(t)
			st   = mock.NewStoring(ctrl)
			key  = "filename"
			vids = []string{"volID", "volID2"}
			rep  = 2
		)
		defer ctrl.Finish()

		st.EXPECT().UpdateFileReplica(gomock.Any(), key, vids, rep).Return(nil)

		h := storing.MakeHandler(st)
		server := httptest.NewServer(h)
		c, err := client.New(server.URL)
		require.NoError(t, err)

		err = c.UpdateFileReplica(context.Background(), key, vids, rep)
		require.NoError(t, err)
	})
	t.Run("Error", func(t *testing.T) {
		var (
			ctrl = gomock.NewController(t)
			st   = mock.NewStoring(ctrl)
			key  = "filename"
			vids = []string{"volID", "volID2"}
			rep  = 2
		)
		defer ctrl.Finish()

		st.EXPECT().UpdateFileReplica(gomock.Any(), key, vids, rep).Return(errors.New("some-error"))

		h := storing.MakeHandler(st)
		server := httptest.NewServer(h)
		c, err := client.New(server.URL)
		require.NoError(t, err)

		err = c.UpdateFileReplica(context.Background(), key, vids, rep)
		assert.EqualError(t, err, "some-error")
	})
}
