package volume

import (
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"path"

	uuid "github.com/satori/go.uuid"
	"github.com/spf13/afero"
	"github.com/xescugc/rebost/file"
	"github.com/xescugc/rebost/idxkey"
	"github.com/xescugc/rebost/uow"
)

//go:generate mockgen -destination=../mock/volume.go -mock_names=Volume=Volume -package=mock github.com/xescugc/rebost/volume Volume

// Volume is an interface to deal with the simples actions
// and basic ones
type Volume interface {
	// CreateFile creates a new file from the reader with the key, there are
	// 4 different use cases to consider:
	// * New key and reader
	// * New key with already known reader
	// * Already known key with new reader
	// * Already known key and reader
	CreateFile(ctx context.Context, key string, reader io.ReadCloser) error

	// GetFile search for the file with the key
	GetFile(ctx context.Context, key string) (io.ReadCloser, error)

	// HasFile checks if a file with the key exists
	HasFile(ctx context.Context, key string) (bool, error)

	// DeleteFile deletes the key, if the key points to a
	// file with 2 keys, then just the key will be deleted
	// and not the content
	DeleteFile(ctx context.Context, key string) error
}

// Local is the definition of a Local volume which
// is an extension of the volume.Volume
type Local interface {
	Volume

	// ID returns the ID of the Volume
	ID() string
}

type local struct {
	fileDir string
	tempDir string
	id      string

	fs      afero.Fs
	files   file.Repository
	idxkeys idxkey.Repository

	startUnitOfWork uow.StartUnitOfWork
}

// New returns an implementation of the volume.Local interface using the provided parameters
// it can return an error because when initialized it also creates the needed directories
// if they are missing which are $root/file and $root/tmps and also the ID
func New(root string, files file.Repository, idxkeys idxkey.Repository, fileSystem afero.Fs, suow uow.StartUnitOfWork) (Local, error) {
	l := &local{
		fileDir: path.Join(root, "file"),
		tempDir: path.Join(root, "tmps"),

		files:   files,
		fs:      fileSystem,
		idxkeys: idxkeys,

		startUnitOfWork: suow,
	}

	err := l.fs.MkdirAll(l.fileDir, os.ModePerm)
	if err != nil {
		return nil, err
	}

	err = l.fs.MkdirAll(l.tempDir, os.ModePerm)
	if err != nil {
		return nil, err
	}

	var id string
	idPath := path.Join(root, "id")
	// Creates or reads the id from the idPath as a Volume
	// must have always the same ID
	if _, err = l.fs.Stat(idPath); os.IsNotExist(err) {
		id = uuid.NewV4().String()
		fh, err := l.fs.Create(idPath)
		if err != nil {
			return nil, err
		}
		defer fh.Close()

		_, err = io.WriteString(fh, id)
		if err != nil {
			return nil, err
		}
	} else {
		fh, err := l.fs.Open(idPath)
		if err != nil {
			return nil, err
		}
		defer fh.Close()

		// This 36 is the length is the length of
		// a UUID string: https://github.com/satori/go.uuid/blob/master/uuid.go#L116
		bid := make([]byte, 36)
		_, err = io.ReadFull(fh, bid)
		if err != nil {
			return nil, err
		}
		id = string(bid)
	}

	l.id = id

	return l, nil
}

func (l *local) ID() string { return l.id }

func (l *local) CreateFile(ctx context.Context, key string, r io.ReadCloser) error {
	tmp := path.Join(l.tempDir, uuid.NewV4().String())

	fh, err := l.fs.Create(tmp)
	if err != nil {
		return err
	}
	defer fh.Close()

	sh1 := sha1.New()
	w := io.MultiWriter(fh, sh1)
	io.Copy(w, r)
	r.Close()

	f := &file.File{
		Keys:      []string{key},
		Signature: fmt.Sprintf("%x", sh1.Sum(nil)),
	}

	p := f.Path(l.fileDir)
	dir, _ := path.Split(p)

	err = l.fs.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return err
	}

	err = l.fs.Rename(tmp, p)
	if err != nil {
		return err
	}

	err = l.startUnitOfWork(ctx, uow.Write, func(uw uow.UnitOfWork) error {
		dbf, err := uw.Files().FindBySignature(ctx, f.Signature)
		if err != nil && err.Error() != "not found" {
			return err
		}

		if dbf != nil {
			ok := false
			for _, k := range dbf.Keys {
				if k == key {
					ok = true
				}
			}
			if ok {
				return nil
			}
			dbf.Keys = append(dbf.Keys, key)
			f = dbf
		}

		err = uw.Files().CreateOrReplace(ctx, f)
		if err != nil {
			return err
		}

		ik, err := uw.IDXKeys().FindByKey(ctx, key)
		if err != nil && err.Error() != "not found" {
			return err
		}

		if ik != nil {
			dbf, err := uw.Files().FindBySignature(ctx, ik.Value)
			if err != nil && err.Error() != "not found" {
				return err
			}
			newKeys := make([]string, 0, len(dbf.Keys)-1)
			for _, k := range dbf.Keys {
				if k == key {
					continue
				}
				newKeys = append(newKeys, k)
			}
			if len(newKeys) == 0 {
				err = uw.Files().DeleteBySignature(ctx, ik.Value)
				if err != nil {
					return err
				}

				err = uw.Fs().Remove(file.Path(l.fileDir, ik.Value))
				if err != nil {
					return err
				}

				err = uw.IDXKeys().DeleteByKey(ctx, key)
				if err != nil {
					return err
				}
			} else {
				dbf.Keys = newKeys

				err = uw.Files().CreateOrReplace(ctx, dbf)
				if err != nil {
					return err
				}
			}
		}

		err = uw.IDXKeys().CreateOrReplace(ctx, idxkey.New(key, f.Signature))
		if err != nil && err.Error() != "not found" {
			return err
		}

		return nil
	}, l.idxkeys, l.files, l.fs)

	if err != nil {
		return err
	}

	return nil
}

func (l *local) GetFile(ctx context.Context, k string) (io.ReadCloser, error) {
	var (
		idk *idxkey.IDXKey
		err error
	)

	err = l.startUnitOfWork(ctx, uow.Read, func(uw uow.UnitOfWork) error {
		idk, err = uw.IDXKeys().FindByKey(ctx, k)
		if err != nil {
			return err
		}
		return nil
	}, l.idxkeys)

	if err != nil {
		return nil, err
	}

	fh, err := l.fs.Open(file.Path(l.fileDir, idk.Value))
	if err != nil {
		return nil, err
	}

	return fh, nil
}

func (l *local) DeleteFile(ctx context.Context, key string) error {
	return l.startUnitOfWork(ctx, uow.Read, func(uw uow.UnitOfWork) error {
		ik, err := uw.IDXKeys().FindByKey(ctx, key)
		if err != nil {
			return err
		}
		dbf, err := uw.Files().FindBySignature(ctx, ik.Value)
		if err != nil && err.Error() != "not found" {
			return err
		}
		newKeys := make([]string, 0, len(dbf.Keys)-1)
		for _, k := range dbf.Keys {
			if k == key {
				continue
			}
			newKeys = append(newKeys, k)
		}
		if len(newKeys) == 0 {
			err = uw.Files().DeleteBySignature(ctx, ik.Value)
			if err != nil {
				return err
			}

			err = uw.Fs().Remove(file.Path(l.fileDir, ik.Value))
			if err != nil {
				return err
			}
		} else {
			dbf.Keys = newKeys

			err = uw.Files().CreateOrReplace(ctx, dbf)
			if err != nil {
				return err
			}
		}

		return uw.IDXKeys().DeleteByKey(ctx, key)
	}, l.idxkeys, l.files, l.fs)
}

func (l *local) HasFile(ctx context.Context, k string) (bool, error) {
	err := l.startUnitOfWork(ctx, uow.Read, func(uw uow.UnitOfWork) error {
		_, err := uw.IDXKeys().FindByKey(ctx, k)
		if err != nil {
			return err
		}
		return nil
	}, l.idxkeys)

	if err != nil {
		if err.Error() == "not found" {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
