package idxvolume

import "context"

//go:generate mockgen -destination=../mock/idxvolume_repository.go -mock_names=Repository=IDXVolumeRepository -package=mock github.com/xescugc/rebost/idxvolume Repository

// Repository is the interface that has to be fulfiled to interact with IDXVolume
type Repository interface {
	CreateOrReplace(ctx context.Context, ik *IDXVolume) error
	FindByVolumeID(ctx context.Context, volumeID string) (*IDXVolume, error)
	DeleteByKey(ctx context.Context, volumeID string) error
}
