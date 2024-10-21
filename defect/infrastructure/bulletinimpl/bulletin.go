package bulletinimpl

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"

	"github.com/opensourceways/defect-manager/defect/domain"
	"github.com/opensourceways/defect-manager/defect/domain/dp"
	"github.com/opensourceways/defect-manager/utils"
)

var instance *bulletinImpl

func Init(cfg *Config) {
	instance = &bulletinImpl{
		cfg: cfg,
	}
}

func Instance() *bulletinImpl {
	return instance
}

type bulletinImpl struct {
	cfg *Config
}

func (impl bulletinImpl) Generate(sb *domain.SecurityBulletin) ([]byte, error) {
	data := CvrfBA{
		Xmlns:              impl.cfg.Xmlns,
		XmlnsCvrf:          impl.cfg.XmlnsCvrf,
		DocumentTitle:      impl.documentTitle(sb),
		DocumentType:       "Security Advisory",
		DocumentPublisher:  impl.documentPublisher(),
		DocumentTracking:   impl.documentTracking(sb),
		DocumentNotes:      impl.documentNotes(sb),
		DocumentReferences: impl.documentReferences(sb),
		ProductTree:        impl.productTree(sb),
		Vulnerability:      impl.vulnerability(sb),
	}

	return xml.MarshalIndent(data, "", "\t")
}

func (impl bulletinImpl) joinVersion(sb *domain.SecurityBulletin) string {
	var title string
	for _, v := range sb.UnPublishedVersion {
		title += v.String() + ","
	}

	return strings.Trim(title, ",")
}

func (impl bulletinImpl) documentTitle(sb *domain.SecurityBulletin) DocumentTitle {
	title := fmt.Sprintf("openEuler Bug Fix Advisory: %s update for %s",
		sb.Component, impl.joinVersion(sb),
	)
	return DocumentTitle{
		XmlLang:       "en",
		DocumentTitle: title,
	}
}

func (impl bulletinImpl) documentPublisher() DocumentPublisher {
	return DocumentPublisher{
		Type:             "Vendor",
		ContactDetails:   impl.cfg.ContactDetails,
		IssuingAuthority: impl.cfg.IssuingAuthority,
	}
}

func (impl bulletinImpl) documentTracking(sb *domain.SecurityBulletin) DocumentTracking {
	return DocumentTracking{
		Identification: Identification{
			Id: sb.Identification,
		},
		Status:  "Final",
		Version: "1.0",
		RevisionHistory: RevisionHistory{
			Revision: []Revision{{
				Number:      "1.0",
				Date:        sb.Date,
				Description: "Initial",
			}},
		},
		InitialReleaseDate: sb.Date,
		CurrentReleaseDate: sb.Date,
		Generator: Generator{
			Engine: "openEuler BA Tool V1.0",
			Date:   sb.Date,
		},
	}
}

func (impl bulletinImpl) documentNotes(sb *domain.SecurityBulletin) DocumentNotes {
	var description string
	var highestLevelIndex int

	for _, defect := range sb.Defects {
		description += fmt.Sprintf("%s(%s)\r\n\r\n", defect.Influence, impl.bugID(defect.Issue.Number))
		// Choose the highest security level in defects, as security level in bulletin
		for k, v := range dp.SequenceSeverityLevel {
			if v == defect.SeverityLevel.String() && k > highestLevelIndex {
				highestLevelIndex = k
			}
		}
	}

	return DocumentNotes{
		Note: []Note{
			{
				Title:   "Synopsis",
				Type:    "General",
				Ordinal: "1",
				XmlLang: "en",
				Note:    fmt.Sprintf("%s bug update", sb.Component),
			},
			{
				Title:   "Summary",
				Type:    "General",
				Ordinal: "2",
				XmlLang: "en",
				Note:    fmt.Sprintf("An update for %s is now available for %s", sb.Component, impl.joinVersion(sb)),
			},
			{
				Title:   "Description",
				Type:    "General",
				Ordinal: "3",
				XmlLang: "en",
				Note:    strings.Trim(description, "\r\n\r\n"),
			},
			{
				Title:   "Severity",
				Type:    "General",
				Ordinal: "5",
				XmlLang: "en",
				Note:    dp.SequenceSeverityLevel[highestLevelIndex],
			},
			{
				Title:   "Affected Component",
				Type:    "General",
				Ordinal: "6",
				XmlLang: "en",
				Note:    sb.Component,
			},
		},
	}
}

func (impl bulletinImpl) documentReferences(sb *domain.SecurityBulletin) DocumentReferences {
	selfUrl := []CveUrl{
		{
			Url: impl.cfg.SecurityBulletinUrlPrefix + sb.Identification,
		},
	}

	var defectUrl, referenceUrl []CveUrl
	for _, defect := range sb.Defects {
		url := fmt.Sprintf("https://gitee.com/%s/%s/issues/%s",
			defect.Issue.Org, defect.Issue.Repo, defect.Issue.Number,
		)
		defectUrl = append(defectUrl, CveUrl{Url: url})

		if defect.ReferenceURL.URL() != "" {
			referenceUrl = append(referenceUrl, CveUrl{Url: defect.ReferenceURL.URL()})
		}
	}

	df := DocumentReferences{
		CveReference: []CveReference{
			{
				Type:   "Self",
				CveUrl: selfUrl,
			},
			{
				Type:   "openEuler Bugfix",
				CveUrl: defectUrl,
			},
		},
	}

	if len(referenceUrl) > 0 {
		df.CveReference = append(df.CveReference, CveReference{
			Type:   "other",
			CveUrl: referenceUrl,
		})
	}

	return df
}

func (impl bulletinImpl) productTree(sb *domain.SecurityBulletin) ProductTree {
	getCpe := func(v string) string {
		t := strings.Split(v, "-")
		return fmt.Sprintf("cpe:/a:%v:%v:%v", t[0], t[0], strings.Join(t[1:], "-"))
	}

	var productOfVersion []FullProductName
	for _, v := range sb.UnPublishedVersion {
		productOfVersion = append(productOfVersion, FullProductName{
			ProductId:       v.String(),
			Cpe:             getCpe(v.String()),
			FullProductName: v.String(),
		})
	}

	branchOfVersion := OpenEulerBranch{
		Type:            "Product Name",
		Name:            "openEuler",
		FullProductName: productOfVersion,
	}

	branches := []OpenEulerBranch{
		branchOfVersion,
	}

	for arch, products := range sb.ProductTree {
		var productOfArch []FullProductName
		for _, p := range products {
			productOfArch = append(productOfArch, FullProductName{
				ProductId:       p.ID,
				Cpe:             getCpe(p.CPE),
				FullProductName: p.FullName,
			})
		}

		branch := OpenEulerBranch{
			Type:            "Package Arch",
			Name:            arch.String(),
			FullProductName: productOfArch,
		}

		branches = append(branches, branch)
	}

	return ProductTree{
		Xmlns:           impl.cfg.Xmlns,
		OpenEulerBranch: branches,
	}
}

func (impl bulletinImpl) vulnerability(sb *domain.SecurityBulletin) []Vulnerability {
	var vs []Vulnerability

	for k, defect := range sb.Defects {
		var idOfStatus []ProductId
		for _, v := range sb.UnPublishedVersion {
			idOfStatus = append(idOfStatus, ProductId{
				ProductId: v.String(),
			})
		}

		vul := Vulnerability{
			Ordinal: strconv.Itoa(k + 1),
			Xmlns:   impl.cfg.Xmlns,
			CveNotes: CveNotes{
				CveNote: CveNote{
					Title:   "Vulnerability Description",
					Type:    "General",
					Ordinal: "1",
					XmlLang: "en",
					Note:    defect.Description,
				},
			},
			ReleaseDate: sb.Date,
			Bug:         impl.bugID(defect.Issue.Number),
			ProductStatuses: ProductStatuses{
				Status: Status{
					Type:      "Fixed",
					ProductId: idOfStatus,
				},
			},
			Threats: Threats{
				Threat: Threat{
					Type:        "Impact",
					Description: defect.SeverityLevel.String(),
				},
			},
			Remediations: Remediations{
				Remediation: Remediation{
					Type:        "Vendor Fix",
					Description: fmt.Sprintf("%s bug update", sb.Component),
					Date:        sb.Date,
					Url:         impl.cfg.SecurityBulletinUrlPrefix + sb.Identification,
				},
			},
		}

		vs = append(vs, vul)
	}

	return vs
}

func (impl bulletinImpl) bugID(issueNumber string) string {
	return fmt.Sprintf("BUG-%d-%s", utils.Year(), issueNumber)
}
