package repository

import (
	"context"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/model"

	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

type ITaskRepository interface {
	// Get lists tasks. Empty filters are skipped: siteIDs nil, status "",
	// createdBy 0.
	Get(ctx context.Context, offset int, limit int, orderBy string, siteIDs []string, status string, createdBy int64) (tasks []*model.Task, count int64, err error)
	// GetBySiteID returns the single task tied to a site (one task per site).
	GetBySiteID(ctx context.Context, siteID string) (task *model.Task, err error)
}

type TaskRepository struct {
	opt opt.Options
}

func NewTaskRepository() ITaskRepository {
	return &TaskRepository{
		opt: global.Setup.Common,
	}
}

func (r *TaskRepository) Get(ctx context.Context, offset int, limit int, orderBy string, siteIDs []string, status string, createdBy int64) (tasks []*model.Task, count int64, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "TaskRepository.Get")
	defer span.End()

	db := readDB(ctx, r.opt)

	if siteIDs != nil {
		db = db.Where("SiteID IN (?)", siteIDs)
	}
	if status != "" {
		db = db.Where("TaskStatus = ?", status)
	}
	if createdBy != 0 {
		db = db.Where("CreatedBy = ?", createdBy)
	}

	if err = db.Model(new(model.Task)).Count(&count).Error; err != nil {
		span.RecordError(err)
		return
	}

	if orderBy != "" {
		db = db.Order(orderBy)
	}
	if limit > 0 {
		db = db.Limit(limit)
	}
	if offset > 0 {
		db = db.Offset(offset)
	}

	if err = db.Find(&tasks).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *TaskRepository) GetBySiteID(ctx context.Context, siteID string) (task *model.Task, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "TaskRepository.GetBySiteID")
	defer span.End()

	db := readDB(ctx, r.opt)

	task = &model.Task{}
	if err = db.Where("SiteID = ?", siteID).First(task).Error; err != nil {
		span.RecordError(err)
		task = nil
		return
	}

	return
}
