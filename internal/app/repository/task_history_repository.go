package repository

import (
	"context"
	"time"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/constants"
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
	// GetFeedByApprover lists, newest first, the history rows with one of
	// actions on every task sent at least once (has an approval round; any
	// staff, any approval chain): the Dept/Div Head Activity Log (Wes, 23
	// Oct). actorID != 0 keeps only that actor's rows; waitingOnly keeps
	// only tasks now waiting for or held by approverID; limit 0 = all rows.
	GetFeedByApprover(ctx context.Context, approverID int64, actions []string, actorID int64, waitingOnly bool, offset int, limit int) (histories []*model.TaskHistory, count int64, err error)
	// CountFeedByApproverSince counts those rows newer than since and not by
	// approverID (the approver's unread badge).
	CountFeedByApproverSince(ctx context.Context, approverID int64, actions []string, since time.Time) (count int64, err error)
	// GetTaskIDsWaitingFor is those of taskIDs now waiting for or held by
	// approverID (his level PENDING or HOLD).
	GetTaskIDsWaitingFor(ctx context.Context, approverID int64, taskIDs []string) (ids []string, err error)
	// GetByTaskIDsAndActor lists the rows of action by actorID on taskIDs.
	GetByTaskIDsAndActor(ctx context.Context, taskIDs []string, actorID int64, action string) (histories []*model.TaskHistory, err error)
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

// approverFeedRows is the approver Activity Log base: history rows joined
// to their task, on every task sent at least once (it has an ApprovalHDR
// round; drafts never sent have none). Wes, 23 Oct: all sent requests,
// not only those where the approver is in the chain.
func (r *TaskHistoryRepository) approverFeedRows(ctx context.Context, actions []string) *gorm.DB {
	sent := readDB(ctx, r.opt).Table("ApprovalHDR").Select("ApprovalHDR.TaskID")
	return readDB(ctx, r.opt).
		Table("TaskHistories").
		Joins("JOIN Tasks ON Tasks.TaskID = TaskHistories.TaskID").
		Where("TaskHistories.TaskID IN (?) AND TaskHistories.Action IN ?", sent, actions)
}

// waitingFor is the TaskIDs whose level of approverID is PENDING or HOLD
// (only the live round has such levels).
func (r *TaskHistoryRepository) waitingFor(ctx context.Context, approverID int64) *gorm.DB {
	return readDB(ctx, r.opt).Table("ApprovalDET").Select("ApprovalHDR.TaskID").
		Joins("JOIN ApprovalHDR ON ApprovalHDR.ID = ApprovalDET.ApprovalHeaderID").
		Where("ApprovalDET.ApproverID = ? AND ApprovalDET.Status IN ?", approverID, []string{constants.ApprovalDetailStatusPending, constants.ApprovalDetailStatusHold})
}

func (r *TaskHistoryRepository) GetFeedByApprover(ctx context.Context, approverID int64, actions []string, actorID int64, waitingOnly bool, offset int, limit int) (histories []*model.TaskHistory, count int64, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "TaskHistoryRepository.GetFeedByApprover")
	defer span.End()

	db := r.approverFeedRows(ctx, actions)
	if actorID != 0 {
		db = db.Where("TaskHistories.ActorID = ?", actorID)
	}
	if waitingOnly {
		db = db.Where("TaskHistories.TaskID IN (?)", r.waitingFor(ctx, approverID))
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

func (r *TaskHistoryRepository) CountFeedByApproverSince(ctx context.Context, approverID int64, actions []string, since time.Time) (count int64, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "TaskHistoryRepository.CountFeedByApproverSince")
	defer span.End()

	err = r.approverFeedRows(ctx, actions).
		Where("TaskHistories.ActorID <> ? AND TaskHistories.CreatedAt > ?", approverID, since).
		Count(&count).Error
	if err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *TaskHistoryRepository) GetTaskIDsWaitingFor(ctx context.Context, approverID int64, taskIDs []string) (ids []string, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "TaskHistoryRepository.GetTaskIDsWaitingFor")
	defer span.End()

	if len(taskIDs) == 0 {
		return
	}
	if err = r.waitingFor(ctx, approverID).Where("ApprovalHDR.TaskID IN ?", taskIDs).Pluck("ApprovalHDR.TaskID", &ids).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *TaskHistoryRepository) GetByTaskIDsAndActor(ctx context.Context, taskIDs []string, actorID int64, action string) (histories []*model.TaskHistory, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "TaskHistoryRepository.GetByTaskIDsAndActor")
	defer span.End()

	if len(taskIDs) == 0 {
		return
	}
	err = readDB(ctx, r.opt).
		Where("TaskID IN ? AND ActorID = ? AND Action = ?", taskIDs, actorID, action).
		Find(&histories).Error
	if err != nil {
		span.RecordError(err)
		return
	}

	return
}
