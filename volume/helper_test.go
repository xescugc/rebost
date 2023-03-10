package volume_test

import (
	context "context"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/spf13/afero/mem"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/rebost/mock"
	"github.com/xescugc/rebost/state"
	"github.com/xescugc/rebost/uow"
	"github.com/xescugc/rebost/volume"
)

// manageVolume is a test structure that hepls initialize a
// Volume with all his 'mocks' initialized an ready to use
type manageVolume struct {
	Files      *mock.FileRepository
	IDXKeys    *mock.IDXKeyRepository
	IDXVolumes *mock.IDXVolumeRepository
	Fs         *mock.Fs
	Replicas   *mock.ReplicaRepository
	State      *mock.StateRepository

	V volume.Local

	ctrl *gomock.Controller
}

// newManageVolume returns the initialization of the ManageVolume
// with all the mocks
func newManageVolume(t *testing.T, root string) manageVolume {
	ctrl := gomock.NewController(t)

	files := mock.NewFileRepository(ctrl)
	idxkeys := mock.NewIDXKeyRepository(ctrl)
	idxvolumes := mock.NewIDXVolumeRepository(ctrl)
	fs := mock.NewFs(ctrl)
	rp := mock.NewReplicaRepository(ctrl)
	sr := mock.NewStateRepository(ctrl)

	uowFn := func(ctx context.Context, t uow.Type, uowFn uow.UnitOfWorkFn, repositories ...interface{}) error {
		uw := mock.NewUnitOfWork(ctrl)
		uw.EXPECT().Files().Return(files).AnyTimes()
		uw.EXPECT().IDXKeys().Return(idxkeys).AnyTimes()
		uw.EXPECT().IDXVolumes().Return(idxvolumes).AnyTimes()
		uw.EXPECT().Fs().Return(fs).AnyTimes()
		uw.EXPECT().Replicas().Return(rp).AnyTimes()
		uw.EXPECT().State().Return(sr).AnyTimes()
		return uowFn(ctx, uw)
	}

	// This first implementation is already tested
	// so we do not need it
	fs.EXPECT().MkdirAll(gomock.Any(), gomock.Any()).Return(nil).Times(2)
	fs.EXPECT().Stat(gomock.Any()).Return(nil, os.ErrNotExist)
	fs.EXPECT().Create(gomock.Any()).Return(mem.NewFileHandle(mem.CreateFile("")), nil)

	sr.EXPECT().Find(gomock.Any(), gomock.Any()).Return(&state.State{}, nil)
	sr.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	v, err := volume.New(root, files, idxkeys, idxvolumes, rp, sr, fs, uowFn)
	require.NoError(t, err)

	return manageVolume{
		Files:      files,
		IDXKeys:    idxkeys,
		IDXVolumes: idxvolumes,
		Fs:         fs,
		Replicas:   rp,
		State:      sr,

		V: v,

		ctrl: ctrl,
	}
}

// expectUpdateState is a helper for the case in which the state is updated with a new size
func expectUpdateState(t *testing.T, mv manageVolume, ctx context.Context, size int) {
	t.Helper()

	dbs := state.State{
		SystemTotalSize: 2000,
		SystemUsedSize:  100,
		VolumeTotalSize: 1000,
		VolumeUsedSize:  100,
	}

	us := state.State{
		SystemTotalSize: 2000,
		SystemUsedSize:  dbs.SystemUsedSize + size,
		VolumeTotalSize: 1000,
		VolumeUsedSize:  dbs.VolumeUsedSize + size,
	}

	vid := mv.V.ID()

	mv.State.EXPECT().Find(ctx, vid).Return(&dbs, nil)
	mv.State.EXPECT().Update(ctx, vid, &us).Return(nil)
}

// Finish finishes all the *Ctrl for the 'gomock' at ones
func (mv *manageVolume) Finish() {
	mv.ctrl.Finish()
	mv.V.Close()
}
