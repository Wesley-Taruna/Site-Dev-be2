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

// IApprovalRepository covers ApprovalHDR and ApprovalDET together (they are
// always read and written as one unit).
type IApprovalRepository interface {
	// GetHeaderByTaskID returns the newest approval round of a task (gorm
	// ErrRecordNotFound if the task was never submitted).
	GetHeaderByTaskID(ctx context.Context, taskID string) (header *model.ApprovalHeader, err error)
	GetHeaderByID(ctx context.Context, id uint) (header *model.ApprovalHeader, err error)
	GetDetailByID(ctx context.Context, id uint) (detail *model.ApprovalDetail, err error)
	// GetDetailsByHeaderID returns the levels of a round, lowest level first.
	GetDetailsByHeaderID(ctx context.Context, headerID uint) (details []*model.ApprovalDetail, err error)
	// GetPendingByApprover lists the PENDING rows of an approver; search
	// matches Sites.SiteName1 (LIKE).
	GetPendingByApprover(ctx context.Context, approverID int64, search string, orderBy string, limit int, offset int) (details []*model.ApprovalDetail, count int64, err error)
	// GetActedByApprover lists the rows an approver already decided
	// (APPROVED/REJECTED/REVISION, and HOLD).
	GetActedByApprover(ctx context.Context, approverID int64, search string, orderBy string, limit int, offset int) (details []*model.ApprovalDetail, count int64, err error)
	// GetPendingDetailByApprover is the PENDING level of this approver in a
	// round.
	GetPendingDetailByApprover(ctx context.Context, headerID uint, approverID int64) (detail *model.ApprovalDetail, err error)
	// GetHeldDetailByApprover is the HOLD level of this approver in a round.
	GetHeldDetailByApprover(ctx context.Context, headerID uint, approverID int64) (detail *model.ApprovalDetail, err error)
	// UpdateDetailIfHeld changes a level only while it is still HOLD.
	UpdateDetailIfHeld(ctx context.Context, id uint, updates map[string]interface{}) (rows int64, err error)
	CreateHeader(ctx context.Context, header *model.ApprovalHeader) (err error)
	CreateDetails(ctx context.Context, details []*model.ApprovalDetail) (err error)
	// UpdateHeader / UpdateDetailByID change only the given columns.
	UpdateHeader(ctx context.Context, id uint, updates map[string]interface{}) (err error)
	UpdateDetailByID(ctx context.Context, id uint, updates map[string]interface{}) (err error)
	// UpdateDetailIfPending changes a level only while it is still PENDING
	// (two approvers acting at once); rows 0 = someone acted already.
	UpdateDetailIfPending(ctx context.Context, id uint, updates map[string]interface{}) (rows int64, err error)
	// GetDecisionRolesByTaskIDs lists, oldest decision first, the role of
	// every level of these tasks (all rounds) whose status is one of
	// statuses (November scope: rejected_by_role).
	GetDecisionRolesByTaskIDs(ctx context.Context, taskIDs []string, statuses []string) (rows []model.TaskDecisionRole, err error)
	// GetNewPendingByApprover lists the levels of this approver that are
	// PENDING and became so (UpdatedAt) after since: approver news NEW_TASK.
	GetNewPendingByApprover(ctx context.Context, approverID int64, since time.Time) (rows []model.ApproverNewsRow, err error)
	// GetDivHeadReturnsAfterApproval lists the Div Head REVISION decisions
	// after since in rounds where this approver approved a lower level:
	// approver news DIV_HEAD_RETURNED.
	GetDivHeadReturnsAfterApproval(ctx context.Context, approverID int64, since time.Time) (rows []model.ApproverNewsRow, err error)
}

type ApprovalRepository struct {
	opt opt.Options
}

func NewApprovalRepository() IApprovalRepository {
	return &ApprovalRepository{
		opt: global.Setup.Common,
	}
}

func (r *ApprovalRepository) GetHeaderByTaskID(ctx context.Context, taskID string) (header *model.ApprovalHeader, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "ApprovalRepository.GetHeaderByTaskID")
	defer span.End()

	db := readDB(ctx, r.opt)

	// Limit+Find, not First: First adds its own ORDER BY on the primary key
	// and SQL Server rejects the same column twice in ORDER BY.
	var headers []*model.ApprovalHeader
	if err = db.Where("TaskID = ?", taskID).Order("ID DESC").Limit(1).Find(&headers).Error; err != nil {
		span.RecordError(err)
		return
	}
	if len(headers) == 0 {
		err = gorm.ErrRecordNotFound
		span.RecordError(err)
		return
	}
	header = headers[0]

	return
}

func (r *ApprovalRepository) GetHeaderByID(ctx context.Context, id uint) (header *model.ApprovalHeader, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "ApprovalRepository.GetHeaderByID")
	defer span.End()

	db := readDB(ctx, r.opt)

	header = &model.ApprovalHeader{}
	if err = db.Where("ID = ?", id).First(header).Error; err != nil {
		span.RecordError(err)
		header = nil
		return
	}

	return
}

func (r *ApprovalRepository) GetDetailByID(ctx context.Context, id uint) (detail *model.ApprovalDetail, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "ApprovalRepository.GetDetailByID")
	defer span.End()

	db := readDB(ctx, r.opt)

	detail = &model.ApprovalDetail{}
	if err = db.Where("ID = ?", id).First(detail).Error; err != nil {
		span.RecordError(err)
		detail = nil
		return
	}

	return
}

func (r *ApprovalRepository) GetDetailsByHeaderID(ctx context.Context, headerID uint) (details []*model.ApprovalDetail, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "ApprovalRepository.GetDetailsByHeaderID")
	defer span.End()

	db := readDB(ctx, r.opt)

	if err = db.Where("ApprovalHeaderID = ?", headerID).Order("Level ASC").Find(&details).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *ApprovalRepository) GetPendingByApprover(ctx context.Context, approverID int64, search string, orderBy string, limit int, offset int) (details []*model.ApprovalDetail, count int64, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "ApprovalRepository.GetPendingByApprover")
	defer span.End()

	db := r.approverRows(ctx).
		Where("ApprovalDET.ApproverID = ? AND ApprovalDET.Status = ?", approverID, constants.ApprovalDetailStatusPending)

	details, count, err = r.page(db, search, orderBy, limit, offset)
	if err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *ApprovalRepository) GetActedByApprover(ctx context.Context, approverID int64, search string, orderBy string, limit int, offset int) (details []*model.ApprovalDetail, count int64, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "ApprovalRepository.GetActedByApprover")
	defer span.End()

	db := r.approverRows(ctx).
		Where("ApprovalDET.ApproverID = ? AND ApprovalDET.Status IN ?", approverID, []string{
			constants.ApprovalDetailStatusApproved, constants.ApprovalDetailStatusRejected, constants.ApprovalDetailStatusRevision,
			constants.ApprovalDetailStatusHold, // November scope (API-CHANGES §13.1)
		})

	details, count, err = r.page(db, search, orderBy, limit, offset)
	if err != nil {
		span.RecordError(err)
		return
	}

	return
}

// approverRows joins ApprovalDET through ApprovalHDR and Tasks to Sites, so
// the search can match the site name (ApprovalDET has no site data).
func (r *ApprovalRepository) approverRows(ctx context.Context) *gorm.DB {
	return readDB(ctx, r.opt).
		Table("ApprovalDET").
		Joins("JOIN ApprovalHDR ON ApprovalHDR.ID = ApprovalDET.ApprovalHeaderID").
		Joins("JOIN Tasks ON Tasks.TaskID = ApprovalHDR.TaskID").
		Joins("JOIN Sites ON Sites.SiteID = Tasks.SiteID")
}

// page adds the site-name search, counts, then loads one page.
func (r *ApprovalRepository) page(db *gorm.DB, search string, orderBy string, limit int, offset int) (details []*model.ApprovalDetail, count int64, err error) {
	if search != "" {
		db = db.Where("Sites.SiteName1 LIKE ?", "%"+search+"%")
	}

	if err = db.Session(&gorm.Session{}).Count(&count).Error; err != nil {
		return
	}

	err = db.Select("ApprovalDET.*").Order(orderBy).Limit(limit).Offset(offset).Find(&details).Error
	return
}

func (r *ApprovalRepository) GetPendingDetailByApprover(ctx context.Context, headerID uint, approverID int64) (detail *model.ApprovalDetail, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "ApprovalRepository.GetPendingDetailByApprover")
	defer span.End()

	if detail, err = r.detailByApproverAndStatus(ctx, headerID, approverID, constants.ApprovalDetailStatusPending); err != nil {
		span.RecordError(err)
	}
	return
}

func (r *ApprovalRepository) GetHeldDetailByApprover(ctx context.Context, headerID uint, approverID int64) (detail *model.ApprovalDetail, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "ApprovalRepository.GetHeldDetailByApprover")
	defer span.End()

	if detail, err = r.detailByApproverAndStatus(ctx, headerID, approverID, constants.ApprovalDetailStatusHold); err != nil {
		span.RecordError(err)
	}
	return
}

func (r *ApprovalRepository) detailByApproverAndStatus(ctx context.Context, headerID uint, approverID int64, status string) (detail *model.ApprovalDetail, err error) {
	detail = &model.ApprovalDetail{}
	err = readDB(ctx, r.opt).
		Where("ApprovalHeaderID = ? AND ApproverID = ? AND Status = ?", headerID, approverID, status).
		First(detail).Error
	if err != nil {
		detail = nil
	}
	return
}

func (r *ApprovalRepository) UpdateDetailIfHeld(ctx context.Context, id uint, updates map[string]interface{}) (rows int64, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "ApprovalRepository.UpdateDetailIfHeld")
	defer span.End()

	tx := writeDB(ctx, r.opt).Model(new(model.ApprovalDetail)).
		Where("ID = ? AND Status = ?", id, constants.ApprovalDetailStatusHold).
		Updates(updates)
	rows, err = tx.RowsAffected, tx.Error
	if err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *ApprovalRepository) CreateHeader(ctx context.Context, header *model.ApprovalHeader) (err error) {
	ctx, span := r.opt.Trace.Start(ctx, "ApprovalRepository.CreateHeader")
	defer span.End()

	if err = writeDB(ctx, r.opt).Create(header).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *ApprovalRepository) CreateDetails(ctx context.Context, details []*model.ApprovalDetail) (err error) {
	ctx, span := r.opt.Trace.Start(ctx, "ApprovalRepository.CreateDetails")
	defer span.End()

	if len(details) == 0 {
		return
	}
	if err = writeDB(ctx, r.opt).Create(&details).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *ApprovalRepository) UpdateHeader(ctx context.Context, id uint, updates map[string]interface{}) (err error) {
	ctx, span := r.opt.Trace.Start(ctx, "ApprovalRepository.UpdateHeader")
	defer span.End()

	if len(updates) == 0 {
		return
	}
	if err = writeDB(ctx, r.opt).Model(new(model.ApprovalHeader)).Where("ID = ?", id).Updates(updates).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *ApprovalRepository) UpdateDetailByID(ctx context.Context, id uint, updates map[string]interface{}) (err error) {
	ctx, span := r.opt.Trace.Start(ctx, "ApprovalRepository.UpdateDetailByID")
	defer span.End()

	if len(updates) == 0 {
		return
	}
	if err = writeDB(ctx, r.opt).Model(new(model.ApprovalDetail)).Where("ID = ?", id).Updates(updates).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *ApprovalRepository) UpdateDetailIfPending(ctx context.Context, id uint, updates map[string]interface{}) (rows int64, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "ApprovalRepository.UpdateDetailIfPending")
	defer span.End()

	tx := writeDB(ctx, r.opt).Model(new(model.ApprovalDetail)).
		Where("ID = ? AND Status = ?", id, constants.ApprovalDetailStatusPending).
		Updates(updates)
	rows, err = tx.RowsAffected, tx.Error
	if err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *ApprovalRepository) GetDecisionRolesByTaskIDs(ctx context.Context, taskIDs []string, statuses []string) (rows []model.TaskDecisionRole, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "ApprovalRepository.GetDecisionRolesByTaskIDs")
	defer span.End()

	if len(taskIDs) == 0 {
		return
	}
	err = readDB(ctx, r.opt).
		Table("ApprovalDET").
		Select("ApprovalHDR.TaskID, ApprovalDET.ApproverRoleID, ApprovalDET.ActedAt").
		Joins("JOIN ApprovalHDR ON ApprovalHDR.ID = ApprovalDET.ApprovalHeaderID").
		Where("ApprovalHDR.TaskID IN ? AND ApprovalDET.Status IN ?", taskIDs, statuses).
		Order("ApprovalDET.ActedAt ASC").
		Find(&rows).Error
	if err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *ApprovalRepository) GetNewPendingByApprover(ctx context.Context, approverID int64, since time.Time) (rows []model.ApproverNewsRow, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "ApprovalRepository.GetNewPendingByApprover")
	defer span.End()

	err = r.approverRows(ctx).
		Select("ApprovalHDR.TaskID, Tasks.SiteID, ApprovalDET.UpdatedAt AS At").
		Where("ApprovalDET.ApproverID = ? AND ApprovalDET.Status = ? AND ApprovalDET.UpdatedAt > ?", approverID, constants.ApprovalDetailStatusPending, since).
		Find(&rows).Error
	if err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *ApprovalRepository) GetDivHeadReturnsAfterApproval(ctx context.Context, approverID int64, since time.Time) (rows []model.ApproverNewsRow, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "ApprovalRepository.GetDivHeadReturnsAfterApproval")
	defer span.End()

	approvedBelow := readDB(ctx, r.opt).Table("ApprovalDET AS mine").Select("1").
		Where("mine.ApprovalHeaderID = ApprovalDET.ApprovalHeaderID AND mine.ApproverID = ? AND mine.Status = ? AND mine.Level < ApprovalDET.Level",
			approverID, constants.ApprovalDetailStatusApproved)
	err = r.approverRows(ctx).
		Select("ApprovalHDR.TaskID, Tasks.SiteID, ApprovalDET.ActedAt AS At").
		Where("ApprovalDET.Status = ? AND ApprovalDET.ApproverRoleID = ? AND ApprovalDET.ActedAt > ?", constants.ApprovalDetailStatusRevision, constants.RoleDivHead, since).
		Where("EXISTS (?)", approvedBelow).
		Find(&rows).Error
	if err != nil {
		span.RecordError(err)
		return
	}

	return
}
