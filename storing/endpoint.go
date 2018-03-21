package storing

import (
	"context"
	"io"

	"github.com/go-kit/kit/endpoint"
)

type createFileRequest struct {
	Key  string
	Body io.Reader
}

type createFileResponse struct {
	Err error
}

func (r createFileResponse) error() error { return r.Err }

func makeCreateFileEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createFileRequest)
		err := s.CreateFile(ctx, req.Key, req.Body)
		return createFileResponse{Err: err}, nil
	}
}

type getFileRequest struct {
	Key string
}

type getFileResponse struct {
	IOR io.Reader
	Err error
}

func (r getFileResponse) error() error { return r.Err }

func makeGetFileEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getFileRequest)
		ior, err := s.GetFile(ctx, req.Key)
		return getFileResponse{IOR: ior, Err: err}, nil
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
	Ok  bool
	Err error
}

func (r hasFileResponse) error() error { return r.Err }

func makeHasFileEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(hasFileRequest)
		ok, err := s.HasFile(ctx, req.Key)
		return hasFileResponse{Ok: ok, Err: err}, nil
	}
}
