package repository

import (
	"context"
	"time"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/model"

	"gitlab.com/fami-app-v2/familib/familabs/opt"
	"gorm.io/gorm"
)

type ITaskHistoryRepository interface {
	// GetByTaskID returns a task's history rows, oldest first.
	GetByTaskID(ctx context.Context, taskID string) (histories []*model.TaskHistory, err error)
	// Get lists history rows of every task, with the total count.
	Get(ctx context.Context, offset int, limit int, orderBy string) (histories []*model.TaskHistory, count int64, err error)
	// GetFeedByCreator lists, newest first, the history rows with one of
	// actions on the tasks created by creatorID (Activity Log). notActorID
	// != 0 leaves out that actor's rows; taskStatus != "" keeps only tasks
	// now in that status; limit 0 = all rows.
	GetFeedByCreator(ctx context.Context, creatorID int64, actions []string, notActorID int64, taskStatus string, offset int, limit int) (histories []*model.TaskHistory, count int64, err error)
	// CountFeedSince counts those rows newer than since and not by creatorID
	// (the unread badge).
	CountFeedSince(ctx context.Context, creatorID int64, actions []string, since time.Time) (count int64, err error)
	// Create appends a history row (rows are never changed).
	Create(ctx context.Context, history *model.TaskHistory) (err error)
}

type TaskHistoryRepository struct {
	opt opt.Options
}

func NewTaskHistoryRepository() ITaskHistoryRepository {
	return &TaskHistoryRepository{
		opt: global.Setup.Common,
	}
}

func (r *TaskHistoryRepository) GetByTaskID(ctx context.Context, taskID string) (histories []*model.TaskHistory, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "TaskHistoryRepository.GetByTaskID")
	defer span.End()

	db := readDB(ctx, r.opt)

	if err = db.Where("TaskID = ?", taskID).Order("CreatedAt ASC").Find(&histories).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *TaskHistoryRepository) Get(ctx context.Context, offset int, limit int, orderBy string) (histories []*model.TaskHistory, count int64, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "TaskHistoryRepository.Get")
	defer span.End()

	db := readDB(ctx, r.opt)

	if err = db.Model(new(model.TaskHistory)).Count(&count).Error; err != nil {
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

	if err = db.Find(&histories).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *TaskHistoryRepository) Create(ctx context.Context, history *model.TaskHistory) (err error) {
	ctx, span := r.opt.Trace.Start(ctx, "TaskHistoryRepository.Create")
	defer span.End()

	db := writeDB(ctx, r.opt)

	if err = db.Create(history).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}

// feedRows is the Activity Log base: history rows joined to their task.
func (r *TaskHistoryRepository) feedRows(ctx context.Context, creatorID int64, actions []string) *gorm.DB {
	return readDB(ctx, r.opt).
		Table("TaskHistories").
		Joins("JOIN Tasks ON Tasks.TaskID = TaskHistories.TaskID").
		Where("Tasks.CreatedBy = ? AND TaskHistories.Action IN ?", creatorID, actions)
}

func (r *TaskHistoryRepository) GetFeedByCreator(ctx context.Context, creatorID int64, actions []string, notActorID int64, taskStatus string, offset int, limit int) (histories []*model.TaskHistory, count int64, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "TaskHistoryRepository.GetFeedByCreator")
	defer span.End()

	db := r.feedRows(ctx, creatorID, actions)
	if notActorID != 0 {
		db = db.Where("TaskHistories.ActorID <> ?", notActorID)
	}
	if taskStatus != "" {
		db = db.Where("Tasks.TaskStatus = ?", taskStatus)
	}

	if err = db.Session(&gorm.Session{}).Count(&count).Error; err != nil {
		span.RecordError(err)
		return
	}

	db = db.Select("TaskHistories.*").Order("TaskHistories.CreatedAt DESC, TaskHistories.ID DESC")
	if limit > 0 {
		db = db.Limit(limit)
	}
	if offset > 0 {
		db = db.Offset(offset)
	}
	if err = db.Find(&histories).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *TaskHistoryRepository) CountFeedSince(ctx context.Context, creatorID int64, actions []string, since time.Time) (count int64, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "TaskHistoryRepository.CountFeedSince")
	defer span.End()

	err = r.feedRows(ctx, creatorID, actions).
		Where("TaskHistories.ActorID <> ? AND TaskHistories.CreatedAt > ?", creatorID, since).
		Count(&count).Error
	if err != nil {
		span.RecordError(err)
		return
	}

	return
}
