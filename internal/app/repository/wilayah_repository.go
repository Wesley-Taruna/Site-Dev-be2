package repository

import (
	"context"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/model"

	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

type IWilayahRepository interface {
	// SearchByName finds villages whose name contains search, with their
	// district, regency and province.
	SearchByName(ctx context.Context, search string, limit int, offset int) (rows []model.WilayahSearchRow, count int64, err error)
	// SearchByPostalCode finds villages with exactly this postal code.
	SearchByPostalCode(ctx context.Context, postalCode string, limit int, offset int) (rows []model.WilayahSearchRow, count int64, err error)
	GetProvinces(ctx context.Context) (provinces []model.MProvince, err error)
	GetProvinceByCode(ctx context.Context, code string) (province *model.MProvince, err error)
	GetRegenciesByProvince(ctx context.Context, provinceCode string) (regencies []model.MRegency, err error)
	GetDistrictsByRegencyCodes(ctx context.Context, regencyCodes []string) (districts []model.MDistrict, err error)
	GetVillagesByDistrictCodes(ctx context.Context, districtCodes []string) (villages []model.MVillage, err error)
}

type WilayahRepository struct {
	opt opt.Options
}

func NewWilayahRepository() IWilayahRepository {
	return &WilayahRepository{
		opt: global.Setup.Common,
	}
}

// wilayahSearchByNameQuery and wilayahSearchByPostalCodeQuery are
// site-dev-api's queries, unchanged.
const wilayahSearchByNameQuery = `
	SELECT
		v.VillageCode, v.VillageName, v.VillageType, v.PostalCode,
		d.DistrictCode, d.DistrictName,
		r.RegencyCode, r.RegencyName,
		p.ProvinceCode, p.ProvinceName
	FROM MVillages v
	INNER JOIN MDistricts d ON d.DistrictCode = v.DistrictCode
	INNER JOIN MRegencies r ON r.RegencyCode = d.RegencyCode
	INNER JOIN MProvinces p ON p.ProvinceCode = r.ProvinceCode
	WHERE v.VillageName LIKE ?
	ORDER BY v.VillageName
	OFFSET ? ROWS FETCH NEXT ? ROWS ONLY
`

const wilayahSearchByPostalCodeQuery = `
	SELECT
		v.VillageCode, v.VillageName, v.VillageType, v.PostalCode,
		d.DistrictCode, d.DistrictName,
		r.RegencyCode, r.RegencyName,
		p.ProvinceCode, p.ProvinceName
	FROM MVillages v
	INNER JOIN MDistricts d ON d.DistrictCode = v.DistrictCode
	INNER JOIN MRegencies r ON r.RegencyCode = d.RegencyCode
	INNER JOIN MProvinces p ON p.ProvinceCode = r.ProvinceCode
	WHERE v.PostalCode = ?
	ORDER BY v.VillageName
	OFFSET ? ROWS FETCH NEXT ? ROWS ONLY
`

func (r *WilayahRepository) SearchByName(ctx context.Context, search string, limit int, offset int) (rows []model.WilayahSearchRow, count int64, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "WilayahRepository.SearchByName")
	defer span.End()

	db := readDB(ctx, r.opt)
	like := "%" + search + "%"

	if err = db.Model(new(model.MVillage)).Where("VillageName LIKE ?", like).Count(&count).Error; err != nil {
		span.RecordError(err)
		return
	}

	if err = db.Raw(wilayahSearchByNameQuery, like, offset, limit).Scan(&rows).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *WilayahRepository) SearchByPostalCode(ctx context.Context, postalCode string, limit int, offset int) (rows []model.WilayahSearchRow, count int64, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "WilayahRepository.SearchByPostalCode")
	defer span.End()

	db := readDB(ctx, r.opt)

	if err = db.Model(new(model.MVillage)).Where("PostalCode = ?", postalCode).Count(&count).Error; err != nil {
		span.RecordError(err)
		return
	}

	if err = db.Raw(wilayahSearchByPostalCodeQuery, postalCode, offset, limit).Scan(&rows).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *WilayahRepository) GetProvinces(ctx context.Context) (provinces []model.MProvince, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "WilayahRepository.GetProvinces")
	defer span.End()

	db := readDB(ctx, r.opt)

	if err = db.Order("ProvinceName").Find(&provinces).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *WilayahRepository) GetProvinceByCode(ctx context.Context, code string) (province *model.MProvince, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "WilayahRepository.GetProvinceByCode")
	defer span.End()

	db := readDB(ctx, r.opt)

	province = &model.MProvince{}
	if err = db.Where("ProvinceCode = ?", code).First(province).Error; err != nil {
		span.RecordError(err)
		province = nil
		return
	}

	return
}

func (r *WilayahRepository) GetRegenciesByProvince(ctx context.Context, provinceCode string) (regencies []model.MRegency, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "WilayahRepository.GetRegenciesByProvince")
	defer span.End()

	db := readDB(ctx, r.opt)

	if err = db.Where("ProvinceCode = ?", provinceCode).Order("RegencyName").Find(&regencies).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *WilayahRepository) GetDistrictsByRegencyCodes(ctx context.Context, regencyCodes []string) (districts []model.MDistrict, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "WilayahRepository.GetDistrictsByRegencyCodes")
	defer span.End()

	if len(regencyCodes) == 0 {
		return
	}

	db := readDB(ctx, r.opt)

	if err = db.Where("RegencyCode IN ?", regencyCodes).Order("DistrictName").Find(&districts).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *WilayahRepository) GetVillagesByDistrictCodes(ctx context.Context, districtCodes []string) (villages []model.MVillage, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "WilayahRepository.GetVillagesByDistrictCodes")
	defer span.End()

	if len(districtCodes) == 0 {
		return
	}

	db := readDB(ctx, r.opt)

	if err = db.Where("DistrictCode IN ?", districtCodes).Order("VillageName").Find(&villages).Error; err != nil {
		span.RecordError(err)
		return
	}

	return
}
