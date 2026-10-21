package service

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"site-dev-backend/internal/app/constants"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/mocks"
	"site-dev-backend/internal/app/model"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func httpCode(t *testing.T, err error) int {
	var he *echo.HTTPError
	require.True(t, errors.As(err, &he), "%v", err)
	return he.Code
}

func TestAccessServiceRequireAdmin(t *testing.T) {
	users := mocks.NewIUserRepository(t)
	s := &AccessService{opt: testOptions(), RepositoryUser: users}
	users.On("GetByID", ctxAny, int64(1)).Return(&model.User{RoleID: constants.RoleAdmin}, nil)
	users.On("GetByID", ctxAny, int64(2)).Return(&model.User{RoleID: constants.RoleDivHead}, nil)
	users.On("GetByID", ctxAny, int64(3)).Return(nil, gorm.ErrRecordNotFound)

	assert.NoError(t, s.RequireAdmin(context.Background(), 1))
	assert.Equal(t, http.StatusForbidden, httpCode(t, s.RequireAdmin(context.Background(), 2)))
	assert.Equal(t, http.StatusUnauthorized, httpCode(t, s.RequireAdmin(context.Background(), 3)))
}

func TestUserServiceUpdate_OnlySentFields(t *testing.T) {
	users, roles := mocks.NewIUserRepository(t), mocks.NewIRoleRepository(t)
	s := &UserService{opt: testOptions(), RepositoryUser: users, RepositoryRole: roles}
	users.On("Update", ctxAny, int64(1001), map[string]interface{}{"Email": "a@b.c"}).Return(nil)
	users.On("GetByID", ctxAny, int64(1001)).Return(&model.User{UserID: 1001, RoleID: "USER"}, nil)
	roles.On("GetByID", ctxAny, "USER").Return(nil, gorm.ErrRecordNotFound)

	res, err := s.Update(context.Background(), 1001, dto.UserRequestUpdate{Email: "a@b.c"})

	require.NoError(t, err)
	assert.Equal(t, "", res.RoleName, "unknown role name stays empty")
}

func TestUserServiceUpdate_UnknownRole(t *testing.T) {
	users, roles := mocks.NewIUserRepository(t), mocks.NewIRoleRepository(t)
	s := &UserService{opt: testOptions(), RepositoryUser: users, RepositoryRole: roles}
	roles.On("GetByID", ctxAny, "CEO").Return(nil, gorm.ErrRecordNotFound)

	_, err := s.Update(context.Background(), 1001, dto.UserRequestUpdate{RoleID: "CEO"})

	assert.EqualError(t, err, "user_service: role CEO not found: record not found")
	users.AssertNotCalled(t, "Update")
}

func TestApprovalMatrixServiceUpdate_OnlySentFields(t *testing.T) {
	matrices := mocks.NewIApprovalMatrixRepository(t)
	s := &ApprovalMatrixService{opt: testOptions(), RepositoryApprovalMatrix: matrices}
	off := false
	matrices.On("Update", ctxAny, uint(2), map[string]interface{}{"IsActive": false}).Return(nil)

	assert.NoError(t, s.Update(context.Background(), 2, dto.ApprovalMatrixRequestUpdate{IsActive: &off}))
}

func TestSettingsServiceFlushUserQuota_ResetsAndReportsFullLimit(t *testing.T) {
	users, quota := mocks.NewIUserRepository(t), mocks.NewIQuotaRepository(t)
	s := &SettingsService{opt: testOptions(), RepositoryUser: users, RepositoryQuota: quota}
	users.On("GetByID", ctxAny, int64(1001)).Return(&model.User{UserID: 1001}, nil)
	quota.On("FlushUser", ctxAny, constants.QuotaAIInsight, int64(1001)).Return(nil)
	quota.On("Limit", ctxAny, constants.QuotaAIInsight).Return(int64(50), nil)

	res, err := s.FlushUserQuota(context.Background(), dto.QuotaFlushRequest{UserID: 1001, QuotaType: constants.QuotaAIInsight})

	require.NoError(t, err)
	assert.Equal(t, dto.QuotaFlushResponse{UserID: 1001, QuotaType: "ai_insight", Remaining: 50, Limit: 50}, res)
}

func TestWilayahServiceGetProvinceTree_NotFound(t *testing.T) {
	repo := mocks.NewIWilayahRepository(t)
	s := &WilayahService{opt: testOptions(), RepositoryWilayah: repo}
	repo.On("GetProvinceByCode", ctxAny, "99").Return(nil, gorm.ErrRecordNotFound)

	_, err := s.GetProvinceTree(context.Background(), "99")

	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
