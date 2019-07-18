package mock

import (
	context "context"
	"errors"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/spf13/afero/mem"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/rebost/uow"
	"github.com/xescugc/rebost/volume"
)

// ManageVolume is a test structure that hepls initialize a
// Volume with all his 'mocks' initialized an ready to use
type ManageVolume struct {
	Files          *FileRepository
	IDXKeys        *IDXKeyRepository
	Fs             *Fs
	ReplicaPendent *ReplicaPendentRepository
	ReplicaRetry   *ReplicaRetryRepository

	V volume.Local

	ctrl *gomock.Controller
}

// NewManageVolume returns the initialization of the ManageVolume
// with all the mocks
func NewManageVolume(t *testing.T, root string) ManageVolume {
	ctrl := gomock.NewController(t)

	files := NewFileRepository(ctrl)
	idxkeys := NewIDXKeyRepository(ctrl)
	fs := NewFs(ctrl)
	rr := NewReplicaRetryRepository(ctrl)
	rp := NewReplicaPendentRepository(ctrl)

	uowFn := func(ctx context.Context, t uow.Type, uowFn uow.UnitOfWorkFn, repositories ...interface{}) error {
		uw := NewUnitOfWork(ctrl)
		uw.EXPECT().Files().Return(files).AnyTimes()
		uw.EXPECT().IDXKeys().Return(idxkeys).AnyTimes()
		uw.EXPECT().Fs().Return(fs).AnyTimes()
		uw.EXPECT().ReplicaPendent().Return(rp).AnyTimes()
		uw.EXPECT().ReplicaRetry().Return(rr).AnyTimes()
		return uowFn(ctx, uw)
	}

	// This first implementation is already tested
	// so we do not need it
	fs.EXPECT().MkdirAll(gomock.Any(), gomock.Any()).Return(nil).Times(2)
	fs.EXPECT().Stat(gomock.Any()).Return(nil, os.ErrNotExist)
	fs.EXPECT().Create(gomock.Any()).Return(mem.NewFileHandle(mem.CreateFile("")), nil)

	// As the volume.New starts a goroutine we have to use this mock to
	// always returns the 'nil' object so it does nothing on it and the
	// test do not fail, this function could or could not be called as it's
	// inside a goroutine
	rr.EXPECT().First(gomock.Any()).Return(nil, errors.New("not found")).AnyTimes()
	rp.EXPECT().First(gomock.Any()).Return(nil, errors.New("not found")).AnyTimes()

	v, err := volume.New(root, files, idxkeys, rp, rr, fs, uowFn)
	require.NoError(t, err)

	return ManageVolume{
		Files:          files,
		IDXKeys:        idxkeys,
		Fs:             fs,
		ReplicaPendent: rp,
		ReplicaRetry:   rr,

		V: v,

		ctrl: ctrl,
	}
}

// Finish finishes all the *Ctrl for the 'gomock' at ones
func (mv *ManageVolume) Finish() {
	mv.ctrl.Finish()
	mv.V.Close()
}
