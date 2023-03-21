package volume

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"code.cloudfoundry.org/bytefmt"
	kitlog "github.com/go-kit/kit/log"
	uuid "github.com/satori/go.uuid"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/spf13/afero"
	"github.com/xescugc/rebost/file"
	"github.com/xescugc/rebost/idxkey"
	"github.com/xescugc/rebost/idxvolume"
	"github.com/xescugc/rebost/replica"
	"github.com/xescugc/rebost/state"
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

	// HasFile checks if a file with the key exists and returns the volumeID
	// of where is it.
	// It's possible to return a vid but false that means we know which volume
	// has it but it's not this one
	HasFile(ctx context.Context, key string) (string, bool, error)

	// DeleteFile deletes the key, if the key points to a
	// file with 2 keys, then just the key will be deleted
	// and not the content
	DeleteFile(ctx context.Context, key string) error

	// UpdateFileReplica updates the Replica information of the file
	// with the given one basically replacing it
	UpdateFileReplica(ctx context.Context, key string, volumeIDs []string, replica int) error
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

	// SynchronizeReplicas checks the replicas related with vID and
	// if this volume is the responsible (next after the removed ID on the files)
	// will start replication of those files which have to
	SynchronizeReplicas(ctx context.Context, vID string) error

	// GetState returns the current State of the volume
	GetState(ctx context.Context) (*state.State, error)
}

type local struct {
	fileDir string
	tempDir string
	id      string

	fs         afero.Fs
	files      file.Repository
	idxkeys    idxkey.Repository
	replicas   replica.Repository
	idxvolumes idxvolume.Repository
	state      state.Repository

	startUnitOfWork uow.StartUnitOfWork

	logger kitlog.Logger

	ctx    context.Context
	cancel context.CancelFunc
}

// New returns an implementation of the volume.Local interface using the provided parameters
// it can return an error because when initialized it also creates the needed directories
// if they are missing which are $root/file and $root/tmps and also the ID
// To define a total size of the volume it has to be appended to the root like `/v1:1GB`
func New(root string, files file.Repository, idxkeys idxkey.Repository, idxvolumes idxvolume.Repository, rp replica.Repository, sr state.Repository, fileSystem afero.Fs, logger kitlog.Logger, suow uow.StartUnitOfWork) (Local, error) {
	ctx, cancel := context.WithCancel(context.Background())
	sroot := strings.Split(root, ":")
	ts := -1
	if len(sroot) > 1 {
		b, err := bytefmt.ToBytes(sroot[1])
		if err != nil {
			cancel()
			return nil, err
		}
		ts = int(b)
		root = sroot[0]
	}
	l := &local{
		fileDir: path.Join(root, "file"),
		tempDir: path.Join(root, "tmps"),

		files:      files,
		fs:         fileSystem,
		idxkeys:    idxkeys,
		idxvolumes: idxvolumes,
		replicas:   rp,
		state:      sr,

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
	l.logger = kitlog.With(logger, "src", "volume", "id", id)

	// Initialize state
	err = l.calculateSize(ctx, root, ts)
	if err != nil {
		return nil, err
	}

	// Every minute we update the State so
	// we can check if anything has changed on
	// the overall System
	go func() {
		tk := time.NewTicker(time.Minute)
		for {
			select {
			case <-ctx.Done():
				tk.Stop()
			case <-tk.C:
				err = l.calculateSize(ctx, root, ts)
				if err != nil {
					l.logger.Log("msg", err.Error())
				}
			}
		}
	}()

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

	fi, err := fh.Stat()
	if err != nil {
		return err
	}
	f := &file.File{
		Keys:      []string{key},
		Signature: fmt.Sprintf("%x", sh1.Sum(nil)),
		Replica:   rep,
		Size:      int(fi.Size()),
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
		} else {
			// Update the State with the new file
			// created size
			st, err := uw.State().Find(ctx, l.id)
			if err != nil {
				return err
			}
			if !st.Use(f.Size) {
				return errors.New("file is too large for the dedicated space left")
			}

			err = uw.State().Update(ctx, l.id, st)
			if err != nil {
				return err
			}
		}

		f.VolumeIDs = append(f.VolumeIDs, l.ID())

		err = uw.Files().CreateOrReplace(ctx, f)
		if err != nil {
			return err
		}

		ik, err := uw.IDXKeys().FindByKey(ctx, key)
		if err != nil && err.Error() != "not found" {
			return err
		}

		// If we have an IDXKey with the same key we are storing
		// means that we have a name collision. We will:
		// * Remove the new key from the File.Keys
		// * If the len(File.Keys) == 0 we'll remove that File/IDXKey
		// * If the len(File.Keys) != 0 we'll update that File
		// At the end the new key will replace the old one found
		// TODO: Update the value on the idxvolumes
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
				VolumeIDs:     []string{l.id},
				VolumeID:      l.id,
			}

			err = uw.Replicas().Create(ctx, rp)
			if err != nil {
				return err
			}
		}

		return nil
	}, l.idxkeys, l.files, l.fs, l.replicas, l.state)

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

			// Update the State with the new file
			// created size
			st, err := uw.State().Find(ctx, l.id)
			if err != nil {
				return err
			}
			if !st.Use(-dbf.Size) {
				return errors.New("file is too large for the dedicated space left")
			}

			err = uw.State().Update(ctx, l.id, st)
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
	}, l.idxkeys, l.files, l.fs, l.state)
}

func (l *local) HasFile(ctx context.Context, k string) (string, bool, error) {
	err := l.startUnitOfWork(ctx, uow.Read, func(ctx context.Context, uw uow.UnitOfWork) error {
		_, err := uw.IDXKeys().FindByKey(ctx, k)
		if err != nil {
			return err
		}
		return nil
	}, l.idxkeys)

	if err != nil {
		if err.Error() == "not found" {
			return "", false, nil
		}
		return "", false, err
	}

	return l.id, true, nil
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
	if rp.Signature == "" {
		return fmt.Errorf("the replica Signature is required")
	}
	if rp.OriginalCount == 0 {
		return fmt.Errorf("the replica OriginalCount is required")
	}
	err := l.startUnitOfWork(ctx, uow.Write, func(ctx context.Context, uw uow.UnitOfWork) error {
		f, err := uw.Files().FindBySignature(ctx, rp.Signature)
		if err != nil {
			return err
		}

		f.VolumeIDs = append(f.VolumeIDs, vID)
		f.Replica = rp.OriginalCount

		idxv, err := uw.IDXVolumes().FindByVolumeID(ctx, vID)
		if err != nil {
			if err.Error() == "not found" {
				idxv = idxvolume.New(vID, []string{})
			} else {
				return err
			}
		}

		idxv.AddSignature(f.Signature)

		err = uw.IDXVolumes().CreateOrReplace(ctx, idxv)
		if err != nil {
			return err
		}

		err = uw.Files().CreateOrReplace(ctx, f)
		if err != nil {
			return err
		}

		// Delete the replica from the  queue to reinsert it later
		// with a different Count
		err = uw.Replicas().Delete(ctx, rp)
		if err != nil {
			return err
		}

		rp.Count--

		if rp.Count > 0 {
			err = uw.Replicas().Create(ctx, rp)
			if err != nil {
				return err
			}
		}

		return nil
	}, l.replicas, l.files, l.idxkeys, l.idxvolumes)

	if err != nil {
		return err
	}

	return nil
}

func (l *local) UpdateFileReplica(ctx context.Context, key string, volumeIDs []string, replica int) error {
	err := l.startUnitOfWork(ctx, uow.Write, func(ctx context.Context, uw uow.UnitOfWork) error {

		var found bool
		for _, v := range volumeIDs {
			if v == l.ID() {
				found = true
				break
			}
		}
		if !found {
			return errors.New("the volume ID has to be on the list of volume")
		}

		ik, err := uw.IDXKeys().FindByKey(ctx, key)
		if err != nil {
			return err
		}
		f, err := uw.Files().FindBySignature(ctx, ik.Value)
		if err != nil {
			return err
		}

		// For all the volumes we add the signature to the IDX
		// so we can easily keep track of which file replicas
		// are in which nodes
		for _, vid := range volumeIDs {
			// If it's this Volume we do not need to do
			// any of this as it's not required
			if vid == l.ID() {
				continue
			}
			idxv, err := uw.IDXVolumes().FindByVolumeID(ctx, vid)
			if err != nil {
				if err.Error() == "not found" {
					idxv = idxvolume.New(vid, []string{})
				} else {
					return err
				}
			}

			idxv.AddSignature(f.Signature)

			err = uw.IDXVolumes().CreateOrReplace(ctx, idxv)
			if err != nil {
				return err
			}
		}

		// TODO: Diff the VolumeIDs and update/create/delete signature from
		// the required idxvolumes to maintain the consistency
		f.VolumeIDs = volumeIDs
		f.Replica = replica

		err = uw.Files().CreateOrReplace(ctx, f)
		if err != nil {
			return err
		}
		return nil
	}, l.files, l.idxkeys, l.idxvolumes)

	if err != nil {
		return err
	}

	return nil
}

func (l *local) SynchronizeReplicas(ctx context.Context, vID string) error {
	err := l.startUnitOfWork(ctx, uow.Write, func(ctx context.Context, uw uow.UnitOfWork) error {
		idxvol, err := uw.IDXVolumes().FindByVolumeID(ctx, vID)
		if err != nil {
			return err
		}

		// Two use cases, one to check if you are the next
		for _, s := range idxvol.Signatures {
			f, err := uw.Files().FindBySignature(ctx, s)
			if err != nil {
				return err
			}

			f.DeleteVolumeID(vID)

			// If after deleting the vID from the
			// file this Node is the first one means
			// it's the master of the file so it has to start
			// replicating
			if f.VolumeIDs[0] == l.ID() {
				numOfReplicasMissing := f.Replica - len(f.VolumeIDs)

				// TODO: We do not know which key was assigned
				// to that Volume so we use the 0 by default
				// this is another argument to not use "Key" and
				// generate an ID for each file and we just
				// group by Signature and not Key
				rp := &replica.Replica{
					ID:            uuid.NewV4().String(),
					Key:           f.Keys[0],
					Count:         numOfReplicasMissing,
					OriginalCount: f.Replica,
					Signature:     f.Signature,
					VolumeID:      l.id,
					VolumeIDs:     f.VolumeIDs,
				}

				err = uw.Replicas().Create(ctx, rp)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}, l.files, l.idxvolumes, l.replicas)

	if err != nil {
		return err
	}

	return nil
}

func (l *local) GetState(ctx context.Context) (*state.State, error) {
	var (
		s   *state.State
		err error
	)

	err = l.startUnitOfWork(ctx, uow.Read, func(ctx context.Context, uw uow.UnitOfWork) error {
		s, err = uw.State().Find(ctx, l.id)
		if err != nil {
			return err
		}
		return nil
	}, l.state)

	if err != nil {
		return nil, err
	}

	return s, nil
}

func (l *local) calculateSize(ctx context.Context, root string, ts int) error {
	err := l.startUnitOfWork(ctx, uow.Write, func(ctx context.Context, uw uow.UnitOfWork) error {
		s, err := uw.State().Find(ctx, l.id)
		if err != nil {
			return err
		}
		us, err := disk.Usage(root)
		if err != nil {
			return err
		}

		ps, err := disk.Partitions(true)
		if err != nil {
			return err
		}
		s.SystemTotalSize = int(us.Total)
		s.SystemUsedSize = int(us.Used)
		s.VolumeTotalSize = ts
		for _, p := range ps {
			mre := regexp.MustCompile(fmt.Sprintf("%s.*", p.Mountpoint))
			if mre.MatchString(root) {
				if len(s.Mountpoint) < len(p.Mountpoint) {
					s.Mountpoint = p.Mountpoint
				}
			}
		}
		err = uw.State().Update(ctx, l.id, s)
		if err != nil {
			return err
		}

		return nil
	}, l.state)
	if err != nil {
		return err
	}
	return nil
}
