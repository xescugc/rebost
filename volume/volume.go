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
	"github.com/xescugc/rebost/replica"
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
	CreateFile(ctx context.Context, key string, reader io.ReadCloser, replica int) error

	// GetFile search for the file with the key
	GetFile(ctx context.Context, key string) (io.ReadCloser, error)

	// HasFile checks if a file with the key exists
	HasFile(ctx context.Context, key string) (bool, error)

	// DeleteFile deletes the key, if the key points to a
	// file with 2 keys, then just the key will be deleted
	// and not the content
	DeleteFile(ctx context.Context, key string) error
}

//go:generate mockgen -destination=../mock/volume_local.go -mock_names=Local=VolumeLocal -package=mock github.com/xescugc/rebost/volume Local

// Local is the definition of a Local volume which
// is an extension of the volume.Volume
type Local interface {
	Volume

	// Close will try to make a clean shutdown
	io.Closer

	// ID returns the ID of the Volume
	ID() string

	// NextReplica returns the next replica
	// inline. A "not found" error means
	// no replica is needed
	NextReplica(ctx context.Context) (*replica.Replica, error)

	// UpdateReplica updates the rp of the index and the File to include
	// the vID as a volume with the Replica
	UpdateReplica(ctx context.Context, rp *replica.Replica, vID string) error
}

type local struct {
	fileDir string
	tempDir string
	id      string

	fs       afero.Fs
	files    file.Repository
	idxkeys  idxkey.Repository
	replicas replica.Repository

	startUnitOfWork uow.StartUnitOfWork

	ctx    context.Context
	cancel context.CancelFunc
}

// New returns an implementation of the volume.Local interface using the provided parameters
// it can return an error because when initialized it also creates the needed directories
// if they are missing which are $root/file and $root/tmps and also the ID
func New(root string, files file.Repository, idxkeys idxkey.Repository, rp replica.Repository, fileSystem afero.Fs, suow uow.StartUnitOfWork) (Local, error) {
	ctx, cancel := context.WithCancel(context.Background())
	l := &local{
		fileDir: path.Join(root, "file"),
		tempDir: path.Join(root, "tmps"),

		files:    files,
		fs:       fileSystem,
		idxkeys:  idxkeys,
		replicas: rp,

		startUnitOfWork: suow,

		ctx:    ctx,
		cancel: cancel,
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

func (l *local) Close() error {
	l.cancel()
	return nil
}

func (l *local) CreateFile(ctx context.Context, key string, r io.ReadCloser, rep int) error {
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
		Replica:   rep,
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

	err = l.startUnitOfWork(ctx, uow.Write, func(ctx context.Context, uw uow.UnitOfWork) error {
		dbf, err := uw.Files().FindBySignature(ctx, f.Signature)
		if err != nil && err.Error() != "not found" {
			return err
		}

		// If the File already exists on the DB with that signature
		// we have to add another key if it's not already there
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

		// If we have an IDXKey with the same key we are storing
		// means that we have a name colision. We will:
		// * Remove the new key from the File.Keys
		// * If the len(File.Keys) == 0 we'll remove that File/IDXKey
		// * If the len(File.Keys) != 0 we'll update that File
		// At the end the new key will replace the old one found
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
				// If no keys we remove the File
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
				// If some keys left we update the File
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

		// As one is already stored on this volume we can reduce it
		rep--
		if rep >= 1 {
			rp := &replica.Replica{
				ID:  uuid.NewV4().String(),
				Key: key,
				// TODO: For now we are ignoring the fact
				// that if the file exists the replicas may
				// chage and be more or lesss
				Count:         rep,
				OriginalCount: rep + 1,
				Signature:     f.Signature,
				VolumeID:      l.id,
			}

			err = uw.Replicas().Create(ctx, rp)
			if err != nil {
				return err
			}
		}

		return nil
	}, l.idxkeys, l.files, l.fs, l.replicas)

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

	err = l.startUnitOfWork(ctx, uow.Read, func(ctx context.Context, uw uow.UnitOfWork) error {
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
	return l.startUnitOfWork(ctx, uow.Write, func(ctx context.Context, uw uow.UnitOfWork) error {
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
	err := l.startUnitOfWork(ctx, uow.Read, func(ctx context.Context, uw uow.UnitOfWork) error {
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

func (l *local) NextReplica(ctx context.Context) (*replica.Replica, error) {
	var (
		err error
		rp  *replica.Replica
	)
	err = l.startUnitOfWork(ctx, uow.Write, func(ctx context.Context, uw uow.UnitOfWork) error {
		rp, err = uw.Replicas().First(ctx)
		if err != nil {
			return err
		}
		return nil
	}, l.replicas)
	if err != nil {
		return nil, err
	}
	return rp, nil
}

func (l *local) UpdateReplica(ctx context.Context, rp *replica.Replica, vID string) error {
	if rp == nil {
		return fmt.Errorf("the replica is required")
	}
	if rp.Signature == "" && rp.Key == "" {
		return fmt.Errorf("the replica Signature or Key are required")
	}
	if rp.OriginalCount == 0 {
		return fmt.Errorf("the replica OriginalCount is required")
	}
	err := l.startUnitOfWork(ctx, uow.Write, func(ctx context.Context, uw uow.UnitOfWork) error {
		var (
			f   *file.File
			err error
		)

		// If we have Signature, when it's the one internal Update, we use it\
		// if we have the key, from remote, we get it from there
		if rp.Signature != "" {
			f, err = uw.Files().FindBySignature(ctx, rp.Signature)
		} else {
			ik, err := uw.IDXKeys().FindByKey(ctx, rp.Key)
			if err != nil && err.Error() != "not found" {
				return err
			}
			f, err = uw.Files().FindBySignature(ctx, ik.Value)
		}
		if err != nil {
			return err
		}

		f.VolumeIDs = append(f.VolumeIDs, vID)
		f.Replica = rp.OriginalCount

		err = uw.Files().CreateOrReplace(ctx, f)
		if err != nil {
			return err
		}

		// If it's not the same volume menas
		// that it's an outise replica so we just have
		// to update the File
		if rp.VolumeID != l.id {
			return nil
		}

		// Delete the replica from the  queue to reinsert it later
		// with a different Count
		err = uw.Replicas().Delete(ctx, rp)
		if err != nil {
			return err
		}

		rp.Count -= 1

		if rp.Count > 0 {
			err = uw.Replicas().Create(ctx, rp)
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}
