package issue

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	sdk "github.com/opensourceways/go-gitee/gitee"
	"github.com/opensourceways/robot-gitee-lib/client"
	"github.com/opensourceways/server-common-lib/utils"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/opensourceways/defect-manager/defect/app"
	"github.com/opensourceways/defect-manager/defect/domain"
	"github.com/opensourceways/defect-manager/defect/domain/dp"
	defectUtils "github.com/opensourceways/defect-manager/utils"
)

var Instance *eventHandler

type EventHandler interface {
	HandleIssueEvent(e *sdk.IssueEvent) error
	HandleNoteEvent(e *sdk.NoteEvent) error
}

type iClient interface {
	UpdateIssue(owner, number string, param sdk.IssueUpdateParam) (sdk.Issue, error)
	CreateIssueComment(org, repo string, number string, comment string) error
	ListIssueComments(org, repo, number string) ([]sdk.Note, error)
	CloseIssue(owner, repo string, number string) error
	ReopenIssue(owner, repo string, number string) error
	GetBot() (sdk.User, error)
	GetIssue(org, repo, number string) (sdk.Issue, error)
}

func InitEventHandler(c *Config, s app.DefectService) error {
	cli := client.NewClient(func() []byte {
		return []byte(c.RobotToken)
	})

	bot, err := cli.GetBot()
	if err != nil {
		return err
	}

	Instance = &eventHandler{
		botName: bot.Login,
		cfg:     c,
		cli:     cli,
		service: s,
	}

	return nil
}

type eventHandler struct {
	botName string
	cfg     *Config
	cli     iClient
	service app.DefectService
}

func (impl eventHandler) HandleIssueEvent(e *sdk.IssueEvent) error {
	if e.Issue.TypeName != impl.cfg.IssueType {
		return nil
	}

	for _, v := range impl.cfg.DevelopVersion {
		if strings.Contains(e.Issue.Body, v) {
			return nil
		}
	}

	switch e.Issue.StateName {
	case StatusFinished:
		return impl.handleIssueClosed(e)
	case StatusAccept:
		return impl.handleIssueClosed(e)
	case StatusTodo:
		return impl.handleIssueOpen(e)

	case StatusCancel:
		return impl.handleIssueReject(e)

	case StatusSuspend:
		return impl.handleIssueReject(e)
	default:
		return nil
	}
}

func (impl eventHandler) handleIssueReject(e *sdk.IssueEvent) error {
	commentIssue := func(content string) error {
		return impl.cli.CreateIssueComment(e.Project.Namespace,
			e.Project.Name, e.Issue.Number, content,
		)
	}

	comments, err := impl.cli.ListIssueComments(e.Project.Namespace, e.Project.Name, e.Issue.Number)
	if err != nil {
		logrus.Errorf("get comments error: %s", err.Error())

		return nil
	}

	for i := len(comments) - 1; i >= 0; i-- {
		if strings.Contains(comments[i].Body, "/reason") && comments[i].User.Login != impl.botName &&
			CommitterInstance.isCommitter(e.Repository.PathWithNamespace, comments[i].User.Login) {
			newLabels := dealLabels(e.Issue.Labels, "")
			if _, err := impl.cli.UpdateIssue(e.Project.Namespace, e.Issue.Number,
				sdk.IssueUpdateParam{Labels: newLabels, Repo: e.Project.Name}); err != nil {
				return fmt.Errorf("update issue error: %s", err.Error())
			}

			str1 := fmt.Sprintf(rejectTb, e.Issue.StateName, e.Sender.UserName, strings.ReplaceAll(comments[i].Body, "/reason", ""))
			if err := commentIssue(str1); err != nil {
				return err
			}

			str2 := fmt.Sprintf(rejectComment, "@"+e.Sender.UserName, e.Issue.StateName)
			return commentIssue(str2)
		}
	}

	if err = impl.cli.ReopenIssue(e.Project.Namespace, e.Project.Name, e.Issue.Number); err != nil {
		return fmt.Errorf("reopen issue error: %s", err.Error())
	}

	logrus.Infof("reopen issue %s %s", e.Project.PathWithNamespace, e.Issue.Number)

	return commentIssue(fmt.Sprintf(suspendTip, "@"+e.Sender.UserName))
}

func (impl eventHandler) handleIssueClosed(e *sdk.IssueEvent) error {
	commentIssue := func(content string) error {
		return impl.cli.CreateIssueComment(e.Project.Namespace,
			e.Project.Name, e.Issue.Number, content,
		)
	}

	issueInfo, err := impl.parseIssue(e.Sender, e.Issue.Body)
	if err != nil {
		//return commentIssue(strings.Replace(err.Error(), ". ", "\n\n", -1))
		logrus.Errorf("parse issue error: %s", err.Error())
	}

	comment := impl.getAnalysisComment(e)
	if comment == "" {
		if err = impl.cli.ReopenIssue(e.Project.Namespace, e.Project.Name, e.Issue.Number); err != nil {
			return fmt.Errorf("reopen issue error: %s", err.Error())
		}

		logrus.Infof("reopen issue %s %s", e.Project.PathWithNamespace, e.Issue.Number)

		return commentIssue(fmt.Sprintf("@%s 未对受影响版本排查/abi变化进行分析，重新打开issue", e.Sender.UserName))
	}

	commentInfo, err := impl.parseComment(e.Sender, comment)
	if err != nil {
		return commentIssue(strings.Replace(err.Error(), ". ", "\n\n", -1))
	}

	if len(commentInfo.AffectedVersion) == 0 {
		newLabels := dealLabels(e.Issue.Labels, unAffectedLabel)
		if _, err := impl.cli.UpdateIssue(e.Project.Namespace, e.Issue.Number,
			sdk.IssueUpdateParam{Labels: newLabels, Repo: e.Project.Name}); err != nil {
			return fmt.Errorf("update issue error: %s", err.Error())
		}

		return nil
	}

	exist, err := impl.service.IsDefectExist(&domain.Issue{
		Number: e.GetIssueNumber(),
		Org:    e.Project.Namespace,
	})
	if err != nil {
		return err
	}

	if exist {
		newLabels := dealLabels(e.Issue.Labels, fixedLabel)
		if _, err := impl.cli.UpdateIssue(e.Project.Namespace, e.Issue.Number,
			sdk.IssueUpdateParam{Labels: newLabels, Repo: e.Project.Name}); err != nil {
			return fmt.Errorf("update issue error: %s", err.Error())
		}

		return nil
	}

	if relatedPRNotMerged := impl.checkRelatedPR(e, commentInfo.AffectedVersion); relatedPRNotMerged != nil {
		if err = impl.cli.ReopenIssue(e.Project.Namespace, e.Project.Name, e.Issue.Number); err != nil {
			return fmt.Errorf("reopen issue error: %s", err.Error())
		}

		logrus.Infof("reopen issue %s %s", e.Project.PathWithNamespace, e.Issue.Number)

		str := fmt.Sprintf(reOpenComment, e.Sender.UserName, e.Issue.Number, strings.Join(relatedPRNotMerged, "/"), PrIssueLink)
		return commentIssue(str)
	}

	cmd, err := impl.toCmd(e.Issue.Title, e.Issue.Number, e.Repository.Namespace, e.Repository.Name, issueInfo, commentInfo)
	if err != nil {
		return fmt.Errorf("to cmd error: %s", err.Error())
	}

	err = impl.service.SaveDefects(cmd)
	if err == nil {
		newLabels := dealLabels(e.Issue.Labels, fixedLabel)
		if _, err := impl.cli.UpdateIssue(e.Project.Namespace, e.Issue.Number,
			sdk.IssueUpdateParam{Labels: newLabels, Repo: e.Project.Name}); err != nil {
			return fmt.Errorf("update issue error: %s", err.Error())
		}

		return nil
	}

	logrus.Errorf("when save defect some err occurred: %s", err.Error())

	return nil
}

func (impl eventHandler) handleIssueOpen(e *sdk.IssueEvent) error {
	if *e.Action == "assign" {
		return nil
	}

	cp := checkIssueParam{
		namespace:     e.Project.Namespace,
		name:          e.Project.Name,
		issueBody:     e.Issue.Body,
		issueNumber:   e.Issue.Number,
		issueId:       e.Issue.Id,
		issueCreateAt: e.Issue.CreatedAt,
		issueUser:     e.User,
		issueAssigner: e.Assignee,
		labels:        e.Issue.Labels,
	}

	return impl.checkIssue(cp)
}

// || e.Comment.User.Login == impl.botName
func (impl eventHandler) HandleNoteEvent(e *sdk.NoteEvent) error {
	if !e.IsIssue() || e.Issue.TypeName != impl.cfg.IssueType ||
		e.Issue.StateName == StatusFinished || e.Issue.StateName == StatusCancel ||
		e.Issue.StateName == StatusSuspend || e.Comment.User.Login == impl.botName {
		return nil
	}
	commentIssue := func(content string) error {
		return impl.cli.CreateIssueComment(e.Project.Namespace,
			e.Project.Name, e.Issue.Number, content,
		)
	}

	if e.Comment.Body == cmdCheck {
		cp := checkIssueParam{
			namespace:     e.Project.Namespace,
			name:          e.Project.Name,
			issueBody:     e.Issue.Body,
			issueNumber:   e.Issue.Number,
			issueId:       e.Issue.Id,
			issueCreateAt: e.Issue.CreatedAt,
			issueUser:     e.Issue.User,
			issueAssigner: e.Issue.Assignee,
			labels:        e.Issue.Labels,
		}

		return impl.checkIssue(cp)
	}

	if strings.Contains(e.Comment.Body, influence) {
		issueInfo, err := impl.parseIssue(e.Comment.User, e.Issue.Body)
		if err != nil {
			logrus.Errorf("parse issue error: %s", err.Error())
			//return commentIssue(strings.Replace(err.Error(), ". ", "\n\n", -1))
		}

		commentInfo, err := impl.parseComment(e.Comment.User, e.Comment.Body)
		if err != nil {
			return commentIssue(err.Error())
		}

		issueUpdateParam := analysisCommentFeedback(e.Issue.Body, e.Project.Name, commentInfo)
		if _, err := impl.cli.UpdateIssue(e.Project.Namespace, e.Issue.Number, issueUpdateParam); err != nil {
			return fmt.Errorf("update issue error: %s", err.Error())
		}

		tbStr := analysisComplete(e.Issue.Assignee, commentInfo)
		if err := commentIssue(tbStr); err != nil {
			return fmt.Errorf("create issue form comment error: %s", err.Error())
		}

		if len(commentInfo.AffectedVersion) == 0 {

			cmd, err := impl.toCmd(e.Issue.Title, e.Issue.Number, e.Repository.Namespace, e.Repository.Name, issueInfo, commentInfo)
			if err != nil {
				return fmt.Errorf("to cmd error: %s", err.Error())
			}

			return impl.service.SaveDefects(cmd)
		}

		return nil

	}

	return nil
}

// the content of the comment of the newest /approve reply to
func (impl eventHandler) getAnalysisComment(e *sdk.IssueEvent) string {
	comments, err := impl.cli.ListIssueComments(e.Project.Namespace, e.Project.Name, e.Issue.Number)
	if err != nil {
		logrus.Errorf("get comments error: %s", err.Error())

		return ""
	}

	for i := len(comments) - 1; i >= 0; i-- {
		if strings.Contains(comments[i].Body, influence) &&
			comments[i].User.Login != impl.botName &&
			CommitterInstance.isCommitter(e.Repository.PathWithNamespace, comments[i].User.Login) {
			return comments[i].Body
		}
	}

	return ""
}

func (impl eventHandler) toCmd(title, number, namespace, name string, issue parseIssueResult, comment parseCommentResult) (
	cmd app.CmdToSaveDefect, err error) {
	systemVersion, err := dp.NewSystemVersion(issue.OS)
	if err != nil {
		return
	}

	securityLevel, err := dp.NewSeverityLevel(comment.SeverityLevel)
	if err != nil {
		return
	}

	var affectedVersion []dp.SystemVersion
	for _, v := range comment.AffectedVersion {
		var dv dp.SystemVersion
		if dv, err = dp.NewSystemVersion(v); err != nil {
			return
		}
		affectedVersion = append(affectedVersion, dv)
	}

	return app.CmdToSaveDefect{
		Kernel:           issue.Kernel,
		Component:        name,
		ComponentVersion: issue.ComponentVersion,
		SystemVersion:    systemVersion,
		Description:      issue.Description,
		ReferenceURL:     nil,
		GuidanceURL:      nil,
		Influence:        comment.Influence,
		SeverityLevel:    securityLevel,
		RootCause:        comment.RootCause,
		AffectedVersion:  affectedVersion,
		ABI:              strings.Join(comment.Abi, ","),
		Issue: domain.Issue{
			Title:  title,
			Number: number,
			Org:    namespace,
			Repo:   name,
			Status: dp.IssueStatusClosed,
		},
	}, nil
}

func (impl eventHandler) checkRelatedPR(e *sdk.IssueEvent, versions []string) []string {
	endpoint := fmt.Sprintf("https://gitee.com/api/v5/repos/%v/issues/%v/pull_requests?access_token=%s&repo=%s",
		e.Project.Namespace, e.Issue.Number, impl.cfg.RobotToken, e.Project.Name,
	)
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		logrus.Errorf("create request error: %s", err.Error())
		return nil
	}

	var prs []sdk.PullRequest
	cli := utils.NewHttpClient(3)
	bytes, _, err := cli.Download(req)
	if err != nil {
		logrus.Errorf("download error: %s", err.Error())
		return nil
	}

	if err := json.Unmarshal(bytes, &prs); err != nil {
		logrus.Errorf("unmarshal error: %s", err.Error())
		return nil
	}

	mergedVersion := sets.NewString()
	for _, pr := range prs {
		if pr.State == sdk.StatusMerged {
			mergedVersion.Insert(pr.Base.Ref)
		}
	}
	var relatedPRNotMerged []string
	for _, v := range versions {
		if !mergedVersion.Has(v) {
			relatedPRNotMerged = append(relatedPRNotMerged, v)
		}
	}

	if len(relatedPRNotMerged) != 0 {
		return relatedPRNotMerged
	}

	return nil
}

type checkIssueParam struct {
	namespace     string
	name          string
	issueBody     string
	issueNumber   string
	issueId       int32
	issueCreateAt time.Time
	issueUser     *sdk.UserHook
	issueAssigner *sdk.UserHook
	labels        []sdk.LabelHook
}

func (impl eventHandler) checkIssue(cp checkIssueParam) error {
	if cp.issueAssigner == nil {
		err := impl.setIssueAssignee(cp.namespace, cp.name, cp.issueNumber)
		if err != nil {
			logrus.Errorf("set issue assignee error: %s", err.Error())
		}
	}

	dp := dealIssueParam{
		namespace:   cp.namespace,
		name:        cp.name,
		issueBody:   cp.issueBody,
		issueNumber: cp.issueNumber,
	}

	_, err := impl.dealIssue(dp)
	if err != nil {
		return fmt.Errorf("deal issue error: %s", err.Error())
	}

	issueUpdateParam := modifyIssueBodyStyle(cp.labels, cp.name)

	if _, err := impl.cli.UpdateIssue(cp.namespace, cp.issueNumber, issueUpdateParam); err != nil {
		return fmt.Errorf("update issue error: %s", err.Error())
	}

	/*  	if _, err := impl.parseIssue(cp.issueUser, newbody); err != nil {
		return impl.cli.CreateIssueComment(cp.namespace,
			cp.name, cp.issueNumber, strings.Replace(err.Error(), ". ", "\n\n", -1),
		)
	}

	if _, err := impl.cli.UpdateIssue(cp.namespace, cp.issueNumber, issueUpdateParam); err != nil {
		return fmt.Errorf("update issue error: %s", err.Error())
	} */

	/* 	if err := impl.cli.CreateIssueComment(cp.namespace, cp.name, cp.issueNumber, fmt.Sprintf(issueCheckSuccess, cp.issueUser.UserName)); err != nil {
		return fmt.Errorf("create issue comment error: %s", err.Error())
	} */

	dl := deadLineParam{
		name:         cp.name,
		enterpriseId: impl.cfg.EnterpriseId,
		issueId:      cp.issueId,
		issueCreatAt: cp.issueCreateAt,
	}

	return impl.updateIssueDeadline(dl)
}

type dealIssueParam struct {
	namespace   string
	name        string
	issueBody   string
	issueNumber string
}

func (impl eventHandler) dealIssue(dp dealIssueParam) (string, error) {
	newbody := dp.issueBody
	if !strings.Contains(dp.issueBody, "二、缺陷分析结构反馈") {
		issueUpdateParam := addAnalysisFeedback(dp.issueBody, dp.name, impl.cfg.MaintainVersion)

		if _, err := impl.cli.UpdateIssue(dp.namespace, dp.issueNumber, issueUpdateParam); err != nil {
			return newbody, fmt.Errorf("update issue error: %s", err.Error())
		}

		newbody = issueUpdateParam.Body
	}

	comments, err := impl.cli.ListIssueComments(dp.namespace, dp.name, dp.issueNumber)
	if err != nil {
		return "", err
	}

	for _, v := range comments {
		if strings.Contains(v.Body, "issue处理注意事项") {
			return newbody, nil
		}
	}
	logrus.Infof("repo: %s", strings.Join([]string{dp.namespace, dp.name}, "/"))
	committerList := CommitterInstance.listCommitter(strings.Join([]string{dp.namespace, dp.name}, "/"))
	if len(committerList) == 0 {
		return "", fmt.Errorf("获取committer列表失败，请联系管理员")
	}

	firstComment := commentTemplate(impl.cfg.MaintainVersion, committerList)

	if firstComment == "" {
		return newbody, fmt.Errorf("issue comment template is empty")
	}

	return newbody, impl.cli.CreateIssueComment(dp.namespace, dp.name, dp.issueNumber, firstComment)
}

type deadLineParam struct {
	name         string
	enterpriseId string
	issueId      int32
	issueCreatAt time.Time
}

func (impl eventHandler) updateIssueDeadline(dl deadLineParam) error {
	endpoint := fmt.Sprintf("https://api.gitee.com/enterprises/%v/issues/%s", dl.enterpriseId, strconv.FormatInt(int64(dl.issueId), base))

	issueReq := impl.setDeadline(dl.name, dl.issueCreatAt)

	issueReqJSON, err := json.Marshal(issueReq)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPut, endpoint, bytes.NewBuffer(issueReqJSON))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	cli := utils.NewHttpClient(3)
	_, _, err = cli.Download(req)

	return err
}

type IssueParams struct {
	AccessToken   string `json:"access_token"`
	PlanStartedAt string `json:"plan_started_at"`
	Deadline      string `json:"deadline"`
}

func (impl eventHandler) setDeadline(name string, createAt time.Time) IssueParams {
	startAt := defectUtils.FormatTime(createAt)
	dl := defectUtils.FormatTime(createAt.AddDate(0, oneMonth, 0))

	for _, policy := range impl.cfg.PkgPolicy {
		for k, v := range policy {
			if name == k {
				dl = defectUtils.FormatTime(createAt.AddDate(0, 0, v))
				break
			}
		}
	}

	return IssueParams{
		AccessToken:   impl.cfg.EnterpriseToken,
		PlanStartedAt: startAt,
		Deadline:      dl,
	}
}

func (impl eventHandler) setIssueAssignee(namespace, repo, number string) error {
	pathWithNamespace := strings.Join([]string{namespace, repo}, "/")
	logrus.Infof("pathWithNamespace: %s", pathWithNamespace)
	assigner := CommitterInstance.getAssigner(pathWithNamespace)
	if assigner == "" {
		return fmt.Errorf("%s get assigner error", namespace)
	}

	if _, err := impl.cli.UpdateIssue(namespace, number, sdk.IssueUpdateParam{Assignee: assigner, Repo: repo}); err != nil {
		return err
	}

	return nil
}
