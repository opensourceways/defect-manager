package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/opensourceways/defect-manager/defect/domain"
	"github.com/opensourceways/defect-manager/defect/domain/backend"
	"github.com/opensourceways/defect-manager/defect/domain/bulletin"
	"github.com/opensourceways/defect-manager/defect/domain/dp"
	"github.com/opensourceways/defect-manager/defect/domain/obs"
	"github.com/opensourceways/defect-manager/defect/domain/producttree"
	"github.com/opensourceways/defect-manager/defect/domain/repository"
	"github.com/opensourceways/defect-manager/utils"
)

const uploadedDefect = "update_defect.txt"

type DefectService interface {
	IsDefectExist(*domain.Issue) (bool, error)
	SaveDefects(CmdToSaveDefect) error
	CollectDefects(time time.Time) ([]CollectDefectsDTO, error)
	GenerateBulletins([]string) error
}

func NewDefectService(
	r repository.DefectRepository,
	t producttree.ProductTree,
	b bulletin.Bulletin,
	be backend.CveBackend,
	o obs.OBS,
) *defectService {
	return &defectService{
		repo:        r,
		productTree: t,
		bulletin:    b,
		backend:     be,
		obs:         o,
	}
}

type defectService struct {
	repo        repository.DefectRepository
	productTree producttree.ProductTree
	bulletin    bulletin.Bulletin
	backend     backend.CveBackend
	obs         obs.OBS
}

func (d defectService) IsDefectExist(issue *domain.Issue) (bool, error) {
	return d.repo.HasDefect(issue)
}

func (d defectService) SaveDefects(cmd CmdToSaveDefect) error {
	has, err := d.repo.HasDefect(&cmd.Issue)
	if err != nil {
		return err
	}

	if has {
		return d.repo.SaveDefect(&cmd)
	} else {
		return d.repo.AddDefect(&cmd)
	}
}

func (d defectService) CollectDefects(date time.Time) (dto []CollectDefectsDTO, err error) {
	opt := repository.OptToFindDefects{
		BeginTime: date,
		Status:    dp.IssueStatusClosed,
	}

	defects, err := d.repo.FindDefects(opt)
	if err != nil || len(defects) == 0 {
		return
	}

	publishedNum, err := d.backend.PublishedDefects()
	if err != nil {
		return
	}

	var unpublishedDefects domain.Defects
	ps := sets.NewString(publishedNum...)
	for _, defect := range defects {
		if _, ok := ps[defect.Issue.Number]; !ok {
			unpublishedDefects = append(unpublishedDefects, defect)
		}
	}

	dto = ToCollectDefectsDTO(unpublishedDefects)

	return
}

func (d defectService) GenerateBulletins(number []string) error {
	opt := repository.OptToFindDefects{
		Number: number,
	}

	defects, err := d.repo.FindDefects(opt)
	if err != nil {
		return err
	}

	maxIdentification, err := d.backend.MaxBulletinID()
	if err != nil {
		return err
	}

	bulletins := defects.GenerateBulletins()

	d.productTree.InitCache()
	defer d.productTree.CleanCache()

	var uploadedFile []string
	for _, b := range bulletins {
		maxIdentification++
		b.Identification = fmt.Sprintf("cvrf-openEuler-BA-%d-%d", utils.Year(), maxIdentification)

		b.ProductTree, err = d.productTree.GetTree(b.Component, b.AffectedVersion)
		if err != nil {
			logrus.Errorf("%s, component %s, get productTree error: %s", b.Identification, b.Component, err.Error())

			continue
		}

		xmlData, err := d.bulletin.Generate(&b)
		if err != nil {
			logrus.Errorf("%s, component: %s, to xml error: %s", b.Identification, b.Component, err.Error())

			continue
		}

		fileName := fmt.Sprintf("%s.xml", b.Identification)
		if err := d.obs.Upload(fileName, xmlData); err != nil {
			logrus.Errorf("%s, component: %s, upload to obs error: %s", b.Identification, b.Component, err.Error())

			continue
		}

		uploadedFile = append(uploadedFile, fileName)
	}

	return d.uploadUploadedFile(uploadedFile)
}

func (d defectService) uploadUploadedFile(files []string) error {
	if len(files) == 0 {
		return nil
	}

	var uploadedFileWithPrefix []string
	for _, v := range files {
		t := fmt.Sprintf("%d/%s", time.Now().Year(), v)
		uploadedFileWithPrefix = append(uploadedFileWithPrefix, t)
	}

	return d.obs.Upload(uploadedDefect, []byte(strings.Join(uploadedFileWithPrefix, "\n")))
}
