package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/opensourceways/defect-manager/defect/domain"
	"github.com/opensourceways/defect-manager/defect/domain/backend"
	"github.com/opensourceways/defect-manager/defect/domain/bulletin"
	"github.com/opensourceways/defect-manager/defect/domain/dp"
	"github.com/opensourceways/defect-manager/defect/domain/obs"
	"github.com/opensourceways/defect-manager/defect/domain/producttree"
	"github.com/opensourceways/defect-manager/defect/domain/repository"
	"github.com/opensourceways/defect-manager/defect/infrastructure/producttreeimpl"
	"github.com/opensourceways/defect-manager/utils"
)

const uploadedDefect = "update_defect.txt"

type DefectService interface {
	IsDefectExist(*domain.Issue) (domain.Defect, bool, error)
	SaveDefects(CmdToSaveDefect) error
	CollectDefects(string) ([]CollectDefectsDTO, error)
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

func (d defectService) IsDefectExist(issue *domain.Issue) (domain.Defect, bool, error) {
	return d.repo.HasDefect(issue)
}

func (d defectService) SaveDefects(cmd CmdToSaveDefect) error {
	_, has, err := d.repo.HasDefect(&cmd.Issue)
	if err != nil {
		return err
	}

	if has {
		return d.repo.SaveDefect(&cmd)
	} else {
		return d.repo.AddDefect(&cmd)
	}
}

func (d defectService) CollectDefects(version string) (dto []CollectDefectsDTO, err error) {
	defer utils.Catchs()

	defects, err := d.repo.FindDefects(repository.OptToFindDefects{})
	if err != nil || len(defects) == 0 {
		return
	}

	var versionForDefects domain.Defects
	for _, d := range defects {
		for _, av := range d.FixedVersion {
			if av == nil {
				continue
			}
			logrus.Infof("FixedVersion is %v, av: %v", d.FixedVersion, av)
			if av.String() == version {
				versionForDefects = append(versionForDefects, d)
			}
		}
	}

	d.productTree.InitCache()
	defer d.productTree.CleanCache()

	var rpmForDefects domain.Defects
	instance := producttreeimpl.Instance()
	for _, vdf := range versionForDefects {
		rpmOfComponent := instance.ParseRPM(vdf.UpdatedAt, vdf.Component, version)
		if rpmOfComponent != "" {
			rpmForDefects = append(rpmForDefects, vdf)
		}
	}

	issuesNumAndVersions, err := d.backend.PublishedDefects()
	if err != nil {
		logrus.Errorf("get published defect error: %s", err.Error())
		return
	}

	issueMap := make(map[string][]string)
	for _, issue := range issuesNumAndVersions {
		issueMap[issue.IssueNum] = append(issueMap[issue.IssueNum], issue.Versions...)
	}

	var unpublishedDefects domain.Defects
	for _, rfd := range rpmForDefects {
		if _, ok := issueMap[rfd.Issue.Number]; !ok {
			unpublishedDefects = append(unpublishedDefects, rfd)
			rfd.UnPublishedVersion = rfd.FixedVersion
			if err = d.SaveDefects(rfd); err != nil {
				logrus.Errorf("when collect defects, save defect error: %s", err.Error())
			}
			continue
		}

		if len(issueMap[rfd.Issue.Number]) != len(rfd.FixedVersion) {
			unpublishedDefects = append(unpublishedDefects, rfd)
			unPubishedVersion := d.findUnPubishedVersion(rfd.FixedVersion, issueMap[rfd.Issue.Number])
			rfd.UnPublishedVersion = unPubishedVersion
			if err = d.SaveDefects(rfd); err != nil {
				logrus.Errorf("when collect defects, save defect error: %s", err.Error())
			}
			continue
		}

		rfd.UnPublishedVersion = []dp.SystemVersion{}
		if err = d.SaveDefects(rfd); err != nil {
			logrus.Errorf("when collect defects, save defect error: %s", err.Error())
		}
	}

	dto = ToCollectDefectsDTO(unpublishedDefects)

	return
}

func (d defectService) GenerateBulletins(number []string) error {
	defer utils.Catchs()

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
		b.Identification = fmt.Sprintf("openEuler-BA-%d-%d", utils.Year(), maxIdentification)

		b.ProductTree, err = d.productTree.GetTree(b.Defects[0].CreatedAt, b.Component, b.UnPublishedVersion)
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

// 对比已修复的版本和已发布的版本，筛选出未发布的版本
func (d defectService) findUnPubishedVersion(fixedVersions []dp.SystemVersion, publishedVersions []string) []dp.SystemVersion {
	unPubishedVersion := make([]dp.SystemVersion, 0)

	// 创建一个 map 用于快速查找 publishedVersion 中的元素
	publishedVersionMap := make(map[dp.SystemVersion]bool)
	for _, element := range publishedVersions {
		sv, err := dp.NewSystemVersion(element)
		if err != nil {
			continue
		}
		publishedVersionMap[sv] = true
	}

	// 遍历 fixedVersion，检查每个元素是否在 publishedVersion 中
	for _, element := range fixedVersions {
		if !publishedVersionMap[element] {
			unPubishedVersion = append(unPubishedVersion, element)
		}
	}

	return unPubishedVersion
}
