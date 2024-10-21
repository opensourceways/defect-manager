package dp

import "github.com/sirupsen/logrus"

const (
	critical = "Critical"
	high     = "High"
	moderate = "Moderate"
	low      = "Low"
)

var validateSeverityLevel = map[string]bool{
	critical: true,
	high:     true,
	moderate: true,
	low:      true,
}

var SequenceSeverityLevel = []string{low, moderate, high, critical}

type severityLevel string

func NewSeverityLevel(s string) (SeverityLevel, error) {
	if !validateSeverityLevel[s] {
		logrus.Warningf("invalid severity level: %s", s)
		return severityLevel(s), nil
	}

	return severityLevel(s), nil
}

type SeverityLevel interface {
	String() string
}

func (s severityLevel) String() string {
	return string(s)
}
