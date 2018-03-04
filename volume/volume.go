package volume

import (
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"path"

	uuid "github.com/satori/go.uuid"
	"github.com/xescugc/rebost/file"
	"github.com/xescugc/rebost/fs"
	"github.com/xescugc/rebost/idxkey"
	"github.com/xescugc/rebost/uow"
)

//go:generate mockgen -destination=../mock/volume.go -mock_names=Volume=Volume -package=mock github.com/xescugc/rebost/volume Volume

// Volume is an interface to deal with the simples actions
// and basic ones
type Volume interface {
	CreateFile(key string, reader io.Reader) (*file.File, error)

	GetFile(key string) (io.Reader, error)

	HasFile(key string) (bool, error)

	DeleteFile(key string) error
}

type volume struct {
	fileDir string
	tempDir string

	fs      fs.Fs
	files   file.Repository
	idxkeys idxkey.Repository

	startUnitOfWork uow.StartUnitOfWork
}

// New returns an implementation of the volume.Volume interface using the provided parameters
// it can return an error because when initialized it also creates the needed directories
// if they are missing which are $root/file and $root/tmps
func New(root string, files file.Repository, idxkeys idxkey.Repository, fileSystem fs.Fs, suow uow.StartUnitOfWork) (Volume, error) {
	v := &volume{
		fileDir: path.Join(root, "file"),
		tempDir: path.Join(root, "tmps"),

		files:   files,
		fs:      fileSystem,
		idxkeys: idxkeys,

		startUnitOfWork: suow,
	}

	err := v.fs.MkdirAll(v.fileDir, os.ModePerm)
	if err != nil {
		return nil, err
	}

	err = v.fs.MkdirAll(v.tempDir, os.ModePerm)
	if err != nil {
		return nil, err
	}

	return v, nil
}

func (v *volume) CreateFile(key string, r io.Reader) (*file.File, error) {
	tmp := path.Join(v.tempDir, uuid.NewV4().String())

	fh, err := v.fs.Create(tmp)
	if err != nil {
		return nil, err
	}
	defer fh.Close()

	sh1 := sha1.New()
	w := io.MultiWriter(fh, sh1)
	io.Copy(w, r)

	f := &file.File{
		Keys:      []string{key},
		Signature: fmt.Sprintf("%x", sh1.Sum(nil)),
	}

	p := f.Path(v.fileDir)
	dir, _ := path.Split(p)

	err = v.fs.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return nil, err
	}

	err = v.fs.Rename(tmp, p)
	if err != nil {
		return nil, err
	}

	err = v.startUnitOfWork(uow.Write, func(uw uow.UnitOfWork) error {
		dbf, err := uw.Files().FindBySignature(f.Signature)
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

		err = uw.Files().CreateOrReplace(f)
		if err != nil {
			return err
		}

		ik, err := uw.IDXKeys().FindByKey(key)
		if err != nil && err.Error() != "not found" {
			return err
		}

		if ik != nil {
			dbf, err := uw.Files().FindBySignature(ik.Value)
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
				err = uw.Files().DeleteBySignature(ik.Value)
				if err != nil {
					return err
				}

				err = v.fs.Remove(file.Path(v.fileDir, ik.Value))
				if err != nil {
					return err
				}

				err = uw.IDXKeys().DeleteByKey(key)
				if err != nil {
					return err
				}
			} else {
				dbf.Keys = newKeys

				err = uw.Files().CreateOrReplace(dbf)
				if err != nil {
					return err
				}
			}
		}

		err = uw.IDXKeys().CreateOrReplace(idxkey.New(key, f.Signature))
		if err != nil && err.Error() != "not found" {
			return err
		}

		return nil
	}, v.idxkeys, v.files)

	if err != nil {
		return nil, err
	}

	return f, nil
}

func (v *volume) GetFile(k string) (io.Reader, error) {
	var (
		idk *idxkey.IDXKey
		err error
	)

	err = v.startUnitOfWork(uow.Read, func(uw uow.UnitOfWork) error {
		idk, err = uw.IDXKeys().FindByKey(k)
		if err != nil {
			return err
		}
		return nil
	}, v.idxkeys)

	if err != nil {
		return nil, err
	}

	pr, pw := io.Pipe()

	fh, err := v.fs.Open(file.Path(v.fileDir, idk.Value))
	if err != nil {
		return nil, err
	}

	go func() {
		defer fh.Close()
		defer pw.Close()
		io.Copy(pw, fh)
	}()

	return pr, nil
}

func (v *volume) DeleteFile(key string) error {
	return v.startUnitOfWork(uow.Read, func(uw uow.UnitOfWork) error {
		ik, err := uw.IDXKeys().FindByKey(key)
		if err != nil {
			return err
		}
		dbf, err := uw.Files().FindBySignature(ik.Value)
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
			err = uw.Files().DeleteBySignature(ik.Value)
			if err != nil {
				return err
			}

			err = v.fs.Remove(file.Path(v.fileDir, ik.Value))
			if err != nil {
				return err
			}
		} else {
			dbf.Keys = newKeys

			err = uw.Files().CreateOrReplace(dbf)
			if err != nil {
				return err
			}
		}

		return uw.IDXKeys().DeleteByKey(key)
	}, v.idxkeys, v.files)
}

func (v *volume) HasFile(k string) (bool, error) {
	err := v.startUnitOfWork(uow.Read, func(uw uow.UnitOfWork) error {
		_, err := uw.IDXKeys().FindByKey(k)
		if err != nil {
			return err
		}
		return nil
	}, v.idxkeys)

	if err != nil {
		if err.Error() == "not found" {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
