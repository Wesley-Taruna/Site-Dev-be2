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

	db := r.opt.Database.Read.WithContext(ctx)

	if err = db.Where("TaskID = ?", taskID).Order("CreatedAt ASC").Find(&histories).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}
