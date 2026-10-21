package repository

import (
	"context"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/model"

	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

type ISiteRepository interface {
	// GetByIDs loads the sites for a set of SiteIDs (e.g. to add site
	// names to a task list).
	GetByIDs(ctx context.Context, siteIDs []string) (sites []*model.Site, err error)
	// GetIDsBySearch returns the SiteIDs whose ID, names or address lines
	// contain search (LIKE '%search%').
	GetIDsBySearch(ctx context.Context, search string) (siteIDs []string, err error)
	GetByID(ctx context.Context, siteID string) (site *model.Site, err error)
	// GetAttachmentsBySiteID returns a site's attachments, oldest first.
	GetAttachmentsBySiteID(ctx context.Context, siteID string) (attachments []*model.SiteAttachment, err error)
	// Get lists sites (city/region "" = no filter), with the total count.
	Get(ctx context.Context, offset int, limit int, orderBy string, city string, region string) (sites []*model.Site, count int64, err error)
	// GetAll returns every site (map view).
	GetAll(ctx context.Context) (sites []*model.Site, err error)
	// GetActiveStoreMaps returns the active stores of AMFIBI_MASTER for the
	// map view.
	GetActiveStoreMaps(ctx context.Context) (stores []model.StoreMap, err error)
	// GetActiveStores is the searchable, paged list of the same stores
	// ("Site Active" table); search matches store name or store type.
	GetActiveStores(ctx context.Context, search string, limit int, offset int) (stores []model.StoreMap, count int64, err error)
}

type SiteRepository struct {
	opt opt.Options
}

func NewSiteRepository() ISiteRepository {
	return &SiteRepository{
		opt: global.Setup.Common,
	}
}

func (r *SiteRepository) GetByIDs(ctx context.Context, siteIDs []string) (sites []*model.Site, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "SiteRepository.GetByIDs")
	defer span.End()

	if len(siteIDs) == 0 {
		return
	}

	db := readDB(ctx, r.opt)

	if err = db.Where("SiteID IN ?", siteIDs).Find(&sites).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *SiteRepository) GetIDsBySearch(ctx context.Context, search string) (siteIDs []string, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "SiteRepository.GetIDsBySearch")
	defer span.End()

	db := readDB(ctx, r.opt)

	like := "%" + search + "%"
	err = db.Model(new(model.Site)).
		Where("SiteID LIKE ? OR SiteName1 LIKE ? OR SiteName2 LIKE ? OR SiteAddress1 LIKE ? OR SiteAddress2 LIKE ? OR SiteAddress3 LIKE ?",
			like, like, like, like, like, like).
		Pluck("SiteID", &siteIDs).Error
	if err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *SiteRepository) GetByID(ctx context.Context, siteID string) (site *model.Site, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "SiteRepository.GetByID")
	defer span.End()

	db := readDB(ctx, r.opt)

	site = &model.Site{}
	if err = db.Where("SiteID = ?", siteID).First(site).Error; err != nil {
		span.RecordError(err)
		site = nil
		return
	}

	return
}

func (r *SiteRepository) GetAttachmentsBySiteID(ctx context.Context, siteID string) (attachments []*model.SiteAttachment, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "SiteRepository.GetAttachmentsBySiteID")
	defer span.End()

	db := readDB(ctx, r.opt)

	if err = db.Where("SiteID = ?", siteID).Order("SiteAttachmentID ASC").Find(&attachments).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *SiteRepository) Get(ctx context.Context, offset int, limit int, orderBy string, city string, region string) (sites []*model.Site, count int64, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "SiteRepository.Get")
	defer span.End()

	db := readDB(ctx, r.opt)

	if city != "" {
		db = db.Where("SiteCity = ?", city)
	}
	if region != "" {
		db = db.Where("SiteRegion = ?", region)
	}

	if err = db.Model(new(model.Site)).Count(&count).Error; err != nil {
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

	if err = db.Find(&sites).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *SiteRepository) GetAll(ctx context.Context) (sites []*model.Site, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "SiteRepository.GetAll")
	defer span.End()

	db := readDB(ctx, r.opt)

	if err = db.Find(&sites).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}

// activeStoreMapQuery, activeStoreListQuery and activeStoreCountQuery are
// site-dev-api's queries, unchanged.
const activeStoreMapQuery = `
	SELECT
		ms.StoreName,
		mt.StoreTypeName,
		'Site Active' as SiteType,
		ms.Longitude,
		ms.Latitude
	FROM AMFIBI_MASTER.dbo.MStore as ms
	INNER JOIN AMFIBI_MASTER.dbo.MStoreType mt on ms.StoreType = mt.StoreTypeID
	WHERE ms.Longitude != 0 and ms.IsActive = 1 and ms.StoreName != 'FOODTRUCK 1'
`

const activeStoreListQuery = `
	SELECT
		ms.StoreCode,
		ms.StoreName,
		mt.StoreTypeName,
		'Site Active' as SiteType,
		ms.Longitude,
		ms.Latitude,
		ms.OpeningStoreDT
	FROM AMFIBI_MASTER.dbo.MStore as ms
	INNER JOIN AMFIBI_MASTER.dbo.MStoreType mt on ms.StoreType = mt.StoreTypeID
	WHERE ms.Longitude != 0 and ms.IsActive = 1 and ms.StoreName != 'FOODTRUCK 1'
		AND (ms.StoreName LIKE ? OR mt.StoreTypeName LIKE ?)
	ORDER BY ms.StoreCode
	OFFSET ? ROWS FETCH NEXT ? ROWS ONLY
`

const activeStoreCountQuery = `
	SELECT COUNT(1)
	FROM AMFIBI_MASTER.dbo.MStore as ms
	INNER JOIN AMFIBI_MASTER.dbo.MStoreType mt on ms.StoreType = mt.StoreTypeID
	WHERE ms.Longitude != 0 and ms.IsActive = 1 and ms.StoreName != 'FOODTRUCK 1'
		AND (ms.StoreName LIKE ? OR mt.StoreTypeName LIKE ?)
`

func (r *SiteRepository) GetActiveStoreMaps(ctx context.Context) (stores []model.StoreMap, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "SiteRepository.GetActiveStoreMaps")
	defer span.End()

	db := readDB(ctx, r.opt)

	if err = db.Raw(activeStoreMapQuery).Scan(&stores).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *SiteRepository) GetActiveStores(ctx context.Context, search string, limit int, offset int) (stores []model.StoreMap, count int64, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "SiteRepository.GetActiveStores")
	defer span.End()

	db := readDB(ctx, r.opt)
	like := "%" + search + "%"

	if err = db.Raw(activeStoreCountQuery, like, like).Scan(&count).Error; err != nil {
		span.RecordError(err)
		return
	}

	if err = db.Raw(activeStoreListQuery, like, like, offset, limit).Scan(&stores).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}
