package repositoryimpl

import (
	"time"

	"github.com/lib/pq"

	"github.com/opensourceways/defect-manager/defect/domain"
	"github.com/opensourceways/defect-manager/defect/domain/dp"
)

type defectDO struct {
	ID               int            `gorm:"column:id;primaryKey;autoIncrement"`
	Number           string         `gorm:"column:number;index"` // Number is the number of issue
	Title            string         `gorm:"column:title"`
	Org              string         `gorm:"column:org"`
	Repo             string         `gorm:"column:repo"`
	Status           string         `gorm:"column:status"`
	Kernel           string         `gorm:"column:kernel"`
	Component        string         `gorm:"column:component"`
	ComponentVersion string         `gorm:"column:component_version"`
	SystemVersion    string         `gorm:"column:system_version"`
	Description      string         `gorm:"column:description"`
	ReferenceURL     string         `gorm:"column:reference_url"`
	GuidanceURL      string         `gorm:"column:guidance_url"`
	Influence        string         `gorm:"column:influence"`
	SeverityLevel    string         `gorm:"column:severity_level"`
	AffectedVersion  pq.StringArray `gorm:"column:affected_version;type:text[];default:'{}'"`
	ABI              string         `gorm:"column:abi"`
	CreatedAt        time.Time      `gorm:"column:created_at;<-:create;index"`
	UpdatedAt        time.Time      `gorm:"column:updated_at"`
}

func (d defectDO) TableName() string {
	return defectTableName
}

func (impl defectImpl) toDefectDO(defect *domain.Defect) defectDO {
	return defectDO{
		Number:           defect.Issue.Number,
		Title:            defect.Issue.Title,
		Org:              defect.Issue.Org,
		Repo:             defect.Issue.Repo,
		Status:           defect.Issue.Status.String(),
		Kernel:           defect.Kernel,
		Component:        defect.Component,
		ComponentVersion: defect.ComponentVersion,
		SystemVersion:    defect.SystemVersion.String(),
		Description:      defect.Description,
		ReferenceURL:     "",
		GuidanceURL:      "",
		Influence:        defect.Influence,
		SeverityLevel:    defect.SeverityLevel.String(),
		AffectedVersion:  toStringArray(defect.AffectedVersion),
		ABI:              defect.ABI,
	}
}

func toStringArray(versions []dp.SystemVersion) pq.StringArray {
	arr := make(pq.StringArray, len(versions))
	for k, v := range versions {
		arr[k] = v.String()
	}

	return arr
}

func toSystemVersion(arr pq.StringArray) []dp.SystemVersion {
	versions := make([]dp.SystemVersion, len(arr))
	for k, v := range arr {
		dpv, _ := dp.NewSystemVersion(v)
		versions[k] = dpv
	}

	return versions
}

func (d defectDO) toDefect() domain.Defect {
	version, _ := dp.NewSystemVersion(d.SystemVersion)
	severityLevel, _ := dp.NewSeverityLevel(d.SeverityLevel)
	status, _ := dp.NewIssueStatus(d.Status)

	return domain.Defect{
		Kernel:           d.Kernel,
		Component:        d.Component,
		ComponentVersion: d.ComponentVersion,
		SystemVersion:    version,
		Description:      d.Description,
		ReferenceURL:     nil,
		GuidanceURL:      nil,
		Influence:        d.Influence,
		SeverityLevel:    severityLevel,
		AffectedVersion:  toSystemVersion(d.AffectedVersion),
		ABI:              d.ABI,
		Issue: domain.Issue{
			Title:  d.Title,
			Number: d.Number,
			Org:    d.Org,
			Repo:   d.Repo,
			Status: status,
		},
	}
}
