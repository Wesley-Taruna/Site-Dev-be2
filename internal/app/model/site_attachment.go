package model

import (
	"encoding/json"
	"time"
)

// SiteAttachment is one uploaded file (photo) of a site.
type SiteAttachment struct {
	SiteAttachmentID uint      `gorm:"column:SiteAttachmentID;primaryKey" json:"site_attachment_id"`
	SiteID           string    `gorm:"column:SiteID" json:"site_id"`
	FileName         string    `gorm:"column:FileName" json:"file_name"`
	Path             string    `gorm:"column:Path" json:"path"`
	CreatedAt        time.Time `gorm:"column:CreatedAt" json:"created_at"`
}

func (SiteAttachment) TableName() string {
	return "SiteAttachments"
}

func (m *SiteAttachment) MarshalJSON() ([]byte, error) {
	type Alias SiteAttachment

	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(m),
	})
}
