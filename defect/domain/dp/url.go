package dp

import (
	"net/url"

	"github.com/sirupsen/logrus"
)

type dpUrl string

type URL interface {
	URL() string
}

func NewURL(s string) (URL, error) {
	if s == "" {
		logrus.Warn("empty url")
		return dpUrl(""), nil
	}

	if _, err := url.ParseRequestURI(s); err != nil {
		logrus.Warn("invalid url")
		return dpUrl(""), nil
	}

	return dpUrl(s), nil
}

func (u dpUrl) URL() string {
	return string(u)
}
