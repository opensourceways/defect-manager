package utils

import (
	"strings"
	"time"
)

func TrimString(str string) string {
	str = strings.Replace(str, " ", "", -1)
	str = strings.Replace(str, "\n", "", -1)
	str = strings.Replace(str, "\r", "", -1)
	return strings.Replace(str, "\t", "", -1)
}

func Now() int64 {
	return time.Now().Unix()
}

func ToDate(n int64) string {
	if n == 0 {
		n = Now()
	}

	return time.Unix(n, 0).Format("2006-01-02")
}

func Date() string {
	return ToDate(Now())
}

func Year() int {
	return time.Now().Year()
}

func FormatTime(t time.Time) string {
	return t.Format("2006-01-02T15:04:05")
}
