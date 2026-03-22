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

	"log/slog"

	"code.cloudfoundry.org/bytefmt"

	uuid "github.com/satori/go.uuid"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/spf13/afero"
	"github.com/xescugc/rebost/deletion"
	"github.com/xescugc/rebost/file"
	"github.com/xescugc/rebost/idxkey"
	"github.com/xescugc/rebost/idxttl"
	"github.com/xescugc/rebost/idxvolume"
	"github.com/xescugc/rebost/replica"
	"github.com/xescugc/rebost/state"
	"github.com/xescugc/rebost/uow"
)

const (
	// TickerDuration is the duration of the internal loop of the volume
	// to recalculate the state
	TickerDuration = 30 * time.Second

	// noTTL is used to have a readable comparison
	// when dealing with TTL logic
	noTTL = 0
)

// FileStat holds metadata about a stored file, used for S3-compatible HEAD responses.
type FileStat struct {
	Size    int64
	ETag    string
	ModTime time.Time
}

//go:generate go tool mockgen -destination=../mock/volume.go -mock_names=Volume=Volume -package=mock github.com/xescugc/rebost/volume Volume

// Volume is an interface to deal with the simples actions
// and basic ones
type Volume interface {
	// CreateFile creates a new file from the reader with the key, ttl and time of creation (if empty will be set to now)
	// There are 4 different use cases to consider:
	// * New key and reader
	// * New key with already known reader
	// * Already known key with new reader
	// * Already known key and reader
	CreateFile(ctx context.Context, key string, reader io.ReadCloser, replica int, ttl time.Duration, ca time.Time) error

	// GetFile search for the file with the key
	GetFile(ctx context.Context, key string) (io.ReadCloser, int64, error)

	// StatFile returns metadata about the file without reading its content.
	StatFile(ctx context.Context, key string) (*FileStat, error)

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

//go:generate go tool mockgen -destination=../mock/volume_local.go -mock_names=Local=VolumeLocal -package=mock github.com/xescugc/rebost/volume Local

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

	// DeleteReplica removes a pending replication job from the queue.
	// Used to clean up stale jobs when the source file no longer exists.
	DeleteReplica(ctx context.Context, rp *replica.Replica) error

	// NextDeletion returns the next pending remote-deletion job.
	// A "not found" error means the queue is empty.
	NextDeletion(ctx context.Context) (*deletion.Deletion, error)

	// DeleteDeletion removes a processed deletion job from the queue.
	DeleteDeletion(ctx context.Context, d *deletion.Deletion) error

	// SynchronizeReplicas checks the replicas related with vID and
	// if this volume is the responsible (next after the removed ID on the files)
	// will start replication of those files which have to
	SynchronizeReplicas(ctx context.Context, vID string) error

	// GetState returns the current State of the volume
	GetState(ctx context.Context) (*state.State, error)

	// Reset will clean all the data of the volume and even change the ID
	Reset(ctx context.Context) error

	// PrepareForDrain creates Replica jobs for all local files that don't have
	// enough external replicas to cover the configured replica count.
	PrepareForDrain(ctx context.Context) error

	// HasPendingReplicas reports whether there are any pending replica jobs queued.
	HasPendingReplicas(ctx context.Context) (bool, error)

	// PurgeAllFiles removes all local files and their keys from this volume
	// without creating deletion jobs for remote nodes (remote copies are preserved).
	PurgeAllFiles(ctx context.Context) error
}

type local struct {
	fileDir string
	tempDir string
	id      string

	root      string
	totalSize int

	fs afero.Fs

	startUnitOfWork uow.StartUnitOfWork

	logger         *slog.Logger
	originalLogger *slog.Logger

	ctx    context.Context
	cancel context.CancelFunc
}

// New returns an implementation of the volume.Local interface using the provided parameters
// it can return an error because when initialized it also creates the needed directories
// if they are missing which are $root/file and $root/tmps and also the ID
// To define a total size of the volume it has to be appended to the root like `/v1:1GB`
func New(root string, fileSystem afero.Fs, logger *slog.Logger, suow uow.StartUnitOfWork) (Local, error) {
	if logger == nil {
		logger = slog.Default()
	}
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
		fileDir:   path.Join(root, "file"),
		tempDir:   path.Join(root, "tmps"),
		root:      root,
		totalSize: ts,

		fs: fileSystem,

		originalLogger: logger,

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
		id, err = l.createID(idPath)
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
	l.logger = logger.With("src", "volume", "id", id)

	// Initialize state
	err = l.startUnitOfWork(ctx, uow.Write, func(ctx context.Context, uw uow.UnitOfWork) error {
		err = l.calculateSize(ctx, uw, root, ts)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Loop that updates the State so
	// we can check if anything has changed on
	// the overall System
	go func() {
		tk := time.NewTicker(TickerDuration)
		for {
			select {
			case <-ctx.Done():
				tk.Stop()
			case <-tk.C:
				err = l.startUnitOfWork(ctx, uow.Write, func(ctx context.Context, uw uow.UnitOfWork) error {
					err = l.calculateSize(ctx, uw, root, ts)
					if err != nil {
						l.logger.Error(err.Error())
					}
					return nil
				})
			}
		}
	}()

	// We check if there is any TTL expiring
	go l.loopTTL()

	return l, nil
}

func (l *local) ID() string { return l.id }

func (l *local) Close() error {
	l.cancel()
	return nil
}

func (l *local) CreateFile(ctx context.Context, key string, r io.ReadCloser, rep int, ttl time.Duration, ca time.Time) error {
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
	// If no time is set then we use the current date
	if ca.IsZero() {
		ca = time.Now()
	}
	f := &file.File{
		Keys:      []string{key},
		Signature: fmt.Sprintf("%x", sh1.Sum(nil)),
		Replica:   rep,
		Size:      int(fi.Size()),
		TTL:       ttl,
		CreatedAt: ca,
	}

	p := f.Path(l.fileDir)
	dir, _ := path.Split(p)

	err = l.fs.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return err
	}

	// -- Phase 1: capacity reservation (UoW write) --
	// Check capacity before renaming the temp file to its final path.
	// On failure, clean up the temp file to avoid orphans.
	var capacityReserved bool
	err = l.startUnitOfWork(ctx, uow.Write, func(ctx context.Context, uw uow.UnitOfWork) error {
		dbf, err := uw.Files().FindBySignature(ctx, f.Signature)
		if err != nil && err.Error() != "not found" {
			return err
		}
		if dbf != nil {
			// Deduplication: existing content, no capacity charge needed.
			return nil
		}
		st, err := uw.State().Find(ctx)
		if err != nil {
			return err
		}
		if !st.Use(f.Size) {
			return ErrNoSpace
		}
		capacityReserved = true
		return uw.State().Update(ctx, st)
	})
	if err != nil {
		// Temp file still at tmp; remove it to avoid an orphan.
		_ = l.fs.Remove(tmp)
		return err
	}

	// -- Rename only after capacity is confirmed --
	if err = l.fs.Rename(tmp, p); err != nil {
		// Roll back the capacity reservation committed in Phase 1.
		// Best-effort: ignore rollback errors to avoid masking the original rename error.
		if capacityReserved {
			_ = l.startUnitOfWork(ctx, uow.Write, func(ctx context.Context, uw uow.UnitOfWork) error {
				st, stErr := uw.State().Find(ctx)
				if stErr != nil {
					return stErr
				}
				st.Use(-f.Size)
				return uw.State().Update(ctx, st)
			})
		}
		return err
	}

	// -- Phase 2: DB metadata write (UoW write) --
	// Capacity was already reserved in Phase 1; skip the Use/Update block here.
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

		// We check if there is an expiration date for the file
		// already on the IDXTTLs
		if ttl != noTTL {
			dbidxttl, err := uw.IDXTTLs().Find(ctx, f.ExpiresAt())
			if err != nil && err.Error() != "not found" {
				return err
			}

			// The case of the not found
			if dbidxttl == nil {
				dbidxttl = &idxttl.IDXTTL{ExpiresAt: f.ExpiresAt()}
			}
			dbidxttl.AddSignatures(f.Signature)

			err = uw.IDXTTLs().CreateOrReplace(ctx, dbidxttl)
			if err != nil {
				return err
			}
		}

		// As one is already stored on this volume we can reduce it
		rep--
		if rep >= 1 {
			rp := &replica.Replica{
				ID:  uuid.NewV4().String(),
				Key: key,
				// TODO: For now we are ignoring the fact
				// that if the file exists the replicas may
				// change and be more or less
				Count:         rep,
				OriginalCount: rep + 1,
				Signature:     f.Signature,
				VolumeIDs:     []string{l.id},
				VolumeID:      l.id,
				TTL:           ttl,
				CreatedAt:     ca,
			}

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

func (l *local) GetFile(ctx context.Context, k string) (io.ReadCloser, int64, error) {
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
	})

	if err != nil {
		return nil, -1, err
	}

	fh, err := l.fs.Open(file.Path(l.fileDir, idk.Value))
	if err != nil {
		return nil, -1, err
	}

	info, err := fh.Stat()
	if err != nil {
		return fh, -1, nil
	}

	return fh, info.Size(), nil
}

func (l *local) StatFile(ctx context.Context, k string) (*FileStat, error) {
	var stat FileStat
	err := l.startUnitOfWork(ctx, uow.Read, func(ctx context.Context, uw uow.UnitOfWork) error {
		idk, err := uw.IDXKeys().FindByKey(ctx, k)
		if err != nil {
			return err
		}
		f, err := uw.Files().FindBySignature(ctx, idk.Value)
		if err != nil {
			return err
		}
		stat = FileStat{
			Size:    int64(f.Size),
			ETag:    f.Signature,
			ModTime: f.CreatedAt,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &stat, nil
}

func (l *local) DeleteFile(ctx context.Context, key string) error {
	return l.startUnitOfWork(ctx, uow.Write, func(ctx context.Context, uw uow.UnitOfWork) error {
		return l.deleteFile(ctx, uw, key)
	})
}

func (l *local) deleteFile(ctx context.Context, uw uow.UnitOfWork, key string) error {
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
		err = l.purgeFile(ctx, dbf, uw)
		if err != nil {
			return err
		}

		// Schedule deletion on all remote replica volumes.
		remoteVIDs := make([]string, 0, len(dbf.VolumeIDs))
		for _, vid := range dbf.VolumeIDs {
			if vid != l.id {
				remoteVIDs = append(remoteVIDs, vid)
			}
		}
		if len(remoteVIDs) > 0 {
			err = uw.Deletions().Create(ctx, &deletion.Deletion{
				Key:       key,
				VolumeIDs: remoteVIDs,
			})
			if err != nil {
				return err
			}
		}

	} else {
		dbf.Keys = newKeys

		err = uw.Files().CreateOrReplace(ctx, dbf)
		if err != nil {
			return err
		}
	}

	return uw.IDXKeys().DeleteByKey(ctx, key)
}

// purgeFile removes a file's content and DB record from this volume without
// creating a Deletion job for remote nodes. All keys in f.Keys are assumed to
// already be removed from idxkey before calling this (or are about to be removed
// by the caller). The file DB record, any pending replica job, the filesystem
// content, and the state size are all cleaned up.
func (l *local) purgeFile(ctx context.Context, f *file.File, uw uow.UnitOfWork) error {
	err := uw.Files().DeleteBySignature(ctx, f.Signature)
	if err != nil {
		return err
	}

	err = uw.Replicas().Delete(ctx, &replica.Replica{VolumeReplicaID: []byte(f.Signature)})
	if err != nil {
		return err
	}

	err = uw.Fs().Remove(file.Path(l.fileDir, f.Signature))
	if err != nil {
		return err
	}

	// Update the State to reflect the freed space.
	st, err := uw.State().Find(ctx)
	if err != nil {
		return err
	}
	if !st.Use(-f.Size) {
		return errors.New("volume accounting underflow: used size would go negative")
	}

	return uw.State().Update(ctx, st)
}

func (l *local) HasFile(ctx context.Context, k string) (string, bool, error) {
	err := l.startUnitOfWork(ctx, uow.Read, func(ctx context.Context, uw uow.UnitOfWork) error {
		_, err := uw.IDXKeys().FindByKey(ctx, k)
		if err != nil {
			return err
		}
		return nil
	})

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
	})
	if err != nil {
		return nil, err
	}
	return rp, nil
}

func (l *local) DeleteReplica(ctx context.Context, rp *replica.Replica) error {
	return l.startUnitOfWork(ctx, uow.Write, func(ctx context.Context, uw uow.UnitOfWork) error {
		return uw.Replicas().Delete(ctx, rp)
	})
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
	})

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
	})

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
	})

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
		s, err = uw.State().Find(ctx)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return s, nil
}

func (l *local) Reset(ctx context.Context) error {
	err := l.startUnitOfWork(ctx, uow.Write, func(ctx context.Context, uw uow.UnitOfWork) error {
		err := uw.Files().DeleteAll(ctx)
		if err != nil {
			return err
		}

		err = uw.IDXKeys().DeleteAll(ctx)
		if err != nil {
			return err
		}

		err = uw.Replicas().DeleteAll(ctx)
		if err != nil {
			return err
		}

		err = uw.IDXVolumes().DeleteAll(ctx)
		if err != nil {
			return err
		}

		err = uw.Fs().RemoveAll(l.fileDir)
		if err != nil {
			return err
		}

		err = uw.Fs().RemoveAll(l.tempDir)
		if err != nil {
			return err
		}

		err = uw.State().DeleteAll(ctx)
		if err != nil {
			return err
		}

		idPath := path.Join(l.root, "id")
		err = uw.Fs().Remove(idPath)
		if err != nil {
			return err
		}

		id, _ := l.createID(idPath)
		l.id = id
		l.logger = l.originalLogger.With("src", "volume", "id", id)

		l.calculateSize(ctx, uw, l.root, l.totalSize)
		return nil
	})
	if err != nil {
		return err
	}
	// TODO: Recreate the ID
	return nil
}

func (l *local) NextDeletion(ctx context.Context) (*deletion.Deletion, error) {
	var (
		err error
		d   *deletion.Deletion
	)
	err = l.startUnitOfWork(ctx, uow.Write, func(ctx context.Context, uw uow.UnitOfWork) error {
		d, err = uw.Deletions().First(ctx)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (l *local) DeleteDeletion(ctx context.Context, d *deletion.Deletion) error {
	return l.startUnitOfWork(ctx, uow.Write, func(ctx context.Context, uw uow.UnitOfWork) error {
		return uw.Deletions().Delete(ctx, d)
	})
}

func (l *local) PrepareForDrain(ctx context.Context) error {
	// Duplicate replica jobs for files that already have some pending
	// replication are intentional: the replication processor performs a
	// HasFile check before transferring content, making each job idempotent.
	return l.startUnitOfWork(ctx, uow.Write, func(ctx context.Context, uw uow.UnitOfWork) error {
		files, err := uw.Files().All(ctx)
		if err != nil {
			return err
		}

		for _, f := range files {
			externalCount := 0
			vidsWithoutSelf := make([]string, 0, len(f.VolumeIDs))
			for _, vid := range f.VolumeIDs {
				if vid != l.id {
					externalCount++
					vidsWithoutSelf = append(vidsWithoutSelf, vid)
				}
			}

			if externalCount < f.Replica {
				rp := &replica.Replica{
					ID:            uuid.NewV4().String(),
					Key:           f.Keys[0],
					Count:         f.Replica - externalCount,
					OriginalCount: f.Replica,
					Signature:     f.Signature,
					VolumeID:      l.id,
					VolumeIDs:     vidsWithoutSelf,
					TTL:           f.TTL,
					CreatedAt:     f.CreatedAt,
				}

				err = uw.Replicas().Create(ctx, rp)
				if err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func (l *local) HasPendingReplicas(ctx context.Context) (bool, error) {
	var (
		has bool
		err error
	)
	err = l.startUnitOfWork(ctx, uow.Read, func(ctx context.Context, uw uow.UnitOfWork) error {
		has, err = uw.Replicas().HasAny(ctx)
		return err
	})
	if err != nil {
		return false, err
	}
	return has, nil
}

func (l *local) PurgeAllFiles(ctx context.Context) error {
	// Two-phase approach: read the full file list in a single read transaction
	// (snapshot), then write-delete each file in its own transaction. This
	// avoids holding one large BoltDB write-transaction lock for the entire
	// operation, which would block every concurrent reader for its duration.
	var files []*file.File
	err := l.startUnitOfWork(ctx, uow.Read, func(ctx context.Context, uw uow.UnitOfWork) error {
		var err error
		files, err = uw.Files().All(ctx)
		return err
	})
	if err != nil {
		return err
	}

	for _, f := range files {
		err := l.startUnitOfWork(ctx, uow.Write, func(ctx context.Context, uw uow.UnitOfWork) error {
			// Remove all idxkey entries for this file.
			for _, k := range f.Keys {
				if err := uw.IDXKeys().DeleteByKey(ctx, k); err != nil {
					return err
				}
			}

			// Remove this file's signature from the idxttl entry if it has a TTL set.
			// Without this cleanup the loopTTL goroutine will log "not found"
			// errors every second for each stale entry.
			// Use Find-remove-update so we don't clobber other files that share
			// the same ExpiresAt timestamp.
			if f.TTL != noTTL {
				dbidxttl, err := uw.IDXTTLs().Find(ctx, f.ExpiresAt())
				if err != nil && err.Error() != "not found" {
					return err
				}
				if err == nil {
					newSigs := make([]string, 0, len(dbidxttl.Signatures))
					for _, s := range dbidxttl.Signatures {
						if s != f.Signature {
							newSigs = append(newSigs, s)
						}
					}
					if len(newSigs) == 0 {
						if err := uw.IDXTTLs().Delete(ctx, f.ExpiresAt()); err != nil {
							return err
						}
					} else {
						dbidxttl.Signatures = newSigs
						if err := uw.IDXTTLs().CreateOrReplace(ctx, dbidxttl); err != nil {
							return err
						}
					}
				}
			}

			return l.purgeFile(ctx, f, uw)
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (l *local) createID(idPath string) (string, error) {
	id := uuid.NewV4().String()
	fh, err := l.fs.Create(idPath)
	if err != nil {
		return "", err
	}
	defer fh.Close()

	_, err = io.WriteString(fh, id)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (l *local) calculateSize(ctx context.Context, uw uow.UnitOfWork, root string, ts int) error {
	s, err := uw.State().Find(ctx)
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
	s.UpdatedAt = time.Now()
	err = uw.State().Update(ctx, s)
	if err != nil {
		return err
	}

	return nil
}
