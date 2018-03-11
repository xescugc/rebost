package mock

//go:generate mockgen -destination=fs.go -mock_names=Fs=Fs -package=mock github.com/spf13/afero Fs
