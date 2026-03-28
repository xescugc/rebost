package integration_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
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
	noTTL     = 0
)

var (
	noCA time.Time
)


type cancelFn func()

// With this we test the basic CRUD actions so we can assume that they pass and the other tests
// we can concentrate on the more edge case/use cases
func TestCRUD(t *testing.T) {
	var (
		keytxt     = "testbucket/keytxt"
		txtcontent = []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.")
		iorctxt    = io.NopCloser(bytes.NewBuffer(txtcontent))

		keyimg = "testbucket/keyimg"
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

		err = cl1.CreateFile(ctx, keytxt, iorctxt, noReplica, noTTL, noCA)
		require.NoError(t, err)

		err = cl3.CreateFile(ctx, keyimg, iorcimg, noReplica, noTTL, noCA)
		require.NoError(t, err)

	})

	t.Run("HasFile", func(t *testing.T) {
		// With HRW placement, files land on the deterministically highest-ranked
		// volume — not necessarily the node that initiated the write. Verify that
		// exactly one node holds each file locally.
		allClients := []*client.Client{cl1, cl2, cl3}
		allVids := []string{vid1, "", vid3} // vid2 was not captured

		keytxtCount, keyimgCount := 0, 0
		for i, c := range allClients {
			vid, ok, err := c.HasFile(ctx, keytxt)
			require.NoError(t, err)
			if ok {
				keytxtCount++
				if allVids[i] != "" {
					assert.Equal(t, allVids[i], vid)
				} else {
					assert.NotEmpty(t, vid)
				}
			} else {
				assert.Equal(t, "", vid)
			}

			vid, ok, err = c.HasFile(ctx, keyimg)
			require.NoError(t, err)
			if ok {
				keyimgCount++
				if allVids[i] != "" {
					assert.Equal(t, allVids[i], vid)
				} else {
					assert.NotEmpty(t, vid)
				}
			} else {
				assert.Equal(t, "", vid)
			}
		}

		assert.Equal(t, 1, keytxtCount, "exactly one node should hold keytxt locally")
		assert.Equal(t, 1, keyimgCount, "exactly one node should hold keyimg locally")
	})

	t.Run("GetFile", func(t *testing.T) {
		for i, c := range clients {
			t.Run(fmt.Sprintf("From node %d", i+1), func(t *testing.T) {
				txtiorc, _, err := c.GetFile(ctx, keytxt)
				require.NoError(t, err)
				txtb, err := io.ReadAll(txtiorc)
				require.NoError(t, err)
				txtiorc.Close()
				vid, _, err := c.HasFile(ctx, keytxt)
				require.NoError(t, err)
				assert.NotEmpty(t, vid)

				assert.Equal(t, txtcontent, txtb)

				imgiorc, _, err := c.GetFile(ctx, keyimg)
				require.NoError(t, err)
				imgb, err := io.ReadAll(imgiorc)
				require.NoError(t, err)
				imgiorc.Close()
				vid, _, err = c.HasFile(ctx, keyimg)
				require.NoError(t, err)
				assert.NotEmpty(t, vid)

				assert.Equal(t, imgcontent, imgb)
			})
		}
	})

	t.Run("DeleteFile", func(t *testing.T) {
		err := cl2.DeleteFile(ctx, keyimg)
		require.NoError(t, err)

		_, _, err = cl2.GetFile(ctx, keyimg)
		assert.EqualError(t, err, "not found")
		_, _, err = cl1.GetFile(ctx, keyimg)
		assert.EqualError(t, err, "not found")
		_, _, err = cl2.GetFile(ctx, keyimg)
		assert.EqualError(t, err, "not found")

		err = cl2.DeleteFile(ctx, keytxt)
		require.NoError(t, err)

		_, _, err = cl2.GetFile(ctx, keytxt)
		assert.EqualError(t, err, "not found")
		_, _, err = cl1.GetFile(ctx, keytxt)
		assert.EqualError(t, err, "not found")
		_, _, err = cl2.GetFile(ctx, keytxt)
		assert.EqualError(t, err, "not found")
	})
}

func TestReplica(t *testing.T) {
	var (
		keytxt     = "testbucket/keytxt"
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
		err := cl1.CreateFile(ctx, keytxt, iorctxt, 3, noTTL, noCA)
		require.NoError(t, err)

		// Poll until background replication has pushed the file to 3/5 nodes
		var okCount, nokCount int
		require.Eventually(t, func() bool {
			okCount, nokCount = 0, 0
			for _, c := range clients {
				_, ok, _ := c.HasFile(ctx, keytxt)
				if ok {
					okCount++
				} else {
					nokCount++
				}
			}
			return okCount == 3
		}, time.Minute, 100*time.Millisecond)

		// As it's a replica 3 so 3/5 have to have it
		for i, c := range clients {
			vid, ok, err := c.HasFile(ctx, keytxt)
			require.NoError(t, err)
			if ok {
				assert.Equal(t, vids[i], vid)
			} else {
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

		// Poll until re-replication restores the replica count to 3/4
		var okCount, nokCount int
		require.Eventually(t, func() bool {
			okCount, nokCount = 0, 0
			for _, c := range clients {
				_, ok, _ := c.HasFile(ctx, keytxt)
				if ok {
					okCount++
				} else {
					nokCount++
				}
			}
			return okCount == 3
		}, time.Minute, 100*time.Millisecond)

		// As it's a replica 3 so 3/4 have to have it
		for i, c := range clients {
			vid, ok, err := c.HasFile(ctx, keytxt)
			require.NoError(t, err)
			if ok {
				assert.Equal(t, vids[i], vid)
			} else {
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

		// Poll until re-replication ensures all 3 remaining nodes have the file
		var okCount, nokCount int
		require.Eventually(t, func() bool {
			okCount, nokCount = 0, 0
			for _, c := range clients {
				_, ok, _ := c.HasFile(ctx, keytxt)
				if ok {
					okCount++
				} else {
					nokCount++
				}
			}
			return okCount == 3
		}, time.Minute, 100*time.Millisecond)

		// As it's a replica 3 so 3/3 have to have it
		for i, c := range clients {
			vid, ok, err := c.HasFile(ctx, keytxt)
			require.NoError(t, err)
			if ok {
				assert.Equal(t, vids[i], vid)
			} else {
				assert.Equal(t, "", vid)
			}
		}
		assert.Equal(t, 3, okCount)
		assert.Equal(t, 0, nokCount)
	})
	t.Run("DeleteFile", func(t *testing.T) {
		clients[0].DeleteFile(ctx, keytxt)

		// Poll until deletion has propagated to all replica nodes
		var okCount, nokCount int
		require.Eventually(t, func() bool {
			okCount, nokCount = 0, 0
			for _, c := range clients {
				_, ok, _ := c.HasFile(ctx, keytxt)
				if ok {
					okCount++
				} else {
					nokCount++
				}
			}
			return okCount == 0
		}, time.Minute, 100*time.Millisecond)

		for i, c := range clients {
			vid, ok, err := c.HasFile(ctx, keytxt)
			require.NoError(t, err)
			if ok {
				assert.Equal(t, vids[i], vid)
			} else {
				assert.Equal(t, "", vid)
			}
		}
		assert.Equal(t, 0, okCount)
		assert.Equal(t, 3, nokCount)
	})
}

// TestHeadEndpointAsymmetry verifies the core invariant of issue #183:
// for a file stored only on node1, node2 must return different results from
// the two HEAD endpoints:
//   - HEAD /{bucket}/{key}       (StatFile, cluster-wide)  → found
//   - HEAD /local/{bucket}/{key} (HasFile, local-only)     → not found
func TestHeadEndpointAsymmetry(t *testing.T) {
	var (
		key        = "testbucket/remoteonly"
		txtcontent = []byte("asymmetry test content")
		ctx        = context.Background()
	)

	cl1, u1, _, ca1 := newClient(t, "n1", firstNode)
	defer ca1()
	cl2, _, _, ca2 := newClient(t, "n2", u1)
	defer ca2()

	// Let the cluster stabilise
	time.Sleep(time.Second)

	// Use CreateReplica to force local storage on node1, bypassing HRW routing which
	// could otherwise place the file on node2 and make the HasFile assertion below fail.
	_, err := cl1.CreateReplica(ctx, key, io.NopCloser(bytes.NewBuffer(txtcontent)), noTTL, noCA)
	require.NoError(t, err)

	// From node2: cluster-wide HEAD must find the file (on node1)
	stat, err := cl2.StatFile(ctx, key)
	require.NoError(t, err)
	assert.NotNil(t, stat, "StatFile should find the file cluster-wide")

	// From node2: local-only HEAD must NOT find the file
	_, ok, err := cl2.HasFile(ctx, key)
	require.NoError(t, err)
	assert.False(t, ok, "HasFile must return false — file is not local to node2")
}

func TestDrain(t *testing.T) {
	var (
		keytxt     = "testbucket/keytxt"
		txtcontent = []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.")
		iorctxt    = io.NopCloser(bytes.NewBuffer(txtcontent))

		ctx = context.Background()
	)

	// Start node1 using newNode so we also have the service for Drain.
	u1, _, svc1, _, ca1 := newNode(t, "n1", firstNode, nil)
	defer ca1()
	cl1, err := client.New(u1)
	require.NoError(t, err)

	// Start node2 with membership exposed so we can verify node1 left the cluster.
	u2, _, _, m2, ca2 := newNode(t, "n2", u1, nil)
	defer ca2()
	cl2, err := client.New(u2)
	require.NoError(t, err)

	cl3, _, _, ca3 := newClient(t, "n3", u1)
	defer ca3()
	cl4, _, _, ca4 := newClient(t, "n4", u1)
	defer ca4()

	// Sleep one second to let the nodes communicate between each other
	// and have the cluster stable
	time.Sleep(time.Second)

	// Upload the file with replica=3 to node1
	err = cl1.CreateFile(ctx, keytxt, iorctxt, 3, noTTL, noCA)
	require.NoError(t, err)

	// Wait for replication to reach exactly 2 of the 3 peer nodes (cl2/cl3/cl4).
	// cl1 is the uploader and always holds the file, so we exclude it to avoid
	// misidentifying cl1 as the "missing" node.
	// We record which peer does NOT have the file so we can assert that
	// drain causes replication specifically to that previously-empty node.
	allClients := []*client.Client{cl2, cl3, cl4}
	var missingNode *client.Client
	require.Eventually(t, func() bool {
		okCount := 0
		missingNode = nil
		for _, c := range allClients {
			_, ok, _ := c.HasFile(ctx, keytxt)
			if ok {
				okCount++
			} else {
				missingNode = c
			}
		}
		return okCount == 2
	}, time.Minute, 100*time.Millisecond, "file should be replicated to exactly 2 peer nodes before drain")
	require.NotNil(t, missingNode, "exactly one node should be missing the file before drain")

	// Record the current node count before draining node1.
	initialNodeCount := len(m2.Nodes())

	// Drain node1 — this should replicate any locally-held files to nodes that
	// don't yet have them, purge node1's local copies, and leave the cluster.
	err = svc1.Drain(ctx)
	require.NoError(t, err)

	// After drain, the previously-missing node must now hold the file.
	require.Eventually(t, func() bool {
		_, ok, _ := missingNode.HasFile(ctx, keytxt)
		return ok
	}, time.Minute, 100*time.Millisecond, "drain should have replicated to the previously-empty node")

	// After drain, all of node2/node3/node4 must hold the file
	require.Eventually(t, func() bool {
		_, ok2, _ := cl2.HasFile(ctx, keytxt)
		_, ok3, _ := cl3.HasFile(ctx, keytxt)
		_, ok4, _ := cl4.HasFile(ctx, keytxt)
		return ok2 && ok3 && ok4
	}, time.Minute, 100*time.Millisecond, "after drain all remaining nodes should have the file")

	// node1 must no longer hold the file locally
	_, ok1, err := cl1.HasFile(ctx, keytxt)
	require.NoError(t, err)
	assert.False(t, ok1, "node1 should not hold the file after drain")

	// node1 must have left the cluster — verified by a count-based check that
	// is robust against localhost vs 127.0.0.1 URL format differences.
	require.Eventually(t, func() bool {
		return len(m2.Nodes()) == initialNodeCount-1
	}, time.Minute, 100*time.Millisecond, "node1 should have left the cluster")
}

func TestTTL(t *testing.T) {
	var (
		keytxt     = "testbucket/keytxt"
		txtcontent = []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.")
		iorctxt    = io.NopCloser(bytes.NewBuffer(txtcontent))

		ctx = context.Background()
		ttl = 5 * time.Second
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

	defer func() {
		for _, c := range cancels {
			c()
		}
	}()

	// Sleep one second to let the nodes communicate between each other
	// and have the cluster stable
	time.Sleep(time.Second)

	t.Run("HasFileReplicatedOn3/5", func(t *testing.T) {
		err := cl1.CreateFile(ctx, keytxt, iorctxt, 3, ttl, noCA)
		require.NoError(t, err)

		// Poll until background replication has pushed the file to 3/5 nodes
		var okCount, nokCount int
		require.Eventually(t, func() bool {
			okCount, nokCount = 0, 0
			for _, c := range clients {
				_, ok, _ := c.HasFile(ctx, keytxt)
				if ok {
					okCount++
				} else {
					nokCount++
				}
			}
			return okCount == 3
		}, time.Minute, 100*time.Millisecond)

		// As it's a replica 3 so 3/5 have to have it
		for i, c := range clients {
			vid, ok, err := c.HasFile(ctx, keytxt)
			require.NoError(t, err)
			if ok {
				assert.Equal(t, vids[i], vid)
			} else {
				assert.Equal(t, "", vid)
			}
		}
		assert.Equal(t, 3, okCount)
		assert.Equal(t, 2, nokCount)
	})

	t.Run("After TTL", func(t *testing.T) {
		// Poll until TTL expiry has propagated the deletion to all 5 nodes
		var okCount, nokCount int
		require.Eventually(t, func() bool {
			okCount, nokCount = 0, 0
			for _, c := range clients {
				_, ok, _ := c.HasFile(ctx, keytxt)
				if ok {
					okCount++
				} else {
					nokCount++
				}
			}
			return nokCount == 5
		}, time.Minute, 100*time.Millisecond)

		for i, c := range clients {
			vid, ok, err := c.HasFile(ctx, keytxt)
			require.NoError(t, err)
			if ok {
				assert.Equal(t, vids[i], vid)
			} else {
				assert.Equal(t, "", vid)
			}
		}
		assert.Equal(t, 0, okCount)
		assert.Equal(t, 5, nokCount)
	})

}
