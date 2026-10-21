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
	GetByID(ctx context.Context, taskID string) (task *model.Task, err error)
	// GetByIDs loads the tasks for a set of TaskIDs.
	GetByIDs(ctx context.Context, taskIDs []string) (tasks []*model.Task, err error)
	// GetByStatuses lists tasks whose status is one of statuses (siteIDs
	// nil = no site filter), with the total count.
	GetByStatuses(ctx context.Context, offset int, limit int, orderBy string, statuses []string, siteIDs []string) (tasks []*model.Task, count int64, err error)
	// CountByStatus counts tasks per TaskStatus (createdBy 0 = all tasks).
	CountByStatus(ctx context.Context, createdBy int64) (counts []model.TaskStatusCount, err error)
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

func (r *TaskRepository) GetByID(ctx context.Context, taskID string) (task *model.Task, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "TaskRepository.GetByID")
	defer span.End()

	db := readDB(ctx, r.opt)

	task = &model.Task{}
	if err = db.Where("TaskID = ?", taskID).First(task).Error; err != nil {
		span.RecordError(err)
		task = nil
		return
	}

	return
}

func (r *TaskRepository) GetByIDs(ctx context.Context, taskIDs []string) (tasks []*model.Task, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "TaskRepository.GetByIDs")
	defer span.End()

	if len(taskIDs) == 0 {
		return
	}

	db := readDB(ctx, r.opt)

	if err = db.Where("TaskID IN (?)", taskIDs).Find(&tasks).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *TaskRepository) GetByStatuses(ctx context.Context, offset int, limit int, orderBy string, statuses []string, siteIDs []string) (tasks []*model.Task, count int64, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "TaskRepository.GetByStatuses")
	defer span.End()

	db := readDB(ctx, r.opt).Where("TaskStatus IN (?)", statuses)

	if siteIDs != nil {
		db = db.Where("SiteID IN (?)", siteIDs)
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

func (r *TaskRepository) CountByStatus(ctx context.Context, createdBy int64) (counts []model.TaskStatusCount, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "TaskRepository.CountByStatus")
	defer span.End()

	db := readDB(ctx, r.opt).Model(new(model.Task))

	if createdBy != 0 {
		db = db.Where("CreatedBy = ?", createdBy)
	}

	if err = db.Select("TaskStatus, COUNT(1) as Count").Group("TaskStatus").Scan(&counts).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}
