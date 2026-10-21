// Package geo has the distance helpers shared by the nearby features.
package geo

import "math"

// earthRadiusM is the mean Earth radius.
const earthRadiusM = 6371000.0

// HaversineM is the distance in metres between two points.
func HaversineM(lat1 float64, lng1 float64, lat2 float64, lng2 float64) float64 {
	const rad = math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLng := (lng2 - lng1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthRadiusM * math.Asin(math.Sqrt(a))
}

// BoundingBox is the square (minLat, maxLat, minLng, maxLng) around a
// circle: a cheap pre-filter before the exact distance.
func BoundingBox(lat float64, lng float64, radiusM float64) (float64, float64, float64, float64) {
	dLat := radiusM / 111320
	dLng := radiusM / (111320 * math.Cos(lat*math.Pi/180))
	return lat - dLat, lat + dLat, lng - dLng, lng + dLng
}
