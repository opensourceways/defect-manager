package domain

import (
	"time"

	"github.com/opensourceways/defect-manager/defect/domain/dp"

	"github.com/opensourceways/defect-manager/utils"
)

type Defects []Defect

// DefectsByComponent is group of defects by component
type DefectsByComponent []Defect

// DefectsByVersion is group of DefectsByComponent by version
type DefectsByVersion []Defect

type Defect struct {
	Kernel             string
	Component          string
	ComponentVersion   string
	SystemVersion      dp.SystemVersion
	Description        string
	ReferenceURL       dp.URL
	GuidanceURL        dp.URL
	Influence          string
	SeverityLevel      dp.SeverityLevel
	RootCause          string
	AffectedVersion    []dp.SystemVersion
	FixedVersion       []dp.SystemVersion
	UnPublishedVersion []dp.SystemVersion
	ABI                string
	Issue              Issue
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type Issue struct {
	Title  string
	Number string
	Org    string
	Repo   string
	Status dp.IssueStatus
}

func (d Defect) isUnPublishedVersion(version dp.SystemVersion) bool {
	for _, v := range d.UnPublishedVersion {
		if v == version {
			return true
		}
	}

	return false
}

// GroupByComponent group defects by component
func (ds Defects) groupByComponent() map[string]DefectsByComponent {
	group := make(map[string]DefectsByComponent)
	for _, d := range ds {
		group[d.Component] = append(group[d.Component], d)
	}

	return group
}

// GenerateBulletins DefectsByComponent is a component-differentiated set of defects,
// Bulletins are consolidated into one when all issues of a component affect all versions currently maintained,
// otherwise they are split into multiple bulletins by version
func (ds Defects) GenerateBulletins() []SecurityBulletin {
	var securityBulletins []SecurityBulletin

	for _, dsc := range ds.groupByComponent() {
		if dsc.isCombined() {
			securityBulletins = append(securityBulletins, dsc.combinedBulletin())
		} else {
			securityBulletins = append(securityBulletins, dsc.separatedBulletins()...)
		}
	}

	return securityBulletins
}

// IsCombined determine whether multiple defects under the same component
// need to be combined into a single bulletin
func (dsc DefectsByComponent) isCombined() bool {
	for _, d := range dsc {
		if len(d.UnPublishedVersion) != len(dp.MaintainVersion) {
			return false
		}

		for _, version := range d.UnPublishedVersion {
			if !dp.MaintainVersion[version] {
				return false
			}
		}
	}

	return true
}

// CombinedBulletin put all defects in one bulletin
func (dsc DefectsByComponent) combinedBulletin() SecurityBulletin {
	return SecurityBulletin{
		UnPublishedVersion: dsc[0].UnPublishedVersion,
		Date:               utils.Date(),
		Component:          dsc[0].Component,
		Defects:            Defects(dsc),
	}
}

// SeparatedBulletins split into multiple bulletins by version
func (dsc DefectsByComponent) separatedBulletins() []SecurityBulletin {
	var sbs []SecurityBulletin
	for version, ds := range dsc.separateByVersion() {
		sbs = append(sbs, ds.bulletinByVersion(version))
	}

	return sbs
}

func (dsc DefectsByComponent) separateByVersion() map[dp.SystemVersion]DefectsByVersion {
	classifyByVersion := make(map[dp.SystemVersion]DefectsByVersion)
	for version := range dp.MaintainVersion {
		for _, d := range dsc {
			if d.isUnPublishedVersion(version) {
				classifyByVersion[version] = append(classifyByVersion[version], d)
			}
		}
	}

	return classifyByVersion
}

func (dsv DefectsByVersion) bulletinByVersion(version dp.SystemVersion) SecurityBulletin {
	return SecurityBulletin{
		UnPublishedVersion: []dp.SystemVersion{version},
		Date:               utils.Date(),
		Component:          dsv[0].Component,
		Defects:            Defects(dsv),
	}
}
