package service

import (
	"context"
	"strings"
	"unicode"

	"site-dev-backend/global"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/model"
	"site-dev-backend/internal/app/repository"

	"gitlab.com/fami-app-v2/familib/familabs/log"
	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

type ISiteService interface {
	// GetByID is GET /v1/site/:id.
	GetByID(ctx context.Context, siteID string) (res dto.SiteResponseGet, err error)
	// Get is GET /v1/site (city/region filters).
	Get(ctx context.Context, req dto.SiteRequestGet) (res []dto.SiteResponseGet, total int64, err error)
	// GetMaps is GET /v1/site/maps: active stores of AMFIBI_MASTER plus the
	// requested sites whose name is not already one of them.
	GetMaps(ctx context.Context) (res dto.SiteMapListResponseGet, err error)
	// GetActiveStores is GET /v1/site/active-list ("Site Active" table,
	// straight from AMFIBI_MASTER).
	GetActiveStores(ctx context.Context, req dto.ListRequestGet) (res []dto.SiteMapResponseGet, total int64, err error)
	// UpsertWithStatus is POST /v1/task/:status (DRAFT or SUBMITED): saves
	// the site, survey and attachments and its task in one transaction;
	// SUBMITED also opens the approval round.
	UpsertWithStatus(ctx context.Context, req dto.SiteRequestUpsert, status string, actorID int64) (res dto.SiteUpsertResponse, err error)
}

type SiteService struct {
	opt                     opt.Options
	RepositorySite          repository.ISiteRepository
	RepositoryTask          repository.ITaskRepository
	RepositorySiteContext   repository.ISiteContextRepository
	RepositorySiteAIInsight repository.ISiteAIInsightRepository
	RepositoryRunningID     repository.IRunningIDRepository
	RepositoryUser          repository.IUserRepository
	RepositoryTransaction   repository.ITransactionRepository
	taskService             *TaskService
}

func NewSiteService() ISiteService {
	return &SiteService{
		opt:                     global.Setup.Common,
		RepositorySite:          repository.NewSiteRepository(),
		RepositoryTask:          repository.NewTaskRepository(),
		RepositorySiteContext:   repository.NewSiteContextRepository(),
		RepositorySiteAIInsight: repository.NewSiteAIInsightRepository(),
		RepositoryRunningID:     repository.NewRunningIDRepository(),
		RepositoryUser:          repository.NewUserRepository(),
		RepositoryTransaction:   repository.NewTransactionRepository(),
		taskService:             newTaskService(),
	}
}

func (s *SiteService) GetByID(ctx context.Context, siteID string) (res dto.SiteResponseGet, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "SiteService.GetByID")
	defer span.End()

	var site *model.Site
	site, err = s.RepositorySite.GetByID(ctx, siteID)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}
	res = toSiteResponse(site)

	return
}

func (s *SiteService) Get(ctx context.Context, req dto.SiteRequestGet) (res []dto.SiteResponseGet, total int64, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "SiteService.Get")
	defer span.End()

	orderBy := orderClause(req.SortBy, req.SortDir, "CreatedAt", map[string]bool{"CreatedAt": true, "SiteName1": true, "SiteCity": true})

	var sites []*model.Site
	sites, total, err = s.RepositorySite.Get(ctx, pageOffset(req.Page), dto.ListPageSize, orderBy, req.City, req.Region)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	res = make([]dto.SiteResponseGet, 0, len(sites))
	for _, site := range sites {
		res = append(res, toSiteResponse(site))
	}

	return
}

// normalizeStoreName drops case, spaces and punctuation so a store and a
// site with slightly different spelling still match ("Fami Mart  01" =
// "FAMI MART 01").
func normalizeStoreName(name string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(name) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (s *SiteService) GetMaps(ctx context.Context) (res dto.SiteMapListResponseGet, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "SiteService.GetMaps")
	defer span.End()

	var stores []model.StoreMap
	stores, err = s.RepositorySite.GetActiveStoreMaps(ctx)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	points := make([]dto.SiteMapResponseGet, 0, len(stores))
	matched := make(map[string]struct{}, len(stores))
	for _, st := range stores {
		points = append(points, dto.SiteMapResponseGet{
			StoreName:     st.StoreName,
			StoreTypeName: st.StoreTypeName,
			SiteType:      st.SiteType,
			Longitude:     st.Longitude,
			Latitude:      st.Latitude,
		})
		matched[normalizeStoreName(st.StoreName)] = struct{}{}
	}

	var sites []*model.Site
	sites, err = s.RepositorySite.GetAll(ctx)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}
	for _, site := range sites {
		if _, ok := matched[normalizeStoreName(site.SiteName1)]; ok {
			continue
		}
		points = append(points, dto.SiteMapResponseGet{
			StoreName:     site.SiteName1,
			StoreTypeName: "Requested",
			SiteType:      "Site Requested",
			Longitude:     site.SiteLon,
			Latitude:      site.SiteLat,
		})
	}

	// count per store type, in order of first appearance
	order := make([]string, 0)
	counts := make(map[string]int)
	for _, p := range points {
		if _, ok := counts[p.StoreTypeName]; !ok {
			order = append(order, p.StoreTypeName)
		}
		counts[p.StoreTypeName]++
	}
	types := make([]dto.SiteMapStoreTypeCount, 0, len(order))
	for _, name := range order {
		types = append(types, dto.SiteMapStoreTypeCount{StoreTypeName: name, Total: counts[name]})
	}

	res = dto.SiteMapListResponseGet{Sites: points, StoreTypes: types}

	return
}

func (s *SiteService) GetActiveStores(ctx context.Context, req dto.ListRequestGet) (res []dto.SiteMapResponseGet, total int64, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "SiteService.GetActiveStores")
	defer span.End()

	var stores []model.StoreMap
	stores, total, err = s.RepositorySite.GetActiveStores(ctx, req.Search, dto.ListPageSize, pageOffset(req.Page))
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	res = make([]dto.SiteMapResponseGet, 0, len(stores))
	for _, st := range stores {
		res = append(res, dto.SiteMapResponseGet{
			StoreCode:      st.StoreCode,
			StoreName:      st.StoreName,
			StoreTypeName:  st.StoreTypeName,
			SiteType:       st.SiteType,
			Longitude:      st.Longitude,
			Latitude:       st.Latitude,
			OpeningStoreDT: st.OpeningStoreDT,
		})
	}

	return
}
