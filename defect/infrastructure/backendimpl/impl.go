package backendimpl

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/opensourceways/server-common-lib/utils"

	"github.com/opensourceways/defect-manager/defect/domain/backend"
	localutils "github.com/opensourceways/defect-manager/utils"
)

var regOfBulletinID = regexp.MustCompile(`openEuler-BA-(\d{4})-(\d{4,5})`)

var instance *backendImpl

func Init(cfg *Config) {
	instance = &backendImpl{
		cli: utils.NewHttpClient(3),
		cfg: cfg,
	}
}

func Instance() *backendImpl {
	return instance
}

type backendImpl struct {
	cli utils.HttpClient
	cfg *Config
}

type maxIdResult struct {
	Code   int    `json:"code"`
	Result string `json:"result"`
	Msg    string `json:"msg"`
}

type publishedDefectResult struct {
	Code   int                          `json:"code"`
	Result []backend.IssueNumAndVersion `json:"result"`
	Msg    string                       `json:"msg"`
}

func (impl backendImpl) MaxBulletinID() (maxId int, err error) {
	url := fmt.Sprintf("%s/cve-security-notice-server/securitynotice/getMaxNoticeId?notice_type=bug",
		impl.cfg.Endpoint,
	)
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return
	}

	r, _, err := impl.cli.Download(request)
	if err != nil {
		return
	}

	var res maxIdResult
	if err = json.Unmarshal(r, &res); err != nil {
		return
	}

	if res.Code != 0 {
		err = errors.New(res.Msg)

		return
	}

	// init id
	if res.Result == "" {
		return 1000, nil
	}

	match := regOfBulletinID.FindAllStringSubmatch(res.Result, -1)
	if len(match) == 0 {
		err = errors.New("invalid bulletin id")

		return
	}

	// reset id to 1000 at new year
	if match[0][1] != strconv.Itoa(localutils.Year()) {
		return 1000, nil
	}

	return strconv.Atoi(match[0][2])
}

// PublishedDefects get all published defects and corresponding versions info
func (impl backendImpl) PublishedDefects() (iv []backend.IssueNumAndVersion, err error) {
	url := fmt.Sprintf("%s/cve-security-notice-server/securitynotice/getPublishedBugs",
		impl.cfg.Endpoint,
	)
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return
	}

	r, _, err := impl.cli.Download(request)
	if err != nil {
		return
	}

	var res publishedDefectResult
	if err = json.Unmarshal(r, &res); err != nil {
		return
	}

	if res.Code != 0 {
		err = errors.New(res.Msg)

		return
	}

	return res.Result, nil
}
