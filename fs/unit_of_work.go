package fs

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/afero"
	"github.com/xescugc/rebost/uow"
)

//go:generate mockgen -destination=../mock/fs.go -mock_names=Fs=Fs -package=mock github.com/spf13/afero Fs

// fsWrappedUoW embeds a UnitOfWork and overrides Fs() to return the tracker.
type fsWrappedUoW struct {
	uow.UnitOfWork
	tracker *uowTracker
}

func (w *fsWrappedUoW) Fs() afero.Fs { return w.tracker }

// UOWWithFs creates a Unit of Work wrapper that tracks filesystem operations
// (Create, Remove, Rename) and rolls them back or commits them depending on
// whether the inner UoW succeeds or fails.
func UOWWithFs(baseFs afero.Fs, suow uow.StartUnitOfWork) uow.StartUnitOfWork {
	return func(ctx context.Context, t uow.Type, uowFn uow.UnitOfWorkFn) error {
		tracker := newUOWTracker(baseFs)

		err := suow(ctx, t, func(ctx context.Context, uw uow.UnitOfWork) error {
			return uowFn(ctx, &fsWrappedUoW{UnitOfWork: uw, tracker: tracker})
		})

		if err != nil {
			for _, ra := range tracker.rollbackActions {
				if rerr := ra(tracker.fs); rerr != nil {
					// TODO: Do we stop execution or continue doing the Rollback actions?
					return rerr
				}
			}
		} else {
			for _, ra := range tracker.commitActions {
				if cerr := ra(tracker.fs); cerr != nil {
					// TODO: Do we stop execution or continue doing the Commit actions?
					return cerr
				}
			}
		}
		return err
	}
}

type actionFn func(afero.Fs) error

type uowTracker struct {
	fs afero.Fs

	rollbackActions []actionFn
	commitActions   []actionFn
}

func newUOWTracker(f afero.Fs) *uowTracker {
	return &uowTracker{
		fs: f,

		rollbackActions: make([]actionFn, 0),
		commitActions:   make([]actionFn, 0),
	}
}

func (uowt *uowTracker) Name() string { return uowt.fs.Name() }

func (uowt *uowTracker) Create(name string) (afero.File, error) {
	uowt.rollbackActions = append(uowt.rollbackActions, func(fs afero.Fs) error {
		return fs.Remove(name)
	})
	return uowt.fs.Create(name)
}

func (uowt *uowTracker) Mkdir(name string, perm os.FileMode) error {
	return uowt.fs.Mkdir(name, perm)
}

func (uowt *uowTracker) MkdirAll(path string, perm os.FileMode) error {
	return uowt.fs.MkdirAll(path, perm)
}

func (uowt *uowTracker) Open(name string) (afero.File, error) {
	return uowt.fs.Open(name)
}

func (uowt *uowTracker) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	return uowt.fs.OpenFile(name, flag, perm)
}

func (uowt *uowTracker) Remove(name string) error {
	tmp := fmt.Sprintf("%s.tmp", name)
	uowt.commitActions = append(uowt.rollbackActions, func(fs afero.Fs) error {
		return fs.Remove(tmp)
	})
	uowt.rollbackActions = append(uowt.rollbackActions, func(fs afero.Fs) error {
		return fs.Rename(tmp, name)
	})
	return uowt.fs.Rename(name, tmp)
}

func (uowt *uowTracker) RemoveAll(path string) error {
	return uowt.fs.RemoveAll(path)
}

func (uowt *uowTracker) Rename(oldname, newname string) error {
	uowt.rollbackActions = append(uowt.rollbackActions, func(fs afero.Fs) error {
		return fs.Rename(newname, oldname)
	})
	return uowt.fs.Rename(oldname, newname)
}

func (uowt *uowTracker) Stat(name string) (os.FileInfo, error) {
	return uowt.fs.Stat(name)
}

func (uowt *uowTracker) Chmod(name string, mode os.FileMode) error {
	return uowt.fs.Chmod(name, mode)
}

func (uowt *uowTracker) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return uowt.fs.Chtimes(name, atime, mtime)
}

func (uowt *uowTracker) Chown(name string, uid, gid int) error {
	return uowt.fs.Chown(name, uid, gid)
}
