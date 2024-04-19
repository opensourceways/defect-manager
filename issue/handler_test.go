package issue

import (
	"errors"
	"testing"
	"time"

	sdk "github.com/opensourceways/go-gitee/gitee"

	"github.com/opensourceways/defect-manager/defect/app"
	"github.com/opensourceways/defect-manager/defect/domain"
)

func TestIssueClosed(t *testing.T) {
	h := &eventHandler{
		cli:     new(cliTest),
		service: new(serviceTest),
	}

	issue := sdk.IssueEvent{
		Issue: &sdk.IssueHook{
			Number: "fksj",
		},
		Project: &sdk.ProjectHook{
			Namespace:         "fdsf",
			Name:              "xxx",
			PathWithNamespace: "xxx",
		},
	}

	err := h.handleIssueClosed(&issue)
	if err == nil || err.Error() != "缺陷数据未收集完成，重新打开issue" {
		t.Failed()
	}
}

type cliTest struct {
}

func (t cliTest) UpdateIssue(owner, number string, param sdk.IssueUpdateParam) (sdk.Issue, error) {
	return sdk.Issue{}, nil
}

func (t cliTest) CreateIssueComment(org, repo string, number string, comment string) error {
	return errors.New("缺陷数据未收集完成，重新打开issue")
}

func (t cliTest) ListIssueComments(org, repo, number string) ([]sdk.Note, error) {
	return nil, nil
}

func (t cliTest) CloseIssue(owner, repo string, number string) error {
	return nil
}

func (t cliTest) ReopenIssue(owner, repo string, number string) error {
	return nil
}

func (t cliTest) GetBot() (sdk.User, error) {
	return sdk.User{}, nil
}

func (t cliTest) GetIssue(org, repo, number string) (sdk.Issue, error) {
	return sdk.Issue{}, nil
}

type serviceTest struct {
}

func (t serviceTest) IsDefectExist(*domain.Issue) (bool, error) {
	return false, nil
}

func (t serviceTest) SaveDefects(app.CmdToSaveDefect) error {
	return nil
}

func (t serviceTest) CollectDefects(time time.Time) ([]app.CollectDefectsDTO, error) {
	return nil, nil
}

func (t serviceTest) GenerateBulletins([]string) error {
	return nil
}
