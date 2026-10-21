package service

import (
	"context"
	"errors"
	"testing"

	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/mocks"
	"site-dev-backend/internal/app/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestNormalizeStoreName(t *testing.T) {
	assert.Equal(t, "FAMIMART01", normalizeStoreName("Fami Mart  01"))
	assert.Equal(t, "FAMIMART01", normalizeStoreName("FAMI-MART 01."))
}

func TestSiteServiceGetMaps_EmptyListsAreArrays(t *testing.T) {
	repo := mocks.NewISiteRepository(t)
	s := &SiteService{opt: testOptions(), RepositorySite: repo}
	repo.On("GetActiveStoreMaps", ctxAny).Return(nil, nil)
	repo.On("GetAll", ctxAny).Return(nil, nil)

	res, err := s.GetMaps(context.Background())

	require.NoError(t, err)
	assert.Equal(t, dto.SiteMapListResponseGet{Sites: []dto.SiteMapResponseGet{}, StoreTypes: []dto.SiteMapStoreTypeCount{}}, res)
}

func TestSiteServiceGetMaps_StoreQueryFailure(t *testing.T) {
	repo := mocks.NewISiteRepository(t)
	s := &SiteService{opt: testOptions(), RepositorySite: repo}
	dbErr := errors.New("no access to AMFIBI_MASTER")
	repo.On("GetActiveStoreMaps", ctxAny).Return(nil, dbErr)

	_, err := s.GetMaps(context.Background())

	assert.ErrorIs(t, err, dbErr)
	repo.AssertNotCalled(t, "GetAll")
}

func TestSiteServiceGet_DefaultOrderAndPaging(t *testing.T) {
	repo := mocks.NewISiteRepository(t)
	s := &SiteService{opt: testOptions(), RepositorySite: repo}
	repo.On("Get", ctxAny, 20, 20, "CreatedAt ASC", "", "").Return([]*model.Site{{SiteID: "ST1"}}, int64(21), nil)

	res, total, err := s.Get(context.Background(), dto.SiteRequestGet{ListRequestGet: dto.ListRequestGet{Page: 2, SortBy: "SiteID"}})

	require.NoError(t, err)
	assert.Equal(t, int64(21), total)
	assert.Equal(t, "ST1", res[0].SiteID)
}

func TestSiteContextServiceGetBySiteID_NotFound(t *testing.T) {
	repo := mocks.NewISiteContextRepository(t)
	s := &SiteContextService{opt: testOptions(), RepositorySiteContext: repo}
	repo.On("GetBySiteID", ctxAny, "ST1").Return(nil, gorm.ErrRecordNotFound)

	_, err := s.GetBySiteID(context.Background(), "ST1")

	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
