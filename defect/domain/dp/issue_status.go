package dp

import "errors"

const (
	open        = "待办的"
	progressing = "修复中"
	closed      = "已完成"
	confirmed   = "已确认"
	rejected    = "已挂起"
	canceled    = "已取消"
	accepted    = "已验收"
)

var (
	validIssueStatus = map[string]bool{
		open:        true,
		progressing: true,
		closed:      true,
		confirmed:   true,
		rejected:    true,
		canceled:    true,
		accepted:    true,
	}

	IssueStatusClosed = issueStatus(closed)
)

type issueStatus string

type IssueStatus interface {
	String() string
}

func NewIssueStatus(s string) (IssueStatus, error) {
	if !validIssueStatus[s] {
		return nil, errors.New("invalid issue status")
	}

	return issueStatus(s), nil
}

func (s issueStatus) String() string {
	return string(s)
}
