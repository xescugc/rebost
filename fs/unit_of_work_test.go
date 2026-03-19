package fs_test

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/xescugc/rebost/fs"
	"github.com/xescugc/rebost/mock"
	"github.com/xescugc/rebost/uow"
)

func TestCreate(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mfs, suow, finishFn := newSuow(t)
		defer finishFn()
		ctx := context.Background()

		mfs.EXPECT().Create("test/path").Return(nil, nil)

		fs.UOWWithFs(mfs, suow)(ctx, uow.Read, func(ctx context.Context, uw uow.UnitOfWork) error {
			uw.Fs().Create("test/path")
			return nil
		})
	})
	t.Run("Error", func(t *testing.T) {
		mfs, suow, finishFn := newSuow(t)
		defer finishFn()
		ctx := context.Background()

		fsc := mfs.EXPECT().Create("test/path").Return(nil, nil)
		mfs.EXPECT().Remove("test/path").Return(nil).After(fsc)

		fs.UOWWithFs(mfs, suow)(ctx, uow.Read, func(ctx context.Context, uw uow.UnitOfWork) error {
			uw.Fs().Create("test/path")
			return errors.New("some error")
		})
	})
}

func TestRemove(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mfs, suow, finishFn := newSuow(t)
		defer finishFn()
		ctx := context.Background()

		fsre := mfs.EXPECT().Rename("test/path", "test/path.tmp").Return(nil)
		mfs.EXPECT().Remove("test/path.tmp").Return(nil).After(fsre)

		fs.UOWWithFs(mfs, suow)(ctx, uow.Read, func(ctx context.Context, uw uow.UnitOfWork) error {
			uw.Fs().Remove("test/path")
			return nil
		})
	})
	t.Run("Error", func(t *testing.T) {
		mfs, suow, finishFn := newSuow(t)
		defer finishFn()
		ctx := context.Background()

		fsre := mfs.EXPECT().Rename("test/path", "test/path.tmp").Return(nil)
		mfs.EXPECT().Rename("test/path.tmp", "test/path").Return(nil).After(fsre)

		fs.UOWWithFs(mfs, suow)(ctx, uow.Read, func(ctx context.Context, uw uow.UnitOfWork) error {
			uw.Fs().Remove("test/path")
			return errors.New("some error")
		})
	})
}

func TestRename(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mfs, suow, finishFn := newSuow(t)
		defer finishFn()
		ctx := context.Background()

		mfs.EXPECT().Rename("test/path", "test/pathtest").Return(nil)

		fs.UOWWithFs(mfs, suow)(ctx, uow.Read, func(ctx context.Context, uw uow.UnitOfWork) error {
			uw.Fs().Rename("test/path", "test/pathtest")
			return nil
		})
	})
	t.Run("Error", func(t *testing.T) {
		mfs, suow, finishFn := newSuow(t)
		defer finishFn()
		ctx := context.Background()

		fsre := mfs.EXPECT().Rename("test/path", "test/pathtest").Return(nil)
		mfs.EXPECT().Rename("test/pathtest", "test/path").Return(nil).After(fsre)

		fs.UOWWithFs(mfs, suow)(ctx, uow.Read, func(ctx context.Context, uw uow.UnitOfWork) error {
			uw.Fs().Rename("test/path", "test/pathtest")
			return errors.New("some error")
		})
	})
}

func newSuow(t *testing.T) (*mock.Fs, uow.StartUnitOfWork, func()) {
	ctrl := gomock.NewController(t)
	finishFn := func() {
		ctrl.Finish()
	}
	mfs := mock.NewFs(ctrl)
	suow := func(ctx context.Context, t uow.Type, uowFn uow.UnitOfWorkFn) error {
		uw := mock.NewUnitOfWork(ctrl)
		return uowFn(ctx, uw)
	}

	return mfs, suow, finishFn
}
