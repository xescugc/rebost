package storing

import (
	"context"
	"io"

	"github.com/go-kit/kit/endpoint"
	"github.com/xescugc/rebost/storing/model"
)

type createFileRequest struct {
	Key     string
	Body    io.ReadCloser
	Replica int
}

type createFileResponse struct {
	Err error
}

func (r createFileResponse) error() error { return r.Err }

func makeCreateFileEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createFileRequest)
		err := s.CreateFile(ctx, req.Key, req.Body, req.Replica)
		return createFileResponse{Err: err}, nil
	}
}

type getFileRequest struct {
	Key string
}

type getFileResponse struct {
	IORC io.ReadCloser
	Err  error
}

func (r getFileResponse) error() error { return r.Err }

func makeGetFileEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getFileRequest)
		iorc, err := s.GetFile(ctx, req.Key)
		return getFileResponse{IORC: iorc, Err: err}, nil
	}
}

type deleteFileRequest struct {
	Key string
}

type deleteFileResponse struct {
	Err error
}

func (r deleteFileResponse) error() error { return r.Err }

func makeDeleteFileEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteFileRequest)
		err := s.DeleteFile(ctx, req.Key)
		return deleteFileResponse{Err: err}, nil
	}
}

type hasFileRequest struct {
	Key string
}

type hasFileResponse struct {
	Ok       bool
	VolumeID string
}

func makeHasFileEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(hasFileRequest)
		vid, ok, err := s.HasFile(ctx, req.Key)
		if err != nil {
			return nil, err
		}
		return hasFileResponse{VolumeID: vid, Ok: ok}, nil
	}
}

type response struct {
	Data interface{} `json:"data,omitempty"`
	Err  error       `json:"error,omitempty"`
}

func (r response) error() error { return r.Err }

func makeGetConfigEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		cfg, err := s.Config(ctx)
		if err != nil {
			return response{Err: err}, nil
		}
		return response{Data: model.ConfigToModel(cfg)}, nil
	}
}

type createReplicaRequest struct {
	Key  string
	Body io.ReadCloser
}

func makeCreateReplicaEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createReplicaRequest)
		volID, err := s.CreateReplica(ctx, req.Key, req.Body)
		if err != nil {
			return response{Err: err}, nil
		}
		return response{Data: model.CreateReplica{VolumeID: volID}}, nil
	}
}

type updateFileReplicaRequest struct {
	Key       string
	Replica   int
	VolumeIDs []string
}

type updateFileReplicaResponse struct {
	Err error
}

func (r updateFileReplicaResponse) error() error { return r.Err }

func makeUpdateFileReplicaEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateFileReplicaRequest)
		err := s.UpdateFileReplica(ctx, req.Key, req.VolumeIDs, req.Replica)
		return updateFileReplicaResponse{Err: err}, nil
	}
}
