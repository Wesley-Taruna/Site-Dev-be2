package constants

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Site ID format (Wes, 27 Oct): "SD" + dd/mm/yyyy (Asia/Jakarta) + daily
// running number of at least 3 digits, e.g. "SD27/10/2026001"; the 1000th
// site of one day is "SD27/10/20261000" (the number simply grows, 16
// characters, still fits Sites.SiteID varchar(20)). Sites made before keep
// their "ST" + yyyymmdd + 6 digits ID; both kinds work everywhere.
//
// The "/" cannot travel as is inside a URL path segment. Clients put the
// URL-safe twin "SD27-10-2026001" ("/" → "-") in paths (SiteIDForPath); a
// "%2F"-encoded ID ("SD27%2F10%2F2026001") is accepted too. Every handler
// reads path IDs through SiteIDFromPath, so the service always sees
// "SD27/10/2026001".
const (
	SiteIDNewPrefix   = "SD"
	SiteIDDateLayout  = "02/01/2006"
	SiteIDNumberWidth = 3
)

// FormatSiteID is the site ID of the n-th site of day (day in Asia/Jakarta).
func FormatSiteID(day time.Time, n int64) string {
	return fmt.Sprintf("%s%s%0*d", SiteIDNewPrefix, day.Format(SiteIDDateLayout), SiteIDNumberWidth, n)
}

// IsSiteID: a site ID of either format ("SD…" new, "ST…" old); a task ID
// starts with "TS".
func IsSiteID(id string) bool {
	return strings.HasPrefix(id, SiteIDNewPrefix) || strings.HasPrefix(id, SiteIDPrefix)
}

// siteIDPathTwin is the URL-safe twin of a new site ID: SDdd-mm-yyyyNNN.
var siteIDPathTwin = regexp.MustCompile(`^SD(\d{2})-(\d{2})-(\d{4}\d{3,})$`)

// SiteIDFromPath turns an ID read from a URL path (or a query parameter)
// into the stored ID: "%2F" decoded, the twin "SD27-10-2026001" back to
// "SD27/10/2026001". Old IDs and task IDs come back unchanged.
func SiteIDFromPath(raw string) string {
	id := strings.TrimSpace(raw)
	if strings.Contains(id, "%") {
		if dec, err := url.PathUnescape(id); err == nil {
			id = dec
		}
	}
	if m := siteIDPathTwin.FindStringSubmatch(id); m != nil {
		return SiteIDNewPrefix + m[1] + "/" + m[2] + "/" + m[3]
	}
	return id
}

// SiteIDForPath is the URL-safe twin of id for use in URL paths
// ("SD27/10/2026001" → "SD27-10-2026001"); old IDs come back unchanged.
func SiteIDForPath(id string) string {
	return strings.ReplaceAll(id, "/", "-")
}
