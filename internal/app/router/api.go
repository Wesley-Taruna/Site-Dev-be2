package router

import (
	"site-dev-backend/internal/app/handler"
)

var handlers = map[string]RouteHandlers{}

func init() {
	handlers["health-check"] = &handler.HealthCheckHandler{}
	handlers["person"] = &handler.PersonHandler{}
	handlers["task"] = &handler.TaskHandler{}
	handlers["dashboard/task"] = &handler.DashboardHandler{}
	handlers["site"] = &handler.SiteHandler{}
	handlers["wilayah"] = &handler.WilayahHandler{}
	handlers["roles"] = &handler.RoleHandler{}
	handlers["users"] = &handler.UserHandler{}
	handlers["approval-matrix"] = &handler.ApprovalMatrixHandler{}
	handlers["settings"] = &handler.SettingsHandler{}
	handlers["validation"] = &handler.ValidationHandler{}
	handlers["upload"] = &handler.UploadHandler{}
	handlers["attachment"] = &handler.AttachmentHandler{}
	handlers["queue"] = &handler.QueueHandler{}
	handlers["geocoding"] = &handler.GeocodingHandler{}
	// November scope
	handlers["activity"] = &handler.ActivityHandler{}
}
