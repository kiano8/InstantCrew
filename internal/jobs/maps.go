package jobs

import (
	"fmt"
	"net/url"
	"strings"
)

// buildMapsUrl prefers exact coordinates
func buildMapsURL(loc string, lat, lng *float64) string {
	if lat != nil && lng != nil {
		return fmt.Sprintf("https://www.google.com/maps/search/?api=1&query=%.6f,%.6f", *lat, *lng)
	}
	if u, err := url.Parse(loc); err == nil && u.Scheme == "https" && isGoogleMaps(u) {
		return u.String()
	}
	return "https://www.google.com/maps/search/?api=1&query=" + url.QueryEscape(loc)
}

func isGoogleMaps(u *url.URL) bool {
	switch strings.ToLower(u.Host) {
	case "maps.app.goo.gl", "maps.google.com":
		return true
	case "www.google.com", "google.com", "goo.gl":
		return strings.HasPrefix(u.Path, "/maps")
	}

	return false
}
