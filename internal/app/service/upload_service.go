package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"path/filepath"
	"strings"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/client"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/model"
	"site-dev-backend/internal/app/repository"

	"gitlab.com/fami-app-v2/familib/familabs/log"
	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

type IUploadService interface {
	// Add stores an uploaded file in the temp bucket under a randomized
	// name (POST /v1/upload/add).
	Add(ctx context.Context, fileName string, content io.Reader) (res dto.UploadResponse, err error)
	// Remove deletes "<bucket>/<object...>" (POST /v1/upload/remove).
	Remove(ctx context.Context, path string) (err error)
	// OpenAttachment opens the file of an attachment for streaming (GET
	// /v1/attachment/:id/file); the bucket is private.
	OpenAttachment(ctx context.Context, attachmentID uint) (attachment *model.SiteAttachment, reader io.ReadCloser, err error)
}

type UploadService struct {
	opt            opt.Options
	RepositorySite repository.ISiteRepository
	ClientStorage  client.IStorageClient
	tempBucket     string
	tempPrefix     string
}

func NewUploadService() IUploadService {
	env := global.Setup.Common.Env
	return &UploadService{
		opt:            global.Setup.Common,
		RepositorySite: repository.NewSiteRepository(),
		ClientStorage:  global.Setup.Clients.Storage,
		tempBucket:     env.GetString("gcs.upload.temp_bucket"),
		tempPrefix:     env.GetString("gcs.upload.temp_prefix"),
	}
}

func (s *UploadService) Add(ctx context.Context, fileName string, content io.Reader) (res dto.UploadResponse, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "UploadService.Add")
	defer span.End()

	name := randomizeFilename(fileName)
	object := fmt.Sprintf("%s/%s", s.tempPrefix, name)

	if err = s.ClientStorage.Write(ctx, s.tempBucket, object, content); err != nil {
		err = fmt.Errorf("write to gcs: %w", err)
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	res = dto.UploadResponse{FileName: name, Path: fmt.Sprintf("%s/%s", s.tempBucket, object)}

	return
}

func (s *UploadService) Remove(ctx context.Context, path string) (err error) {
	ctx, span := s.opt.Trace.Start(ctx, "UploadService.Remove")
	defer span.End()

	var bucket, object string
	if bucket, object, err = splitObjectPath(path); err != nil {
		span.RecordError(err)
		return
	}
	if err = s.ClientStorage.Delete(ctx, bucket, object); err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	return
}

func (s *UploadService) OpenAttachment(ctx context.Context, attachmentID uint) (attachment *model.SiteAttachment, reader io.ReadCloser, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "UploadService.OpenAttachment")
	defer span.End()

	attachment, err = s.RepositorySite.GetAttachmentByID(ctx, attachmentID)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	var bucket, object string
	if bucket, object, err = splitObjectPath(attachment.Path); err == nil {
		reader, err = s.ClientStorage.Open(ctx, bucket, object)
	}
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	return
}

// splitObjectPath splits "<bucket>/<object...>".
func splitObjectPath(path string) (bucket string, object string, err error) {
	bucket, object, ok := strings.Cut(path, "/")
	if !ok || bucket == "" || object == "" {
		return "", "", fmt.Errorf("invalid object path: %q", path)
	}
	return bucket, object, nil
}

// randomizeFilename turns "logo.png" into "logo-482910.png" (6 random
// digits), dropping any folder part of the original name.
func randomizeFilename(original string) string {
	base := filepath.Base(original)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("%s-%d%s", name, n.Int64()+100000, ext)
}
