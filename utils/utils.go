package utils

import (
	"errors"
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

func NewMultiErrors() *MultiError {
	return new(MultiError)
}

type MultiError struct {
	es []string
}

func (e *MultiError) Add(s string) {
	if e != nil {
		if len(e.es) > 0 && strings.Contains(e.es[0], "@") {
			trimmedString := strings.SplitN(s, " ", 2)[1]
			trimmedString = strings.TrimSpace(trimmedString)
			e.es = append(e.es, trimmedString)
		} else {
			e.es = append(e.es, s)
		}
	}
}

func (e *MultiError) AddError(err error) {
	if err != nil {
		e.Add(err.Error())
	}
}

func (e *MultiError) Err() error {
	if e == nil || len(e.es) == 0 {
		return nil
	}
	return errors.New(strings.Join(e.es, ". "))
}

func RemoveDuplicates(strSlice []string) []string {
	uniqueMap := make(map[string]bool)
	result := []string{}

	for _, str := range strSlice {
		if !uniqueMap[str] {
			uniqueMap[str] = true
			result = append(result, str)
		}
	}

	return result
}
