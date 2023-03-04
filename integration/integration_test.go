package integration_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/rebost/client"
)

const (
	firstNode = ""
	noReplica = 0
)

func init() {
	log.SetFlags(log.Llongfile)
}

type cancelFn func()

// With this we test the basic CRUD actions so we can assume that they pass and the other tests
// we can concentrate on the more edge case/use cases
func TestCRUD(t *testing.T) {
	var (
		keytxt     = "keytxt"
		txtcontent = []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.")
		iorctxt    = io.NopCloser(bytes.NewBuffer(txtcontent))

		keyimg = "keyimg"
		ctx    = context.Background()
	)

	imgcontent, err := os.ReadFile("./testdata/gopher.png")
	require.NoError(t, err)
	iorcimg := io.NopCloser(bytes.NewBuffer(imgcontent))

	cl1, u1, vid1, ca1 := newClient(t, "n1", firstNode)
	defer ca1()
	cl2, _, _, ca2 := newClient(t, "n2", u1)
	defer ca2()
	cl3, _, vid3, ca3 := newClient(t, "n3", u1)
	defer ca3()

	// Sleep one second to let the nodes communicate between each other
	// and have the cluster stable
	time.Sleep(time.Second)

	clients := []*client.Client{cl1, cl2, cl3}

	t.Run("CreateFile", func(t *testing.T) {

		err = cl1.CreateFile(ctx, keytxt, iorctxt, noReplica)
		require.NoError(t, err)

		err = cl3.CreateFile(ctx, keyimg, iorcimg, noReplica)
		require.NoError(t, err)

	})

	t.Run("HasFile", func(t *testing.T) {
		vid, ok, err := cl1.HasFile(ctx, keytxt)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, vid1, vid)

		vid, ok, err = cl1.HasFile(ctx, keyimg)
		require.NoError(t, err)
		assert.False(t, ok)
		assert.Equal(t, "", vid)

		vid, ok, err = cl2.HasFile(ctx, keytxt)
		require.NoError(t, err)
		assert.False(t, ok)
		assert.Equal(t, "", vid)

		vid, ok, err = cl2.HasFile(ctx, keyimg)
		require.NoError(t, err)
		assert.False(t, ok)
		assert.Equal(t, "", vid)

		vid, ok, err = cl3.HasFile(ctx, keytxt)
		require.NoError(t, err)
		assert.False(t, ok)
		assert.Equal(t, "", vid)

		vid, ok, err = cl3.HasFile(ctx, keyimg)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, vid3, vid)
	})

	t.Run("GetFile", func(t *testing.T) {
		for i, c := range clients {
			t.Run(fmt.Sprintf("From node %d", i+1), func(t *testing.T) {
				txtiorc, err := c.GetFile(ctx, keytxt)
				require.NoError(t, err)
				txtb, err := io.ReadAll(txtiorc)
				require.NoError(t, err)
				txtiorc.Close()

				assert.Equal(t, txtcontent, txtb)

				imgiorc, err := c.GetFile(ctx, keyimg)
				require.NoError(t, err)
				imgb, err := io.ReadAll(imgiorc)
				require.NoError(t, err)
				imgiorc.Close()

				assert.Equal(t, imgcontent, imgb)
			})
		}
	})

	t.Run("DeleteFile", func(t *testing.T) {
		err := cl2.DeleteFile(ctx, keyimg)
		require.NoError(t, err)

		_, err = cl2.GetFile(ctx, keyimg)
		assert.EqualError(t, err, "not found")
		_, err = cl1.GetFile(ctx, keyimg)
		assert.EqualError(t, err, "not found")
		_, err = cl2.GetFile(ctx, keyimg)
		assert.EqualError(t, err, "not found")

		err = cl2.DeleteFile(ctx, keytxt)
		require.NoError(t, err)

		_, err = cl2.GetFile(ctx, keytxt)
		assert.EqualError(t, err, "not found")
		_, err = cl1.GetFile(ctx, keytxt)
		assert.EqualError(t, err, "not found")
		_, err = cl2.GetFile(ctx, keytxt)
		assert.EqualError(t, err, "not found")
	})
}

func TestReplica(t *testing.T) {
	var (
		keytxt     = "keytxt"
		txtcontent = []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.")
		iorctxt    = io.NopCloser(bytes.NewBuffer(txtcontent))

		ctx = context.Background()
	)

	cl1, u1, vid1, ca1 := newClient(t, "n1", firstNode)
	defer ca1()
	cl2, _, vid2, ca2 := newClient(t, "n2", u1)
	defer ca2()
	cl3, _, vid3, ca3 := newClient(t, "n3", u1)
	defer ca3()
	cl4, _, vid4, ca4 := newClient(t, "n4", u1)
	defer ca4()
	cl5, _, vid5, ca5 := newClient(t, "n5", u1)
	defer ca5()

	clients := []*client.Client{cl1, cl2, cl3, cl4, cl5}
	vids := []string{vid1, vid2, vid3, vid4, vid5}
	cancels := []cancelFn{ca1, ca2, ca3, ca4, ca5}

	// Sleep one second to let the nodes communicate between each other
	// and have the cluster stable
	time.Sleep(time.Second)

	t.Run("HasFileReplicatedOn3/5", func(t *testing.T) {
		err := cl1.CreateFile(ctx, keytxt, iorctxt, 3)
		require.NoError(t, err)

		// As the goroutine has a delay of 1s we may have to
		// w8 for it
		time.Sleep(2 * time.Second)

		// As it's a replica 3 so 3/5 have to have it
		var (
			okCount  int
			nokCount int
		)
		for i, c := range clients {
			vid, ok, err := c.HasFile(ctx, keytxt)
			require.NoError(t, err)
			if ok {
				okCount++
				assert.Equal(t, vids[i], vid)
			} else {
				nokCount++
				assert.Equal(t, "", vid)
			}
		}
		assert.Equal(t, 3, okCount)
		assert.Equal(t, 2, nokCount)
	})

	t.Run("ExpandReplicaAfterNodeDead", func(t *testing.T) {
		for i, c := range clients {
			// We skip the first one because we know it's the owner
			// and we do now want to kill it now
			if i == 0 {
				continue
			}
			vid, ok, err := c.HasFile(ctx, keytxt)
			require.NoError(t, err)
			if ok {
				cancels[i]()
				assert.Equal(t, vids[i], vid)
				vids = append(vids[0:i], vids[i+1:]...)
				clients = append(clients[0:i], clients[i+1:]...)
				cancels = append(cancels[0:i], cancels[i+1:]...)
				break
			}
		}

		time.Sleep(5 * time.Second)

		// As it's a replica 3 so 3/4 have to have it
		var (
			okCount  int
			nokCount int
		)
		for i, c := range clients {
			vid, ok, err := c.HasFile(ctx, keytxt)
			require.NoError(t, err)
			if ok {
				okCount++
				assert.Equal(t, vids[i], vid)
			} else {
				nokCount++
				assert.Equal(t, "", vid)
			}
		}
		assert.Equal(t, 3, okCount)
		assert.Equal(t, 1, nokCount)
	})

	t.Run("ExpandReplicaAfterMasterOfFileDead", func(t *testing.T) {
		cancels[0]()
		clients = clients[1:]
		cancels = cancels[1:]
		vids = vids[1:]

		time.Sleep(5 * time.Second)

		// As it's a replica 3 so 3/4 have to have it
		var (
			okCount  int
			nokCount int
		)
		for i, c := range clients {
			vid, ok, err := c.HasFile(ctx, keytxt)
			require.NoError(t, err)
			if ok {
				okCount++
				assert.Equal(t, vids[i], vid)
			} else {
				nokCount++
				assert.Equal(t, "", vid)
			}
		}
		assert.Equal(t, 3, okCount)
		assert.Equal(t, 0, nokCount)
	})

}
