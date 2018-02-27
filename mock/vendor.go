package mock

// If it fails you need to install the github.com/spf13/afero on your $GOPATH
//go:generate mockgen -destination=fs.go -mock_names=Fs=Fs -package=mock github.com/spf13/afero Fs
