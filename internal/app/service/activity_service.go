package service

import (
	"context"
	"errors"
	"sort"
	"time"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/constants"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/model"
	"site-dev-backend/internal/app/repository"

	"gitlab.com/fami-app-v2/familib/familabs/log"
	"gitlab.com/fami-app-v2/familib/familabs/opt"
	"gorm.io/gorm"
)

// IActivityService is the staff's Activity Log (GET /v1/activity,
// /unread-count, POST /seen) and the approver's "Baru" news (GET
// /v1/task/approver-news, POST .../seen). November scope, API-CHANGES
// §13.4 and prototype 21 Oct: the Activity Log is for site staff only,
// approvers get the news instead; other roles get empty answers.
type IActivityService interface {
	Get(ctx context.Context, req dto.ActivityRequestGet, userID int64) (res []dto.ActivityResponseGet, total int64, err error)
	UnreadCount(ctx context.Context, userID int64) (res dto.UnreadCountResponse, err error)
	MarkSeen(ctx context.Context, userID int64) (err error)
	GetApproverNews(ctx context.Context, userID int64) (res []dto.ApproverNewsResponseGet, err error)
	MarkApproverNewsSeen(ctx context.Context, userID int64) (err error)
}

type ActivityService struct {
	opt                    opt.Options
	RepositoryTaskHistory  repository.ITaskHistoryRepository
	RepositoryTask         repository.ITaskRepository
	RepositorySite         repository.ISiteRepository
	RepositoryUser         repository.IUserRepository
	RepositoryApproval     repository.IApprovalRepository
	RepositoryUserFeedSeen repository.IUserFeedSeenRepository
	// now is time.Now (tests fix it)
	now func() time.Time
}

func NewActivityService() IActivityService {
	return &ActivityService{
		opt:                    global.Setup.Common,
		RepositoryTaskHistory:  repository.NewTaskHistoryRepository(),
		RepositoryTask:         repository.NewTaskRepository(),
		RepositorySite:         repository.NewSiteRepository(),
		RepositoryUser:         repository.NewUserRepository(),
		RepositoryApproval:     repository.NewApprovalRepository(),
		RepositoryUserFeedSeen: repository.NewUserFeedSeenRepository(),
		now:                    time.Now,
	}
}

func (s *ActivityService) fail(err error) {
	s.opt.Logger.AddMessage(log.ErrorLevel, err)
}

// seenSince is when the user last opened feed; never opened = the last
// FeedNeverSeenWindowHours (default, as the prototype).
func (s *ActivityService) seenSince(ctx context.Context, userID int64, feed string) (time.Time, error) {
	seen, err := s.RepositoryUserFeedSeen.Get(ctx, userID, feed)
	if err == nil {
		return seen.SeenAt, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.now().Add(-constants.FeedNeverSeenWindowHours * time.Hour), nil
	}
	return time.Time{}, err
}

func (s *ActivityService) Get(ctx context.Context, req dto.ActivityRequestGet, userID int64) (res []dto.ActivityResponseGet, total int64, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "ActivityService.Get")
	defer span.End()

	res, total, err = s.get(ctx, req, userID)
	if err != nil {
		span.RecordError(err)
		s.fail(err)
		res, total = nil, 0
	}
	return
}

func (s *ActivityService) get(ctx context.Context, req dto.ActivityRequestGet, userID int64) (res []dto.ActivityResponseGet, total int64, err error) {
	res = []dto.ActivityResponseGet{}
	user, err := getUser(ctx, s.RepositoryUser, userID)
	if err != nil {
		return
	}
	if user.RoleID != constants.RoleUser {
		return // staff only (Wes, 21 Oct)
	}

	since, err := s.seenSince(ctx, userID, constants.FeedActivity)
	if err != nil {
		return
	}

	var histories []*model.TaskHistory
	switch req.Filter {
	case constants.ActivityFilterAction:
		// newest row per request that now needs the staff (REVISION)
		var all []*model.TaskHistory
		if all, _, err = s.RepositoryTaskHistory.GetFeedByCreator(ctx, userID, constants.ActivityFeedActions, 0, constants.TaskStatusRevision, 0, 0); err != nil {
			return
		}
		seenTask := map[string]bool{}
		for _, h := range all {
			if !seenTask[h.TaskID] {
				seenTask[h.TaskID] = true
				histories = append(histories, h)
			}
		}
		total = int64(len(histories))
		histories = pageSlice(histories, req.Offset, req.Limit)
	case constants.ActivityFilterMine:
		// the approvers' decisions on my requests
		histories, total, err = s.RepositoryTaskHistory.GetFeedByCreator(ctx, userID, constants.ActivityFeedActions, userID, "", req.Offset, req.Limit)
	default:
		histories, total, err = s.RepositoryTaskHistory.GetFeedByCreator(ctx, userID, constants.ActivityFeedActions, 0, "", req.Offset, req.Limit)
	}
	if err != nil || len(histories) == 0 {
		return
	}

	return s.rows(ctx, histories, userID, since)
}

// pageSlice is one page of rows.
func pageSlice[T any](rows []T, offset int, limit int) []T {
	if offset >= len(rows) {
		return nil
	}
	rows = rows[offset:]
	if limit > 0 && limit < len(rows) {
		rows = rows[:limit]
	}
	return rows
}

// rows adds task status, site name, actor name and role (one query for the
// tasks, one for the sites, one per actor).
func (s *ActivityService) rows(ctx context.Context, histories []*model.TaskHistory, userID int64, since time.Time) (res []dto.ActivityResponseGet, total int64, err error) {
	taskIDs := make([]string, 0, len(histories))
	for _, h := range histories {
		taskIDs = append(taskIDs, h.TaskID)
	}
	tasks, err := s.RepositoryTask.GetByIDs(ctx, taskIDs)
	if err != nil {
		return
	}
	taskByID := make(map[string]*model.Task, len(tasks))
	siteIDs := make([]string, 0, len(tasks))
	for _, t := range tasks {
		taskByID[t.TaskID] = t
		siteIDs = append(siteIDs, t.SiteID)
	}
	sites, err := s.RepositorySite.GetByIDs(ctx, siteIDs)
	if err != nil {
		return
	}
	siteByID := make(map[string]*model.Site, len(sites))
	for _, site := range sites {
		siteByID[site.SiteID] = site
	}

	actors := map[int64]*model.User{}
	res = make([]dto.ActivityResponseGet, 0, len(histories))
	for _, h := range histories {
		actor, known := actors[h.ActorID]
		if !known {
			actor, _ = s.RepositoryUser.GetByID(ctx, h.ActorID)
			actors[h.ActorID] = actor
		}
		row := dto.ActivityResponseGet{
			ID:        h.ID,
			TaskID:    h.TaskID,
			Action:    h.Action,
			ActorID:   h.ActorID,
			CreatedAt: h.CreatedAt,
			Round:     1,
			Unread:    h.ActorID != userID && h.CreatedAt.After(since),
		}
		if h.Round != nil {
			row.Round = *h.Round
		}
		if actor != nil {
			row.ActorName = actor.Username
			row.ActorRole = actor.RoleID // rows written before ActorRole existed: role now
		}
		if h.ActorRole != nil {
			row.ActorRole = *h.ActorRole
		}
		if t := taskByID[h.TaskID]; t != nil {
			row.SiteID = t.SiteID
			row.TaskStatus = t.TaskStatus
			row.NeedsAction = t.TaskStatus == constants.TaskStatusRevision
			if site := siteByID[t.SiteID]; site != nil {
				row.SiteName1 = site.SiteName1
			}
		}
		res = append(res, row)
	}
	return
}

func (s *ActivityService) UnreadCount(ctx context.Context, userID int64) (res dto.UnreadCountResponse, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "ActivityService.UnreadCount")
	defer span.End()

	user, err := getUser(ctx, s.RepositoryUser, userID)
	if err != nil || user.RoleID != constants.RoleUser {
		return
	}
	since, err := s.seenSince(ctx, userID, constants.FeedActivity)
	if err == nil {
		res.Count, err = s.RepositoryTaskHistory.CountFeedSince(ctx, userID, constants.ActivityFeedActions, since)
	}
	if err != nil {
		span.RecordError(err)
		s.fail(err)
		res = dto.UnreadCountResponse{}
	}
	return
}

func (s *ActivityService) MarkSeen(ctx context.Context, userID int64) (err error) {
	return s.markSeen(ctx, userID, constants.FeedActivity, "ActivityService.MarkSeen")
}

func (s *ActivityService) MarkApproverNewsSeen(ctx context.Context, userID int64) (err error) {
	return s.markSeen(ctx, userID, constants.FeedApproverNews, "ActivityService.MarkApproverNewsSeen")
}

func (s *ActivityService) markSeen(ctx context.Context, userID int64, feed string, spanName string) (err error) {
	ctx, span := s.opt.Trace.Start(ctx, spanName)
	defer span.End()

	if _, err = getUser(ctx, s.RepositoryUser, userID); err == nil {
		err = s.RepositoryUserFeedSeen.Set(ctx, userID, feed, s.now())
	}
	if err != nil {
		span.RecordError(err)
		s.fail(err)
	}
	return
}

func (s *ActivityService) GetApproverNews(ctx context.Context, userID int64) (res []dto.ApproverNewsResponseGet, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "ActivityService.GetApproverNews")
	defer span.End()

	res, err = s.approverNews(ctx, userID)
	if err != nil {
		span.RecordError(err)
		s.fail(err)
		res = nil
	}
	return
}

func (s *ActivityService) approverNews(ctx context.Context, userID int64) (res []dto.ApproverNewsResponseGet, err error) {
	res = []dto.ApproverNewsResponseGet{}
	user, err := getUser(ctx, s.RepositoryUser, userID)
	if err != nil || user.RoleID == constants.RoleUser {
		return
	}
	since, err := s.seenSince(ctx, userID, constants.FeedApproverNews)
	if err != nil {
		return
	}

	pending, err := s.RepositoryApproval.GetNewPendingByApprover(ctx, userID, since)
	if err != nil {
		return
	}
	for _, r := range pending {
		res = append(res, dto.ApproverNewsResponseGet{TaskID: r.TaskID, SiteID: r.SiteID, Kind: constants.ApproverNewsNewTask, CreatedAt: r.At})
	}
	returned, err := s.RepositoryApproval.GetDivHeadReturnsAfterApproval(ctx, userID, since)
	if err != nil {
		return
	}
	for _, r := range returned {
		res = append(res, dto.ApproverNewsResponseGet{TaskID: r.TaskID, SiteID: r.SiteID, Kind: constants.ApproverNewsDivHeadReturned, CreatedAt: r.At})
	}
	sort.SliceStable(res, func(i, j int) bool { return res[i].CreatedAt.After(res[j].CreatedAt) })
	return
}
