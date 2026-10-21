package service

import (
	"math"
	"net/http"

	"site-dev-backend/internal/pkg/geo"

	"github.com/labstack/echo/v4"
)

// haversineM / boundingBox: see internal/pkg/geo.
var (
	haversineM  = geo.HaversineM
	boundingBox = geo.BoundingBox
)

// validatePointInIndonesia: 400 "Koordinat tidak valid (harus di
// Indonesia)." outside a rough box around Indonesia or at 0,0.
func validatePointInIndonesia(lat float64, lng float64) error {
	if math.IsNaN(lat) || math.IsNaN(lng) || lat < -11.2 || lat > 6.3 || lng < 94.7 || lng > 141.1 || (lat == 0 && lng == 0) {
		return echo.NewHTTPError(http.StatusBadRequest, "Koordinat tidak valid (harus di Indonesia).")
	}
	return nil
}
