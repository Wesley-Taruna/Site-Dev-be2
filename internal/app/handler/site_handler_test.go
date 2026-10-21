package handler

import (
	"net/http"
	"testing"
	"time"

	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/middleware"
	"site-dev-backend/internal/app/mocks"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/fami-app-v2/familib/familabs"
)

func newSiteTestEcho(t *testing.T) (*echo.Echo, *mocks.ISiteService, *mocks.ISiteContextService) {
	e := echo.New()
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return next(&familabs.Context{Context: c, ResponseFormat: familabs.NewResponse()})
		}
	})
	e.Binder = &familabs.Binder{}
	e.HTTPErrorHandler = familabs.HTTPErrorHandler
	site, siteContext := mocks.NewISiteService(t), mocks.NewISiteContextService(t)
	o := testHandlerOptions()
	h := SiteHandler{Option: o, ServicesSite: site, ServicesSiteContext: siteContext}
	h.routes(e.Group("v1/").Group("site"), &middleware.Middleware{Option: o.Common})
	return e, site, siteContext
}

func TestSiteHandlerRoutes(t *testing.T) {
	e, site, siteContext := newSiteTestEcho(t)
	token := testToken(t, 1001, time.Now().Add(time.Hour))
	site.On("Get", mock.Anything, dto.SiteRequestGet{ListRequestGet: dto.ListRequestGet{Page: 1}, City: "Jakarta", Region: "DKI"}).Return([]dto.SiteResponseGet{}, int64(0), nil)
	site.On("GetMaps", mock.Anything).Return(dto.SiteMapListResponseGet{Sites: []dto.SiteMapResponseGet{}}, nil)
	site.On("GetActiveStores", mock.Anything, dto.ListRequestGet{Page: 3, Search: "fami"}).Return([]dto.SiteMapResponseGet{}, int64(0), nil)
	site.On("GetByID", mock.Anything, "ST1").Return(dto.SiteResponseGet{SiteID: "ST1"}, nil)
	siteContext.On("GetBySiteID", mock.Anything, "ST1").Return(dto.SiteContextResponseGet{SiteID: "ST1"}, nil)

	for _, path := range []string{"/v1/site?city=Jakarta&region=DKI", "/v1/site/maps", "/v1/site/active-list?page=3&q=fami", "/v1/site/ST1", "/v1/site/ST1/context"} {
		code, _ := doGet(e, path, token)
		assert.Equal(t, http.StatusOK, code, path)
	}
}
