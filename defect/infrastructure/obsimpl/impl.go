package obsimpl

import (
	"bytes"
	"fmt"

	"github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"

	"github.com/opensourceways/defect-manager/utils"
)

const uploadedDefect = "update_defect.txt"

var instance *obsImpl

func Init(cfg *Config) error {
	cli, err := obs.New(cfg.AccessKey, cfg.SecretKey, cfg.Endpoint)
	if err != nil {
		return err
	}

	instance = &obsImpl{
		cfg: cfg,
		cli: cli,
	}

	return nil
}

func Instance() *obsImpl {
	return instance
}

type obsImpl struct {
	cfg *Config
	cli *obs.ObsClient
}

func (impl obsImpl) Upload(fileName string, data []byte) error {
	input := &obs.PutObjectInput{}
	input.Bucket = impl.cfg.Bucket
	if fileName == uploadedDefect {
		input.Key = fmt.Sprintf("%s/%s", impl.cfg.Directory, fileName)
	} else {
		input.Key = fmt.Sprintf("%s/%d/%s", impl.cfg.Directory, utils.Year(), fileName)
	}
	input.Body = bytes.NewReader(data)

	_, err := impl.cli.PutObject(input)

	return err
}
