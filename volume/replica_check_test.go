package volume

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/rebost/file"
	"github.com/xescugc/rebost/replica"
	"github.com/xescugc/rebost/uow"
)

// firstReplicaJob reads the first pending replica job from BoltDB (nil if none).
func firstReplicaJob(t *testing.T, lv *local) *replica.Replica {
	t.Helper()
	var rp *replica.Replica
	err := lv.startUnitOfWork(context.Background(), uow.Read, func(ctx context.Context, uw uow.UnitOfWork) error {
		var e error
		rp, e = uw.Replicas().First(ctx)
		if e != nil && e.Error() == "not found" {
			return nil
		}
		return e
	})
	require.NoError(t, err)
	return rp
}

// registerReplicaJob inserts a replica job directly into BoltDB.
func registerReplicaJob(t *testing.T, lv *local, rp *replica.Replica) {
	t.Helper()
	err := lv.startUnitOfWork(context.Background(), uow.Write, func(ctx context.Context, uw uow.UnitOfWork) error {
		return uw.Replicas().Create(ctx, rp)
	})
	require.NoError(t, err)
}

func TestRunReplicaCheck(t *testing.T) {
	t.Run("OwnedUnderReplicated", func(t *testing.T) {
		lv, cleanup := newScrubTestVolume(t)
		defer cleanup()

		content := []byte("under-replicated content")
		sig := computeSHA1(content)
		writeFileAtSig(t, lv, sig, content)
		registerFile(t, lv, &file.File{
			Keys:      []string{"bucket/under"},
			Signature: sig,
			VolumeIDs: []string{lv.id}, // only one copy, needs 3
			Replica:   3,
		})

		lv.runReplicaCheck()

		rp := firstReplicaJob(t, lv)
		require.NotNil(t, rp, "under-replicated file should produce a replica job")
		assert.Equal(t, "bucket/under", rp.Key)
		assert.Equal(t, 2, rp.Count) // 3 - 1
		assert.Equal(t, 3, rp.OriginalCount)
		assert.Equal(t, sig, rp.Signature)
		assert.Equal(t, lv.id, rp.VolumeID)
	})

	t.Run("OwnedFullyReplicated", func(t *testing.T) {
		lv, cleanup := newScrubTestVolume(t)
		defer cleanup()

		content := []byte("fully replicated content")
		sig := computeSHA1(content)
		writeFileAtSig(t, lv, sig, content)
		registerFile(t, lv, &file.File{
			Keys:      []string{"bucket/full"},
			Signature: sig,
			VolumeIDs: []string{lv.id, "remote-1", "remote-2"},
			Replica:   3,
		})

		lv.runReplicaCheck()

		assert.Nil(t, firstReplicaJob(t, lv), "fully replicated file should not produce a replica job")
	})

	t.Run("NonOwned", func(t *testing.T) {
		lv, cleanup := newScrubTestVolume(t)
		defer cleanup()

		content := []byte("non-owned content")
		sig := computeSHA1(content)
		writeFileAtSig(t, lv, sig, content)
		registerFile(t, lv, &file.File{
			Keys:      []string{"bucket/nonowned"},
			Signature: sig,
			VolumeIDs: []string{"other-volume", lv.id}, // other volume is owner
			Replica:   3,
		})

		lv.runReplicaCheck()

		assert.Nil(t, firstReplicaJob(t, lv), "non-owned file should not produce a replica job")
	})

	t.Run("AlreadyQueued", func(t *testing.T) {
		lv, cleanup := newScrubTestVolume(t)
		defer cleanup()

		content := []byte("already queued content")
		sig := computeSHA1(content)
		writeFileAtSig(t, lv, sig, content)
		registerFile(t, lv, &file.File{
			Keys:      []string{"bucket/queued"},
			Signature: sig,
			VolumeIDs: []string{lv.id},
			Replica:   3,
		})

		// Pre-insert a replica job for this key.
		registerReplicaJob(t, lv, &replica.Replica{
			ID:            "existing-job",
			Key:           "bucket/queued",
			Count:         2,
			OriginalCount: 3,
			VolumeID:      lv.id,
		})

		lv.runReplicaCheck()

		// Still only one job.
		rp := firstReplicaJob(t, lv)
		require.NotNil(t, rp)
		assert.Equal(t, "existing-job", rp.ID, "should not create a second job when one already exists")
	})
}
