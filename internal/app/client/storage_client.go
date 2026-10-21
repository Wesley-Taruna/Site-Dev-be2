package client

import (
	"context"
	"errors"
	"io"

	"gitlab.com/fami-app-v2/familib/familabs/env"
)

// ErrStorageNotSetUp is returned until the real file storage client exists.
var ErrStorageNotSetUp = errors.New("storage: Google Cloud Storage client is not set up yet (TODO, plan Q4)")

// IStorageClient stores the site photos. Objects are addressed by bucket +
// object name; the services keep the "<bucket>/<object...>" path format of
// site-dev-api.
type IStorageClient interface {
	Write(ctx context.Context, bucket string, object string, content io.Reader) (err error)
	Open(ctx context.Context, bucket string, object string) (reader io.ReadCloser, err error)
	Delete(ctx context.Context, bucket string, object string) (err error)
}

// NewStorageClient returns the file storage client.
//
// TODO(plan Q4, needs Wes's OK): site-dev-api writes to Google Cloud Storage
// with cloud.google.com/go/storage, which is not in the boilerplate's
// library list (familabs v1.0.2 "storage" is S3/MinIO). Until the library or
// another storage is approved, every call answers ErrStorageNotSetUp.
func NewStorageClient(_ env.Provider) IStorageClient {
	return pendingStorageClient{}
}

type pendingStorageClient struct{}

func (pendingStorageClient) Write(context.Context, string, string, io.Reader) error {
	return ErrStorageNotSetUp
}

func (pendingStorageClient) Open(context.Context, string, string) (io.ReadCloser, error) {
	return nil, ErrStorageNotSetUp
}

func (pendingStorageClient) Delete(context.Context, string, string) error {
	return ErrStorageNotSetUp
}
