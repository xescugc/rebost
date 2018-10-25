package mock

import (
	context "context"
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
	rp := NewReplicaPendentRepository(ctrl)

	uowFn := func(ctx context.Context, t uow.Type, uowFn uow.UnitOfWorkFn, repositories ...interface{}) error {
		uw := NewUnitOfWork(ctrl)
		uw.EXPECT().Files().Return(files).AnyTimes()
		uw.EXPECT().IDXKeys().Return(idxkeys).AnyTimes()
		uw.EXPECT().Fs().Return(fs).AnyTimes()
		uw.EXPECT().ReplicaPendent().Return(rp).AnyTimes()
		return uowFn(uw)
	}

	// This first implementation is already tested
	// so we do not need it
	fs.EXPECT().MkdirAll(gomock.Any(), gomock.Any()).Return(nil).Times(2)
	fs.EXPECT().Stat(gomock.Any()).Return(nil, os.ErrNotExist)
	fs.EXPECT().Create(gomock.Any()).Return(mem.NewFileHandle(mem.CreateFile("")), nil)

	v, err := volume.New(root, files, idxkeys, rp, fs, uowFn)
	require.NoError(t, err)

	return ManageVolume{
		Files:          files,
		IDXKeys:        idxkeys,
		Fs:             fs,
		ReplicaPendent: rp,

		V: v,

		ctrl: ctrl,
	}
}

// Finish finishes all the *Ctrl for the 'gomock' at ones
func (mv *ManageVolume) Finish() {
	mv.ctrl.Finish()
}
