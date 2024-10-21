package domain

import (
	"github.com/opensourceways/defect-manager/defect/domain/dp"
)

type SecurityBulletin struct {
	UnPublishedVersion []dp.SystemVersion
	Identification     string
	Date               string
	Component          string
	ProductTree        ProductTree
	Defects            Defects
}

type ProductTree = map[dp.Arch][]Product

type Product struct {
	ID       string
	CPE      string
	FullName string
}
