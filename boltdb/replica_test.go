package boltdb_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/rebost/boltdb"
	"github.com/xescugc/rebost/replica"
	"github.com/xescugc/rebost/uow"
)

func TestReplicaRepository_HasAny(t *testing.T) {
	ctx := context.Background()

	db, cleanup := newTestDB(t)
	defer cleanup()

	startUOW, err := boltdb.NewUOW(db)
	require.NoError(t, err)

	// Empty queue returns false
	var hasAny bool
	err = startUOW(ctx, uow.Read, func(ctx context.Context, uw uow.UnitOfWork) error {
		var err error
		hasAny, err = uw.Replicas().HasAny(ctx)
		return err
	})
	require.NoError(t, err)
	assert.False(t, hasAny, "empty queue should return false")

	// After Create returns true
	rp := &replica.Replica{
		ID:        "replica-1",
		Key:       "bucket/file.txt",
		Signature: "abc123",
		Count:     2,
	}
	err = startUOW(ctx, uow.Write, func(ctx context.Context, uw uow.UnitOfWork) error {
		return uw.Replicas().Create(ctx, rp)
	})
	require.NoError(t, err)

	err = startUOW(ctx, uow.Read, func(ctx context.Context, uw uow.UnitOfWork) error {
		var err error
		hasAny, err = uw.Replicas().HasAny(ctx)
		return err
	})
	require.NoError(t, err)
	assert.True(t, hasAny, "queue with one item should return true")

	// After Delete returns false
	err = startUOW(ctx, uow.Write, func(ctx context.Context, uw uow.UnitOfWork) error {
		return uw.Replicas().Delete(ctx, rp)
	})
	require.NoError(t, err)

	err = startUOW(ctx, uow.Read, func(ctx context.Context, uw uow.UnitOfWork) error {
		var err error
		hasAny, err = uw.Replicas().HasAny(ctx)
		return err
	})
	require.NoError(t, err)
	assert.False(t, hasAny, "empty queue after delete should return false")
}
