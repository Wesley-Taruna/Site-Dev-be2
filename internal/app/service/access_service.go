package service

import (
	"context"
	"net/http"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/constants"
	"site-dev-backend/internal/app/model"
	"site-dev-backend/internal/app/repository"

	"github.com/labstack/echo/v4"
	"gitlab.com/fami-app-v2/familib/familabs/log"
	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

// IAccessService holds the role checks that come before anything else in a
// handler (site-dev-api's requireAdmin). The role is Users.RoleID; how the
// caller is identified depends on the token question (plan Q1b).
type IAccessService interface {
	// RequireAdmin: unknown user 401, not ADMIN 403 "hanya admin yang bisa
	// mengakses fitur ini".
	RequireAdmin(ctx context.Context, userID int64) (err error)
}

type AccessService struct {
	opt            opt.Options
	RepositoryUser repository.IUserRepository
}

func NewAccessService() IAccessService {
	return &AccessService{
		opt:            global.Setup.Common,
		RepositoryUser: repository.NewUserRepository(),
	}
}

func (s *AccessService) RequireAdmin(ctx context.Context, userID int64) (err error) {
	ctx, span := s.opt.Trace.Start(ctx, "AccessService.RequireAdmin")
	defer span.End()

	var user *model.User
	user, err = getUser(ctx, s.RepositoryUser, userID)
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		return
	}
	if user.RoleID != constants.RoleAdmin {
		err = echo.NewHTTPError(http.StatusForbidden, "hanya admin yang bisa mengakses fitur ini")
		span.RecordError(err)
		return
	}

	return
}
