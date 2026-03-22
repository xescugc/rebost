package boltdb_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/rebost/boltdb"
	"github.com/xescugc/rebost/file"
	"github.com/xescugc/rebost/uow"
	bolt "go.etcd.io/bbolt"
)

func newTestDB(t *testing.T) (*bolt.DB, func()) {
	t.Helper()
	f, err := os.CreateTemp("", "rebost-boltdb-test-*.db")
	require.NoError(t, err)
	f.Close()

	db, err := bolt.Open(f.Name(), 0600, nil)
	require.NoError(t, err)

	return db, func() {
		db.Close()
		os.Remove(f.Name())
	}
}

func TestFileRepository_All(t *testing.T) {
	ctx := context.Background()

	db, cleanup := newTestDB(t)
	defer cleanup()

	startUOW, err := boltdb.NewUOW(db)
	require.NoError(t, err)

	// Store multiple files
	f1 := &file.File{Signature: "aaa111", Keys: []string{"key1"}}
	f2 := &file.File{Signature: "bbb222", Keys: []string{"key2"}}
	f3 := &file.File{Signature: "ccc333", Keys: []string{"key3"}}

	err = startUOW(ctx, uow.Write, func(ctx context.Context, uw uow.UnitOfWork) error {
		if err := uw.Files().CreateOrReplace(ctx, f1); err != nil {
			return err
		}
		if err := uw.Files().CreateOrReplace(ctx, f2); err != nil {
			return err
		}
		return uw.Files().CreateOrReplace(ctx, f3)
	})
	require.NoError(t, err)

	// Call All and assert all files returned
	var files []*file.File
	err = startUOW(ctx, uow.Read, func(ctx context.Context, uw uow.UnitOfWork) error {
		var err error
		files, err = uw.Files().All(ctx)
		return err
	})
	require.NoError(t, err)

	assert.Len(t, files, 3)

	sigs := make([]string, 0, len(files))
	for _, f := range files {
		sigs = append(sigs, f.Signature)
	}
	assert.ElementsMatch(t, []string{"aaa111", "bbb222", "ccc333"}, sigs)
}

func TestFileRepository_All_Empty(t *testing.T) {
	ctx := context.Background()

	db, cleanup := newTestDB(t)
	defer cleanup()

	startUOW, err := boltdb.NewUOW(db)
	require.NoError(t, err)

	var files []*file.File
	err = startUOW(ctx, uow.Read, func(ctx context.Context, uw uow.UnitOfWork) error {
		var err error
		files, err = uw.Files().All(ctx)
		return err
	})
	require.NoError(t, err)
	assert.Empty(t, files)
}
