package repositoryimpl

import (
	postgres "github.com/opensourceways/server-common-lib/postgre"

	"github.com/opensourceways/defect-manager/defect/domain"
	"github.com/opensourceways/defect-manager/defect/domain/repository"
)

const (
	fieldOrg       = "org"
	fieldNumber    = "number"
	fieldStatus    = "status"
	fieldCreatedAt = "created_at"
)

var instance repository.DefectRepository

var defectTableName string

func Init(cfg *Config) error {
	defectTableName = cfg.Table.Defect

	impl := defectImpl{postgres.NewDBTable(cfg.Table.Defect)}

	instance = impl

	err := impl.db.AutoMigrate(defectDO{})

	return err
}

func Instance() repository.DefectRepository {
	return instance
}

type defectImpl struct {
	db dbimpl
}

func (impl defectImpl) HasDefect(issue *domain.Issue) (domain.Defect, bool, error) {
	filter := defectDO{
		Number: issue.Number,
		Org:    issue.Org,
	}

	var result defectDO
	err := impl.db.GetRecord(&filter, &result)
	if err != nil {
		if impl.db.IsRowNotFound(err) {
			err = nil
		}

		return result.toDefect(), false, err
	}

	return result.toDefect(), true, nil
}

func (impl defectImpl) AddDefect(defect *domain.Defect) error {
	do := impl.toDefectDO(defect)
	return impl.db.Insert(&do)
}

func (impl defectImpl) SaveDefect(defect *domain.Defect) error {
	do := impl.toDefectDO(defect)
	filter := defectDO{
		Number: defect.Issue.Number,
		Org:    defect.Issue.Org,
	}

	return impl.db.UpdateRecord(filter, &do)
}

func (impl defectImpl) FindDefects(opt repository.OptToFindDefects) (ds domain.Defects, err error) {
	var filter []postgres.ColumnFilter
	//filter = append(filter, postgres.NewGreaterFilter(fieldCreatedAt, opt.BeginTime))

	if len(opt.Number) > 0 {
		filter = append(filter, postgres.NewInFilter(fieldNumber, opt.Number))
	}

	if opt.Org != "" {
		filter = append(filter, postgres.NewEqualFilter(fieldOrg, opt.Org))
	}

	if opt.Status != nil {
		filter = append(filter, postgres.NewEqualFilter(fieldStatus, opt.Status.String()))
	}

	var dos []defectDO
	err = impl.db.GetRecords(
		filter, &dos,
		postgres.Pagination{},
		[]postgres.SortByColumn{
			{Column: fieldCreatedAt},
		})
	if err != nil {
		return
	}

	ds = make(domain.Defects, len(dos))
	for k, d := range dos {
		ds[k] = d.toDefect()
	}

	return
}
