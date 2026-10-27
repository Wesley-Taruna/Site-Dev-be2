package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/constants"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/model"
	"site-dev-backend/internal/app/repository"

	"github.com/labstack/echo/v4"
	"gitlab.com/fami-app-v2/familib/familabs/log"
	"gitlab.com/fami-app-v2/familib/familabs/opt"
	"gorm.io/gorm"
)

// IPortalService is the "Site Dev" menu of FamilyMart's web portal
// (November scope, API-CHANGES §11 + 20/21 Oct): site log, counts, report,
// ADS.
//
// Who may use it is OPEN (portal login, review §5 #12; ADS editor, #11).
// DEFAULTS until decided: reading = every known user except site staff
// (USER); editing ADS = ADMIN only (the existing admin check).
type IPortalService interface {
	Get(ctx context.Context, req dto.PortalSiteRequestGet, userID int64) (res []dto.PortalSiteResponseGet, total int64, err error)
	CountByGroup(ctx context.Context, userID int64) (res []dto.PortalStatusCountResponseGet, err error)
	GetReport(ctx context.Context, siteID string, userID int64) (res dto.PortalReportResponseGet, err error)
	UpdateAds(ctx context.Context, siteID string, req dto.SiteAdsRequestUpdate, userID int64) (res dto.SiteAdsResponse, err error)
}

type PortalService struct {
	opt                opt.Options
	RepositoryTask     repository.ITaskRepository
	RepositorySite     repository.ISiteRepository
	RepositoryUser     repository.IUserRepository
	RepositoryApproval repository.IApprovalRepository
	RepositorySiteAds  repository.ISiteAdsRepository
	ServicesAIInsight  IAIInsightService
	taskService        *TaskService
	now                func() time.Time
}

func NewPortalService() IPortalService {
	return &PortalService{
		opt:                global.Setup.Common,
		RepositoryTask:     repository.NewTaskRepository(),
		RepositorySite:     repository.NewSiteRepository(),
		RepositoryUser:     repository.NewUserRepository(),
		RepositoryApproval: repository.NewApprovalRepository(),
		RepositorySiteAds:  repository.NewSiteAdsRepository(),
		ServicesAIInsight:  NewAIInsightService(),
		taskService:        newTaskService(),
		now:                time.Now,
	}
}

// portalGroupOrder is the order of the counts (prototype mock.ts).
var portalGroupOrder = []string{dto.PortalGroupWaitingDeptHead, dto.PortalGroupWaitingDivHead, dto.PortalGroupHold, dto.PortalGroupApproved, dto.PortalGroupRejected}

// portalGroupStatuses are the task statuses of each group.
var portalGroupStatuses = map[string][]string{
	dto.PortalGroupWaitingDeptHead: {constants.TaskStatusWaiting(constants.RoleDeptHead)},
	dto.PortalGroupWaitingDivHead:  {constants.TaskStatusWaiting(constants.RoleDivHead)},
	dto.PortalGroupHold:            constants.HoldStatuses,
	dto.PortalGroupApproved:        {constants.TaskStatusCompleted, constants.TaskStatusApproved},
	dto.PortalGroupRejected:        {constants.TaskStatusRejected, constants.TaskStatusRevision},
}

// portalGroupOf is the group of a task status ("" = not in the portal).
func portalGroupOf(status string) string {
	for _, g := range portalGroupOrder {
		for _, st := range portalGroupStatuses[g] {
			if st == status {
				return g
			}
		}
	}
	return ""
}

// portalStatuses is every status shown in the portal.
func portalStatuses() (out []string) {
	for _, g := range portalGroupOrder {
		out = append(out, portalGroupStatuses[g]...)
	}
	return
}

// requireReader: DEFAULT portal access, everyone known except site staff.
func (s *PortalService) requireReader(ctx context.Context, userID int64) error {
	user, err := getUser(ctx, s.RepositoryUser, userID)
	if err != nil {
		return err
	}
	if user.RoleID == constants.RoleUser {
		return echo.NewHTTPError(http.StatusForbidden, "Portal Site Dev hanya untuk pengguna kantor")
	}
	return nil
}

func (s *PortalService) fail(err error) {
	s.opt.Logger.AddMessage(log.ErrorLevel, err)
}

func (s *PortalService) Get(ctx context.Context, req dto.PortalSiteRequestGet, userID int64) (res []dto.PortalSiteResponseGet, total int64, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "PortalService.Get")
	defer span.End()

	res, total, err = s.get(ctx, req, userID)
	if err != nil {
		span.RecordError(err)
		s.fail(err)
		res, total = nil, 0
	}
	return
}

func (s *PortalService) get(ctx context.Context, req dto.PortalSiteRequestGet, userID int64) (res []dto.PortalSiteResponseGet, total int64, err error) {
	if err = s.requireReader(ctx, userID); err != nil {
		return
	}
	statuses := portalStatuses()
	if req.Status != "" {
		var ok bool
		if statuses, ok = portalGroupStatuses[req.Status]; !ok {
			err = echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("status %q tidak dikenal", req.Status))
			return
		}
	}

	var tasks []*model.Task
	if tasks, total, err = s.RepositoryTask.GetPortal(ctx, statuses, strings.TrimSpace(req.Search), req.Offset, req.Limit); err != nil {
		return
	}
	res = []dto.PortalSiteResponseGet{}
	if len(tasks) == 0 {
		return
	}
	res, err = s.rows(ctx, tasks)
	return
}

// rows adds site, creator, first submit, hold reason and ADS (one query
// each, plus one per creator / ADS editor).
func (s *PortalService) rows(ctx context.Context, tasks []*model.Task) (res []dto.PortalSiteResponseGet, err error) {
	taskIDs := make([]string, 0, len(tasks))
	siteIDs := make([]string, 0, len(tasks))
	var heldIDs []string
	for _, t := range tasks {
		taskIDs = append(taskIDs, t.TaskID)
		siteIDs = append(siteIDs, t.SiteID)
		if portalGroupOf(t.TaskStatus) == dto.PortalGroupHold {
			heldIDs = append(heldIDs, t.TaskID)
		}
	}

	sites, err := s.RepositorySite.GetByIDs(ctx, siteIDs)
	if err != nil {
		return
	}
	siteByID := make(map[string]*model.Site, len(sites))
	for _, site := range sites {
		siteByID[site.SiteID] = site
	}
	submitted, err := s.RepositoryApproval.GetFirstSubmittedAt(ctx, taskIDs)
	if err != nil {
		return
	}
	submittedByTask := make(map[string]time.Time, len(submitted))
	for _, r := range submitted {
		submittedByTask[r.TaskID] = r.SubmittedAt
	}
	holdNotes := map[string]string{}
	if len(heldIDs) > 0 {
		var notes []model.TaskLevelNote
		if notes, err = s.RepositoryApproval.GetLevelNotesByTaskIDs(ctx, heldIDs, constants.ApprovalDetailStatusHold); err != nil {
			return
		}
		for _, n := range notes {
			holdNotes[n.TaskID] = n.Notes
		}
	}
	ads, err := s.adsBySite(ctx, siteIDs)
	if err != nil {
		return
	}

	names := map[int64]string{}
	res = make([]dto.PortalSiteResponseGet, 0, len(tasks))
	for _, t := range tasks {
		row := dto.PortalSiteResponseGet{
			SiteID:       t.SiteID,
			TaskID:       t.TaskID,
			TaskStatus:   t.TaskStatus,
			StatusGroup:  portalGroupOf(t.TaskStatus),
			CreatedBy:    t.CreatedBy,
			CreatedByNIK: nikOf(t.CreatedBy),
			UpdatedAt:    t.UpdatedAt,
			Ads:          ads[t.SiteID],
		}
		if site := siteByID[t.SiteID]; site != nil {
			row.SiteName1, row.SiteAddress1, row.SiteAddress2 = site.SiteName1, site.SiteAddress1, site.SiteAddress2
			row.SiteDistrict, row.SiteCity, row.SiteRegion, row.SitePostalCode = site.SiteDistrict, site.SiteCity, site.SiteRegion, site.SitePostalCode
			row.SiteLat, row.SiteLon = site.SiteLat, site.SiteLon
		}
		if name, ok := names[t.CreatedBy]; ok {
			row.CreatedByName = name
		} else {
			row.CreatedByName = usernameOf(ctx, s.RepositoryUser, t.CreatedBy)
			names[t.CreatedBy] = row.CreatedByName
		}
		if at, ok := submittedByTask[t.TaskID]; ok {
			at := at
			row.SubmittedAt = &at
		}
		if row.StatusGroup == dto.PortalGroupHold {
			note := holdNotes[t.TaskID]
			row.HoldNote = &note
		}
		res = append(res, row)
	}
	return
}

// nikOf: Users.UserID holds the employee's NIK (model.User).
func nikOf(userID int64) string {
	if userID == 0 {
		return ""
	}
	return strconv.FormatInt(userID, 10)
}

// adsBySite loads the ADS rows with the editor's name.
func (s *PortalService) adsBySite(ctx context.Context, siteIDs []string) (map[string]*dto.SiteAdsResponse, error) {
	rows, err := s.RepositorySiteAds.GetBySiteIDs(ctx, siteIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[string]*dto.SiteAdsResponse, len(rows))
	names := map[int64]string{}
	for _, a := range rows {
		name, ok := names[a.UpdatedBy]
		if !ok {
			name = usernameOf(ctx, s.RepositoryUser, a.UpdatedBy)
			names[a.UpdatedBy] = name
		}
		out[a.SiteID] = toSiteAdsResponse(a, name)
	}
	return out, nil
}

func toSiteAdsResponse(a *model.SiteAds, updatedByName string) *dto.SiteAdsResponse {
	at := a.UpdatedAt
	return &dto.SiteAdsResponse{AdsValue: a.AdsValue, AdsNote: a.AdsNote, UpdatedByName: updatedByName, UpdatedAt: &at}
}

func (s *PortalService) CountByGroup(ctx context.Context, userID int64) (res []dto.PortalStatusCountResponseGet, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "PortalService.CountByGroup")
	defer span.End()

	if err = s.requireReader(ctx, userID); err != nil {
		span.RecordError(err)
		s.fail(err)
		return
	}
	var counts []model.TaskStatusCount
	if counts, err = s.RepositoryTask.CountByStatus(ctx, 0); err != nil {
		span.RecordError(err)
		s.fail(err)
		return
	}

	byGroup := map[string]int64{}
	for _, c := range counts {
		if g := portalGroupOf(c.TaskStatus); g != "" {
			byGroup[g] += c.Count
		}
	}
	res = make([]dto.PortalStatusCountResponseGet, 0, len(portalGroupOrder))
	for _, g := range portalGroupOrder {
		res = append(res, dto.PortalStatusCountResponseGet{StatusGroup: g, Count: byGroup[g]})
	}
	return
}

func (s *PortalService) GetReport(ctx context.Context, siteID string, userID int64) (res dto.PortalReportResponseGet, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "PortalService.GetReport")
	defer span.End()

	res, err = s.report(ctx, siteID, userID)
	if err != nil {
		span.RecordError(err)
		s.fail(err)
		res = dto.PortalReportResponseGet{}
	}
	return
}

func (s *PortalService) report(ctx context.Context, siteID string, userID int64) (res dto.PortalReportResponseGet, err error) {
	if err = s.requireReader(ctx, userID); err != nil {
		return
	}
	if !constants.IsSiteID(siteID) {
		err = echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("id %q bukan site_id", siteID))
		return
	}

	// same parts as GET /v1/task/:id
	detail, err := s.taskService.detail(ctx, siteID)
	if err != nil {
		return
	}
	res.Site = dto.PortalReportSite{SiteResponseGet: detail.Site, CreatedByNIK: nikOf(detail.Site.CreatedBy)}
	res.Context, res.Task, res.Attachment = detail.Context, detail.Task, detail.Attachment
	res.StaffNote = staffNoteOf(detail.Task.History)
	if g := portalGroupOf(detail.Task.TaskStatus); g != "" {
		res.StatusGroup = &g
	}

	// the stored insight (never generated = null); TODO when §6 stores the
	// map data ("measured"), read it here so a later PDF shows the same
	// numbers (review #20, needs a DB change)
	insight, insightErr := s.ServicesAIInsight.Get(ctx, siteID)
	if insightErr == nil {
		res.Insight = &insight
	} else if !errors.Is(insightErr, gorm.ErrRecordNotFound) {
		err = insightErr
		return
	}

	ads, err := s.adsBySite(ctx, []string{siteID})
	if err != nil {
		return
	}
	res.Ads = ads[siteID]
	return
}

// staffNoteOf is the staff's note of the newest send: the "(catatan: …)"
// part of the newest SUBMIT/RESUBMIT history row, without the automatic
// "Site diajukan oleh …" text. DEFAULT: there is no column for the raw
// note, so it is read back from the history text.
func staffNoteOf(history []dto.TaskHistoryResponseGet) string {
	const marker = " (catatan: "
	for i := len(history) - 1; i >= 0; i-- {
		h := history[i]
		if h.Action != constants.TaskActionSubmit && h.Action != constants.TaskActionResubmit {
			continue
		}
		at := strings.Index(h.Notes, marker)
		if at < 0 || !strings.HasSuffix(h.Notes, ")") {
			return ""
		}
		note := h.Notes[at+len(marker) : len(h.Notes)-1]
		if strings.HasPrefix(note, "Site diajukan oleh ") && strings.HasSuffix(note, " untuk proses approval") {
			return ""
		}
		return note
	}
	return ""
}

func (s *PortalService) UpdateAds(ctx context.Context, siteID string, req dto.SiteAdsRequestUpdate, userID int64) (res dto.SiteAdsResponse, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "PortalService.UpdateAds")
	defer span.End()

	res, err = s.updateAds(ctx, siteID, req, userID)
	if err != nil {
		span.RecordError(err)
		s.fail(err)
		res = dto.SiteAdsResponse{}
	}
	return
}

func (s *PortalService) updateAds(ctx context.Context, siteID string, req dto.SiteAdsRequestUpdate, userID int64) (res dto.SiteAdsResponse, err error) {
	// the editor check (DEFAULT: ADMIN, review §5 #11) is done by the
	// handler before the body is read, as for the other admin endpoints
	task, err := s.RepositoryTask.GetBySiteID(ctx, siteID)
	if err != nil {
		return
	}
	if portalGroupOf(task.TaskStatus) == "" {
		err = echo.NewHTTPError(http.StatusBadRequest, "ADS hanya untuk site yang tampil di portal.")
		return
	}

	row := &model.SiteAds{SiteID: siteID, AdsValue: req.AdsValue, AdsNote: strings.TrimSpace(req.AdsNote), UpdatedBy: userID, UpdatedAt: s.now()}
	if err = s.RepositorySiteAds.Upsert(ctx, row); err != nil {
		return
	}
	res = *toSiteAdsResponse(row, usernameOf(ctx, s.RepositoryUser, userID))
	return
}
