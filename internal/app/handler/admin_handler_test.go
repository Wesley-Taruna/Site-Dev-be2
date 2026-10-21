package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func doJSON(e *echo.Echo, method, path, token string, body any) (int, map[string]any) {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(method, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func newBareEcho() *echo.Echo {
	e := echo.New()
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return next(&familabs.Context{Context: c, ResponseFormat: familabs.NewResponse()})
		}
	})
	e.Binder = &familabs.Binder{}
	e.HTTPErrorHandler = familabs.HTTPErrorHandler
	return e
}

func TestUserHandler_AdminCheckBeforeBody(t *testing.T) {
	e := newBareEcho()
	access, users := mocks.NewIAccessService(t), mocks.NewIUserService(t)
	o := testHandlerOptions()
	h := UserHandler{Option: o, ServicesAccess: access, ServicesUser: users}
	h.routes(e.Group("v1/").Group("users"), &middleware.Middleware{Option: o.Common})
	access.On("RequireAdmin", mock.Anything, int64(3001)).Return(echo.NewHTTPError(http.StatusForbidden, "hanya admin yang bisa mengakses fitur ini"))
	access.On("RequireAdmin", mock.Anything, int64(9001)).Return(nil)
	users.On("Create", mock.Anything, dto.UserRequestCreate{UserID: 5, Username: "eko", Email: "eko@example.com", RoleID: "USER"}).Return(dto.UserResponseGet{UserID: 5}, nil)
	users.On("Delete", mock.Anything, int64(5)).Return(nil)

	code, _ := doJSON(e, http.MethodPost, "/v1/users", testToken(t, 3001, time.Now().Add(time.Hour)), map[string]any{})
	assert.Equal(t, http.StatusForbidden, code)

	admin := testToken(t, 9001, time.Now().Add(time.Hour))
	code, body := doJSON(e, http.MethodPost, "/v1/users", admin, map[string]any{"user_id": 5, "username": "eko", "email": "eko@example.com", "role_id": "USER"})
	assert.Equal(t, http.StatusOK, code, body)
	code, _ = doJSON(e, http.MethodDelete, "/v1/users/5", admin, nil)
	assert.Equal(t, http.StatusOK, code)
	code, _ = doJSON(e, http.MethodDelete, "/v1/users/x", admin, nil)
	assert.Equal(t, http.StatusUnprocessableEntity, code)
}

func TestWilayahHandlerSearch_Branches(t *testing.T) {
	e := newBareEcho()
	svc := mocks.NewIWilayahService(t)
	o := testHandlerOptions()
	h := WilayahHandler{Option: o, ServicesWilayah: svc}
	h.routes(e.Group("v1/").Group("wilayah"), &middleware.Middleware{Option: o.Common})
	token := testToken(t, 1001, time.Now().Add(time.Hour))
	svc.On("SearchByPostalCode", mock.Anything, "16320", 1).Return([]dto.WilayahSearchResponseGet{}, int64(0), nil)
	svc.On("SearchByName", mock.Anything, "tajur", 2).Return([]dto.WilayahSearchResponseGet{}, int64(0), nil)

	code, _ := doGet(e, "/v1/wilayah/search?postal_code=16320", token)
	assert.Equal(t, http.StatusOK, code)
	code, _ = doGet(e, "/v1/wilayah/search?q=tajur&page=2", token)
	assert.Equal(t, http.StatusOK, code)
	code, _ = doGet(e, "/v1/wilayah/search?q=ta", token)
	assert.Equal(t, http.StatusUnprocessableEntity, code)
	code, body := doGet(e, "/v1/wilayah/search", token)
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, []any{}, body["data"])
}
