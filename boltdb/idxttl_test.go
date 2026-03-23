package boltdb_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/rebost/boltdb"
	"github.com/xescugc/rebost/idxttl"
	"github.com/xescugc/rebost/uow"
)

func TestIDXTTLRepository_FindRoundTrip(t *testing.T) {
	ctx := context.Background()

	db, cleanup := newTestDB(t)
	defer cleanup()

	startUOW, err := boltdb.NewUOW(db)
	require.NoError(t, err)

	// Use a fixed time truncated to seconds (RFC3339 has second precision)
	expiresAt := time.Now().Add(time.Hour).Truncate(time.Second).UTC()
	ittl := idxttl.New(expiresAt, "sig1", "sig2")

	err = startUOW(ctx, uow.Write, func(ctx context.Context, uw uow.UnitOfWork) error {
		return uw.IDXTTLs().CreateOrReplace(ctx, ittl)
	})
	require.NoError(t, err)

	var got *idxttl.IDXTTL
	err = startUOW(ctx, uow.Read, func(ctx context.Context, uw uow.UnitOfWork) error {
		var err error
		got, err = uw.IDXTTLs().Find(ctx, expiresAt)
		return err
	})
	require.NoError(t, err)

	assert.True(t, expiresAt.Equal(got.ExpiresAt), "ExpiresAt mismatch: got %v, want %v", got.ExpiresAt, expiresAt)
	assert.ElementsMatch(t, []string{"sig1", "sig2"}, got.Signatures)
}
