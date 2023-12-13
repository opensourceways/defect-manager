package repository

import (
	"time"

	"github.com/opensourceways/defect-manager/defect/domain/dp"

	"github.com/opensourceways/defect-manager/defect/domain"
)

type OptToFindDefects struct {
	BeginTime time.Time
	Org       string
	Number    []string
	Status    dp.IssueStatus
}

type DefectRepository interface {
	HasDefect(*domain.Issue) (bool, error)
	AddDefect(*domain.Defect) error
	SaveDefect(*domain.Defect) error
	FindDefects(OptToFindDefects) (domain.Defects, error)
}
