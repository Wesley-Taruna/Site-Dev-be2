package repository

import (
	"context"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/model"

	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

type ITaskHistoryRepository interface {
	// GetByTaskID returns a task's history rows, oldest first.
	GetByTaskID(ctx context.Context, taskID string) (histories []*model.TaskHistory, err error)
	// Get lists history rows of every task, with the total count.
	Get(ctx context.Context, offset int, limit int, orderBy string) (histories []*model.TaskHistory, count int64, err error)
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
