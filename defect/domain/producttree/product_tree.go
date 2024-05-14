package producttree

import (
	"time"

	"github.com/opensourceways/defect-manager/defect/domain"
	"github.com/opensourceways/defect-manager/defect/domain/dp"
)

type ProductTree interface {
	InitCache()
	CleanCache()
	GetTree(defectTime time.Time, component string, version []dp.SystemVersion) (domain.ProductTree, error)
}
