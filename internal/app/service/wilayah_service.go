package service

import (
	"context"

	"site-dev-backend/global"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/model"
	"site-dev-backend/internal/app/repository"

	"gitlab.com/fami-app-v2/familib/familabs/log"
	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

type IWilayahService interface {
	// SearchByName finds villages by part of the name.
	SearchByName(ctx context.Context, search string, page int) (res []dto.WilayahSearchResponseGet, total int64, err error)
	// SearchByPostalCode finds villages by exact postal code.
	SearchByPostalCode(ctx context.Context, postalCode string, page int) (res []dto.WilayahSearchResponseGet, total int64, err error)
	GetProvinces(ctx context.Context) (res []dto.ProvinceResponseGet, err error)
	// GetProvinceTree is one province down to every village.
	GetProvinceTree(ctx context.Context, provinceCode string) (res dto.ProvinceTreeResponseGet, err error)
}

type WilayahService struct {
	opt               opt.Options
	RepositoryWilayah repository.IWilayahRepository
}

func NewWilayahService() IWilayahService {
	return &WilayahService{
		opt:               global.Setup.Common,
		RepositoryWilayah: repository.NewWilayahRepository(),
	}
}

func toWilayahSearchResponses(rows []model.WilayahSearchRow) []dto.WilayahSearchResponseGet {
	res := make([]dto.WilayahSearchResponseGet, 0, len(rows))
	for _, r := range rows {
		res = append(res, dto.WilayahSearchResponseGet{
			VillageCode:  r.VillageCode,
			VillageName:  r.VillageName,
			VillageType:  r.VillageType,
			PostalCode:   r.PostalCode,
			DistrictCode: r.DistrictCode,
			DistrictName: r.DistrictName,
			RegencyCode:  r.RegencyCode,
			RegencyName:  r.RegencyName,
			ProvinceCode: r.ProvinceCode,
			ProvinceName: r.ProvinceName,
		})
	}
	return res
}

func (s *WilayahService) SearchByName(ctx context.Context, search string, page int) (res []dto.WilayahSearchResponseGet, total int64, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "WilayahService.SearchByName")
	defer span.End()

	var rows []model.WilayahSearchRow
	rows, total, err = s.RepositoryWilayah.SearchByName(ctx, search, dto.ListPageSize, pageOffset(page))
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}
	res = toWilayahSearchResponses(rows)

	return
}

func (s *WilayahService) SearchByPostalCode(ctx context.Context, postalCode string, page int) (res []dto.WilayahSearchResponseGet, total int64, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "WilayahService.SearchByPostalCode")
	defer span.End()

	var rows []model.WilayahSearchRow
	rows, total, err = s.RepositoryWilayah.SearchByPostalCode(ctx, postalCode, dto.ListPageSize, pageOffset(page))
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}
	res = toWilayahSearchResponses(rows)

	return
}

func (s *WilayahService) GetProvinces(ctx context.Context) (res []dto.ProvinceResponseGet, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "WilayahService.GetProvinces")
	defer span.End()

	var provinces []model.MProvince
	provinces, err = s.RepositoryWilayah.GetProvinces(ctx)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}

	res = make([]dto.ProvinceResponseGet, 0, len(provinces))
	for _, p := range provinces {
		res = append(res, dto.ProvinceResponseGet{ProvinceCode: p.ProvinceCode, ProvinceName: p.ProvinceName})
	}

	return
}

func (s *WilayahService) GetProvinceTree(ctx context.Context, provinceCode string) (res dto.ProvinceTreeResponseGet, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "WilayahService.GetProvinceTree")
	defer span.End()

	res, err = s.provinceTree(ctx, provinceCode)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		res = dto.ProvinceTreeResponseGet{}
		return
	}

	return
}

func (s *WilayahService) provinceTree(ctx context.Context, provinceCode string) (res dto.ProvinceTreeResponseGet, err error) {
	province, err := s.RepositoryWilayah.GetProvinceByCode(ctx, provinceCode)
	if err != nil {
		return
	}

	regencies, err := s.RepositoryWilayah.GetRegenciesByProvince(ctx, provinceCode)
	if err != nil {
		return
	}
	regencyCodes := make([]string, 0, len(regencies))
	for _, r := range regencies {
		regencyCodes = append(regencyCodes, r.RegencyCode)
	}

	districts, err := s.RepositoryWilayah.GetDistrictsByRegencyCodes(ctx, regencyCodes)
	if err != nil {
		return
	}
	districtCodes := make([]string, 0, len(districts))
	for _, d := range districts {
		districtCodes = append(districtCodes, d.DistrictCode)
	}

	villages, err := s.RepositoryWilayah.GetVillagesByDistrictCodes(ctx, districtCodes)
	if err != nil {
		return
	}

	villagesByDistrict := make(map[string][]dto.VillageItem, len(districts))
	for _, v := range villages {
		villagesByDistrict[v.DistrictCode] = append(villagesByDistrict[v.DistrictCode], dto.VillageItem{
			VillageCode: v.VillageCode,
			VillageName: v.VillageName,
			VillageType: v.VillageType,
			PostalCode:  v.PostalCode,
		})
	}
	districtsByRegency := make(map[string][]dto.DistrictTreeItem, len(regencies))
	for _, d := range districts {
		districtsByRegency[d.RegencyCode] = append(districtsByRegency[d.RegencyCode], dto.DistrictTreeItem{
			DistrictCode: d.DistrictCode,
			DistrictName: d.DistrictName,
			Villages:     villagesByDistrict[d.DistrictCode],
		})
	}
	items := make([]dto.RegencyTreeItem, 0, len(regencies))
	for _, r := range regencies {
		items = append(items, dto.RegencyTreeItem{
			RegencyCode: r.RegencyCode,
			RegencyName: r.RegencyName,
			Districts:   districtsByRegency[r.RegencyCode],
		})
	}

	res = dto.ProvinceTreeResponseGet{
		ProvinceCode: province.ProvinceCode,
		ProvinceName: province.ProvinceName,
		Regencies:    items,
	}
	return
}
