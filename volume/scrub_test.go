package volume

import (
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/rebost/boltdb"
	"github.com/xescugc/rebost/file"
	"github.com/xescugc/rebost/fs"
	"github.com/xescugc/rebost/scrub"
	"github.com/xescugc/rebost/uow"
	bolt "go.etcd.io/bbolt"
)

// newScrubTestVolume creates a real volume backed by an OS temp dir and BoltDB.
// The caller must call the returned cleanup function.
func newScrubTestVolume(t *testing.T) (*local, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "rebost-scrub-*")
	require.NoError(t, err)

	dbPath := path.Join(tmpDir, "my.db")
	db, err := bolt.Open(dbPath, 0600, nil)
	require.NoError(t, err)

	boltUow, err := boltdb.NewUOW(db)
	require.NoError(t, err)

	osfs := afero.NewOsFs()
	suow := fs.UOWWithFs(osfs, boltUow)

	v, err := New(tmpDir, osfs, slog.New(slog.NewTextHandler(io.Discard, nil)), suow, time.Hour, 0)
	require.NoError(t, err)

	lv := v.(*local)
	return lv, func() {
		v.Close()
		db.Close()
		os.RemoveAll(tmpDir)
	}
}

// computeSHA1 computes the hex SHA1 of data.
func computeSHA1(data []byte) string {
	h := sha1.New()
	h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// writeFileAtSig writes data to the filesystem path corresponding to sig.
func writeFileAtSig(t *testing.T, lv *local, sig string, data []byte) {
	t.Helper()
	p := file.Path(lv.fileDir, sig)
	dir, _ := path.Split(p)
	require.NoError(t, lv.fs.MkdirAll(dir, os.ModePerm))
	fh, err := lv.fs.Create(p)
	require.NoError(t, err)
	_, err = fh.Write(data)
	require.NoError(t, err)
	fh.Close()
}

// registerFile stores a file record in BoltDB.
func registerFile(t *testing.T, lv *local, f *file.File) {
	t.Helper()
	err := lv.startUnitOfWork(context.Background(), uow.Write, func(ctx context.Context, uw uow.UnitOfWork) error {
		return uw.Files().CreateOrReplace(ctx, f)
	})
	require.NoError(t, err)
}

// firstScrubJob reads the first pending scrub job from BoltDB (nil if none).
func firstScrubJob(t *testing.T, lv *local) *scrub.Scrub {
	t.Helper()
	var sc *scrub.Scrub
	err := lv.startUnitOfWork(context.Background(), uow.Read, func(ctx context.Context, uw uow.UnitOfWork) error {
		var e error
		sc, e = uw.Scrubs().First(ctx)
		if e != nil && e.Error() == "not found" {
			return nil
		}
		return e
	})
	require.NoError(t, err)
	return sc
}

func TestRunScrub_HealthyFile(t *testing.T) {
	lv, cleanup := newScrubTestVolume(t)
	defer cleanup()

	content := []byte("hello scrub world")
	sig := computeSHA1(content)

	writeFileAtSig(t, lv, sig, content)
	registerFile(t, lv, &file.File{
		Keys:      []string{"bucket/key"},
		Signature: sig,
		VolumeIDs: []string{lv.id},
	})

	lv.runScrub()

	assert.Nil(t, firstScrubJob(t, lv), "healthy file should not produce a scrub job")
}

func TestRunScrub_CorruptFile(t *testing.T) {
	lv, cleanup := newScrubTestVolume(t)
	defer cleanup()

	content := []byte("correct content")
	sig := computeSHA1(content)
	remoteVID := "remote-vol-1"

	// Write *wrong* bytes so the checksum will mismatch.
	writeFileAtSig(t, lv, sig, []byte("CORRUPTED!"))
	registerFile(t, lv, &file.File{
		Keys:      []string{"bucket/corrupt"},
		Signature: sig,
		VolumeIDs: []string{lv.id, remoteVID},
	})

	lv.runScrub()

	sc := firstScrubJob(t, lv)
	require.NotNil(t, sc, "corrupt file should produce a scrub job")
	assert.Equal(t, "bucket/corrupt", sc.Key)
	assert.Equal(t, sig, sc.Signature)
	// Local volume ID must be filtered out.
	assert.Equal(t, []string{remoteVID}, sc.VolumeIDs)
}

func TestRunScrub_CorruptFile_NoRemoteReplicas(t *testing.T) {
	lv, cleanup := newScrubTestVolume(t)
	defer cleanup()

	content := []byte("only local copy")
	sig := computeSHA1(content)

	writeFileAtSig(t, lv, sig, []byte("CORRUPTED!"))
	registerFile(t, lv, &file.File{
		Keys:      []string{"bucket/lonely"},
		Signature: sig,
		VolumeIDs: []string{lv.id}, // no remote replicas
	})

	lv.runScrub()

	sc := firstScrubJob(t, lv)
	require.NotNil(t, sc, "corrupt file with no remote replicas should still produce a scrub job")
	assert.Empty(t, sc.VolumeIDs, "VolumeIDs should be empty when only local copy exists")
}
