package mock

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/rebost/uow"
	"github.com/xescugc/rebost/volume"
)

type ManageVolume struct {
	Files   *FileRepository
	IDXKeys *IDXKeyRepository
	Fs      *Fs

	V volume.Volume

	filesCtrl   *gomock.Controller
	idxKeysCtrl *gomock.Controller
	fsCtrl      *gomock.Controller
	uowCtrl     *gomock.Controller
}

func NewManageVolume(t *testing.T, root string) ManageVolume {
	filesCtrl := gomock.NewController(t)
	idxKeysCtrl := gomock.NewController(t)
	fsCtrl := gomock.NewController(t)
	uowCtrl := gomock.NewController(t)

	files := NewFileRepository(filesCtrl)
	idxkeys := NewIDXKeyRepository(idxKeysCtrl)
	fs := NewFs(fsCtrl)

	uowFn := func(t uow.Type, uowFn uow.UnitOfWorkFn, repositories ...interface{}) error {
		uw := NewUnitOfWork(uowCtrl)
		uw.EXPECT().Files().Return(files).AnyTimes()
		uw.EXPECT().IDXKeys().Return(idxkeys).AnyTimes()
		return uowFn(uw)
	}

	// This first implementation is already tested
	// so we do not need it
	fs.EXPECT().MkdirAll(gomock.Any(), gomock.Any()).Return(nil).Times(2)

	v, err := volume.New(root, files, idxkeys, fs, uowFn)
	require.NoError(t, err)

	fsCtrl.Finish()
	fsCtrl = gomock.NewController(t)

	return ManageVolume{
		Files:   files,
		IDXKeys: idxkeys,
		Fs:      fs,

		V: v,

		filesCtrl:   filesCtrl,
		idxKeysCtrl: idxKeysCtrl,
		fsCtrl:      fsCtrl,
		uowCtrl:     uowCtrl,
	}

}

func (mv *ManageVolume) Finish() {
	mv.filesCtrl.Finish()
	mv.idxKeysCtrl.Finish()
	mv.fsCtrl.Finish()
	mv.uowCtrl.Finish()
}
