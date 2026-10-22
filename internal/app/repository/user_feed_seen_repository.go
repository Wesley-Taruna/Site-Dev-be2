package repository

import (
	"context"
	"time"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/model"

	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

// IUserFeedSeenRepository keeps when a user last opened a feed (November
// scope: Activity Log, approver news).
type IUserFeedSeenRepository interface {
	// Get returns the row (gorm ErrRecordNotFound = never opened).
	Get(ctx context.Context, userID int64, feed string) (seen *model.UserFeedSeen, err error)
	// Set stores seenAt (creates the row the first time).
	Set(ctx context.Context, userID int64, feed string, seenAt time.Time) (err error)
}

type UserFeedSeenRepository struct {
	opt opt.Options
}

func NewUserFeedSeenRepository() IUserFeedSeenRepository {
	return &UserFeedSeenRepository{
		opt: global.Setup.Common,
	}
}

func (r *UserFeedSeenRepository) Get(ctx context.Context, userID int64, feed string) (seen *model.UserFeedSeen, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "UserFeedSeenRepository.Get")
	defer span.End()

	seen = &model.UserFeedSeen{}
	if err = readDB(ctx, r.opt).Where("UserID = ? AND Feed = ?", userID, feed).First(seen).Error; err != nil {
		span.RecordError(err)
		seen = nil
		return
	}

	return
}

func (r *UserFeedSeenRepository) Set(ctx context.Context, userID int64, feed string, seenAt time.Time) (err error) {
	ctx, span := r.opt.Trace.Start(ctx, "UserFeedSeenRepository.Set")
	defer span.End()

	db := writeDB(ctx, r.opt)

	tx := db.Model(new(model.UserFeedSeen)).Where("UserID = ? AND Feed = ?", userID, feed).Update("SeenAt", seenAt)
	if err = tx.Error; err != nil {
		span.RecordError(err)
		return
	}
	if tx.RowsAffected > 0 {
		return
	}
	if err = db.Create(&model.UserFeedSeen{UserID: userID, Feed: feed, SeenAt: seenAt}).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}
