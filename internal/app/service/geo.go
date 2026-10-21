package service

import (
	"math"
	"net/http"

	"github.com/labstack/echo/v4"
)

// earthRadiusM is the mean Earth radius.
const earthRadiusM = 6371000.0

// haversineM is the distance in metres between two points.
func haversineM(lat1 float64, lng1 float64, lat2 float64, lng2 float64) float64 {
	const rad = math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLng := (lng2 - lng1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthRadiusM * math.Asin(math.Sqrt(a))
}

// boundingBox is the square (minLat, maxLat, minLng, maxLng) around a
// circle: a cheap pre-filter before the exact distance.
func boundingBox(lat float64, lng float64, radiusM float64) (float64, float64, float64, float64) {
	dLat := radiusM / 111320
	dLng := radiusM / (111320 * math.Cos(lat*math.Pi/180))
	return lat - dLat, lat + dLat, lng - dLng, lng + dLng
}

// validatePointInIndonesia: 400 "Koordinat tidak valid (harus di
// Indonesia)." outside a rough box around Indonesia or at 0,0.
func validatePointInIndonesia(lat float64, lng float64) error {
	if math.IsNaN(lat) || math.IsNaN(lng) || lat < -11.2 || lat > 6.3 || lng < 94.7 || lng > 141.1 || (lat == 0 && lng == 0) {
		return echo.NewHTTPError(http.StatusBadRequest, "Koordinat tidak valid (harus di Indonesia).")
	}
	return nil
}
