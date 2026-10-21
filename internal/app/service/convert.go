package service

import (
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/model"
)

// toSiteResponse is site-dev-api's dto.ToSiteResp.
func toSiteResponse(m *model.Site) dto.SiteResponseGet {
	return dto.SiteResponseGet{
		SiteID:                 m.SiteID,
		SiteName1:              m.SiteName1,
		SiteName2:              m.SiteName2,
		SiteAddress1:           m.SiteAddress1,
		SiteAddress2:           m.SiteAddress2,
		SiteAddress3:           m.SiteAddress3,
		SitePostalCode:         m.SitePostalCode,
		SiteDistrict:           m.SiteDistrict,
		SiteCity:               m.SiteCity,
		SiteRegion:             m.SiteRegion,
		SiteLat:                m.SiteLat,
		SiteLon:                m.SiteLon,
		SiteTelepon:            m.SiteTelepon,
		SiteTeleponExt:         m.SiteTeleponExt,
		SiteMobilePhone:        m.SiteMobilePhone,
		SitePic:                m.SitePic,
		SiteFax:                m.SiteFax,
		SitePlannedOpeningDate: dto.DateOnly{Time: m.SitePlannedOpeningDate},
		SiteSalesArea:          m.SiteSalesArea,
		SiteSellingAlcohol:     m.SiteSellingAlcohol,
		SiteOpen24Hours:        m.SiteOpen24Hours,
		CreatedBy:              m.CreatedBy,
		UpdatedBy:              m.UpdatedBy,
		CreatedAt:              m.CreatedAt,
		UpdatedAt:              m.UpdatedAt,
	}
}

// toSiteContextResponse is site-dev-api's dto.ToSiteContextResp (lists
// stay null when empty, as there).
func toSiteContextResponse(m *model.SiteContext, competitors []*model.SiteCompetitor, notes []*model.SiteCustomNote) dto.SiteContextResponseGet {
	res := dto.SiteContextResponseGet{
		SiteID:                 m.SiteID,
		FootTrafficObservation: m.FootTrafficObservation,
		PeakHours:              m.PeakHours,
		PublicTransport:        m.PublicTransport,
		AnchorLandmark:         m.AnchorLandmark,
		ObservedDemographics:   m.ObservedDemographics,
		CreatedAt:              m.CreatedAt,
		UpdatedAt:              m.UpdatedAt,
	}
	if m.ParkingAvailability != nil {
		res.ParkingAvailability = *m.ParkingAvailability
	}
	for _, c := range competitors {
		res.Competitors = append(res.Competitors, dto.CompetitorResponseGet{
			ID:             c.ID,
			CompetitorName: c.CompetitorName,
			CompetitorType: c.CompetitorType,
			Distance:       c.Distance,
		})
	}
	for _, n := range notes {
		res.CustomNotes = append(res.CustomNotes, dto.CustomNoteResponseGet{
			ID:      n.ID,
			Title:   n.Title,
			Content: n.Content,
		})
	}
	return res
}

func toTaskHistoryResponse(m *model.TaskHistory) dto.TaskHistoryResponseGet {
	return dto.TaskHistoryResponseGet{
		ID:        m.ID,
		Action:    m.Action,
		ActorID:   m.ActorID,
		Notes:     m.Notes,
		CreatedAt: m.CreatedAt,
	}
}

// toTaskResponse is site-dev-api's dto.ToTaskResp (site may be nil).
func toTaskResponse(task *model.Task, site *model.Site) dto.TaskResponseGet {
	row := dto.TaskResponseGet{
		TaskID:     task.TaskID,
		SiteID:     task.SiteID,
		TaskStatus: task.TaskStatus,
		CreatedBy:  task.CreatedBy,
		UpdatedBy:  task.UpdatedBy,
		CreatedAt:  task.CreatedAt,
		UpdatedAt:  task.UpdatedAt,
	}
	if site != nil {
		row.SiteName1 = site.SiteName1
		row.SiteAddress1 = site.SiteAddress1
		row.SiteAddress2 = site.SiteAddress2
		row.SiteAddress3 = site.SiteAddress3
	}
	return row
}

func toAttachmentResponses(attachments []*model.SiteAttachment, uploadedBy int64) []dto.TaskAttachmentResponseGet {
	var res []dto.TaskAttachmentResponseGet
	for _, a := range attachments {
		res = append(res, dto.TaskAttachmentResponseGet{
			ID:         a.SiteAttachmentID,
			FileName:   a.FileName,
			FileURL:    a.Path,
			UploadedBy: uploadedBy,
			UploadedAt: a.CreatedAt,
		})
	}
	return res
}
