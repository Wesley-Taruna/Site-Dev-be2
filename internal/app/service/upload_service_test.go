package service

import (
	"context"
	"strings"
	"testing"

	"site-dev-backend/internal/app/client"
	"site-dev-backend/internal/app/mocks"
	"site-dev-backend/internal/app/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRandomizeFilename(t *testing.T) {
	assert.Regexp(t, `^logo-\d{6}\.png$`, randomizeFilename("logo.png"))
	assert.Regexp(t, `^passwd-\d{6}$`, randomizeFilename("../../etc/passwd"))
}

func TestSplitObjectPath(t *testing.T) {
	b, o, err := splitObjectPath("bucket/a/b.jpg")
	require.NoError(t, err)
	assert.Equal(t, "bucket", b)
	assert.Equal(t, "a/b.jpg", o)
	for _, bad := range []string{"", "bucket", "/x", "bucket/"} {
		_, _, err = splitObjectPath(bad)
		assert.Error(t, err, bad)
	}
}

func TestUploadServiceAdd_StorageErrorIsWrapped(t *testing.T) {
	s := &UploadService{opt: testOptions(), ClientStorage: client.NewStorageClient(nil), tempBucket: "b", tempPrefix: "p"}

	_, err := s.Add(context.Background(), "x.jpg", strings.NewReader("x"))

	assert.ErrorIs(t, err, client.ErrStorageNotSetUp)
	assert.Contains(t, err.Error(), "write to gcs: ")
}

func TestUploadServiceOpenAttachment_UnknownAttachment(t *testing.T) {
	sites := mocks.NewISiteRepository(t)
	s := &UploadService{opt: testOptions(), RepositorySite: sites, ClientStorage: mocks.NewIStorageClient(t)}
	sites.On("GetAttachmentByID", ctxAny, uint(9)).Return(nil, gorm.ErrRecordNotFound)

	a, r, err := s.OpenAttachment(context.Background(), 9)

	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	assert.Nil(t, a)
	assert.Nil(t, r)
}

func TestUploadServiceOpenAttachment_BadPathKeepsAttachment(t *testing.T) {
	sites := mocks.NewISiteRepository(t)
	s := &UploadService{opt: testOptions(), RepositorySite: sites, ClientStorage: mocks.NewIStorageClient(t)}
	sites.On("GetAttachmentByID", ctxAny, uint(9)).Return(&model.SiteAttachment{Path: "nopath"}, nil)

	a, _, err := s.OpenAttachment(context.Background(), 9)

	assert.Error(t, err)
	assert.NotNil(t, a, "handler answers 'attachment file' not found")
}
