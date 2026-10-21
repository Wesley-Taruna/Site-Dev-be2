package repository

import (
	"context"

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
	// GetDetailsByHeaderID returns the levels of a round, lowest level first.
	GetDetailsByHeaderID(ctx context.Context, headerID uint) (details []*model.ApprovalDetail, err error)
	// GetPendingByApprover lists the PENDING rows of an approver; search
	// matches Sites.SiteName1 (LIKE).
	GetPendingByApprover(ctx context.Context, approverID int64, search string, orderBy string, limit int, offset int) (details []*model.ApprovalDetail, count int64, err error)
	// GetActedByApprover lists the rows an approver already decided
	// (APPROVED/REJECTED/REVISION).
	GetActedByApprover(ctx context.Context, approverID int64, search string, orderBy string, limit int, offset int) (details []*model.ApprovalDetail, count int64, err error)
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
