package producttreeimpl

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/opensourceways/robot-gitee-lib/client"
	"github.com/sirupsen/logrus"

	"github.com/opensourceways/defect-manager/defect/domain"
	"github.com/opensourceways/defect-manager/defect/domain/dp"
)

var instance *productTreeImpl

func Init(cfg *Config) {
	instance = &productTreeImpl{
		cli: client.NewClient(func() []byte {
			return []byte(cfg.Token)
		}),
		cfg:                 cfg,
		rpmCache:            make(map[string][]byte),
		rpmOfComponentCache: make(map[string]string),
		taskCount:           0,
	}
}

func Instance() *productTreeImpl {
	return instance
}

type productTreeImpl struct {
	cli client.Client
	cfg *Config

	rpmCache            map[string][]byte
	rpmOfComponentCache map[string]string

	lock      sync.Mutex
	wg        sync.WaitGroup
	taskCount int64
}

func (impl *productTreeImpl) InitCache() {
	atomic.AddInt64(&impl.taskCount, 1)

	impl.initRPMCache()
}

// CleanCache use atomic to avoid cleaning when other tasks are being performed
func (impl *productTreeImpl) CleanCache() {
	atomic.AddInt64(&impl.taskCount, -1)
	if atomic.LoadInt64(&impl.taskCount) == 0 {
		impl.rpmCache = make(map[string][]byte)
		impl.rpmOfComponentCache = make(map[string]string)
	}
}

func (impl *productTreeImpl) GetTree(defectTime time.Time, component string, versions []dp.SystemVersion) (domain.ProductTree, error) {
	affectedRPM := make(map[string]string)
	for _, v := range versions {
		key := fmt.Sprintf("%s_%s", component, v.String())
		rpm, ok := impl.rpmOfComponentCache[key]
		if !ok {
			rpm = impl.ParseRPM(defectTime, component, v.String())
			if rpm == "" {
				return nil, fmt.Errorf("parse rpm error")
			}
			impl.rpmOfComponentCache[key] = rpm
		}

		affectedRPM[v.String()] = rpm
	}

	return impl.buildTree(affectedRPM), nil
}

func (impl *productTreeImpl) ParseRPM(defectTime time.Time, component, version string) string {
	// content of buf, example:
	// https://gitee.com/openeuler_latest_rpms/obs_pkg_rpms_20230517/raw/master/latest_rpm/openEuler-22.03-LTS.csv
	buf := bytes.NewBuffer(impl.rpmCache[version])
	layout := "20060102 15-04-05"
	targetLayout := "2006-01-02 15:04:05"

	var rpmOfComponent string
	for {
		line, err := buf.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			logrus.Errorf("error reading %s from %s error %s", component, version, err.Error())
			continue
		}

		split := strings.Split(line, ",")
		if len(split) != 3 {
			logrus.Errorf("the format of line error")
			continue
		}

		if split[1] == component {
			parsedTime, err := time.Parse(layout, split[0])
			if err != nil {
				logrus.Errorf("parse time error %s", err.Error())
				break
			}

			formattedTime := parsedTime.Format(targetLayout)

			targetParsedTime, err := time.Parse(targetLayout, formattedTime)
			if err != nil {
				logrus.Errorf("parse targetParsedTime error %s", err.Error())
				break
			}

			if targetParsedTime.After(defectTime) {
				rpmOfComponent = split[2]
				break
			}
		}
	}

	return rpmOfComponent
}

func (impl *productTreeImpl) initRPMCache() {
	// use lock to avoid duplicate execution
	impl.lock.Lock()
	defer impl.lock.Unlock()
	if len(impl.rpmCache) == len(dp.MaintainVersion) {
		return
	}

	for version := range dp.MaintainVersion {
		v := version.String()
		impl.wg.Add(1)
		go func() {
			impl.fetchRPMData(v)
			impl.wg.Done()
		}()
	}

	impl.wg.Wait()
}

func (impl *productTreeImpl) fetchRPMData(version string) {
	count := 0
	maxCount := 10
	interval := time.Second * 3

	for {
		if count > maxCount {
			logrus.Errorf("fetch rpm data of %s failed after %d times", version, maxCount)
			break
		}
		count++

		content, err := impl.cli.GetPathContent(
			impl.cfg.PkgRPM.Org,
			impl.cfg.PkgRPM.Repo,
			fmt.Sprintf("%s%s.csv", impl.cfg.PkgRPM.PathPrefix, version),
			impl.cfg.PkgRPM.Branch,
		)
		if err != nil {
			logrus.Errorf("get content of %s error %s", version, err.Error())
			time.Sleep(interval)
			continue
		}

		decodeContent, err := base64.StdEncoding.DecodeString(content.Content)
		if err != nil {
			logrus.Errorf("base64decode content of %s error %s", version, err.Error())
			time.Sleep(interval)
			continue
		}

		impl.rpmCache[version] = decodeContent

		break
	}
}

func (impl *productTreeImpl) buildTree(affectedRPM map[string]string) domain.ProductTree {
	tree := make(map[dp.Arch][]domain.Product)
	for version, rpms := range affectedRPM {

		rpmSlice := strings.Fields(rpms)
		for _, rpm := range rpmSlice {
			// example of rpm: zbar-0.22-4.oe2203.src.rpm
			t := strings.Split(rpm, ".")
			arch := t[len(t)-2]
			productId := strings.Join(t[:len(t)-3], ".")

			product := domain.Product{
				ID:       productId,
				CPE:      version,
				FullName: rpm,
			}

			dpArch := dp.NewArch(arch)
			tree[dpArch] = append(tree[dpArch], product)
		}
	}

	return tree
}
