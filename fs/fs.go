package fs

import "github.com/spf13/afero"

// If it fails you need to install the github.com/spf13/afero on your $GOPATH
//go:generate mockgen -destination=../mock/fs.go -mock_names=Fs=Fs -package=mock github.com/xescugc/rebost/fs Fs

// Fs is a wrapper for the afero.Fs package
type Fs interface {
	afero.Fs
}
