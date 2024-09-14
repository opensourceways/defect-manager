package issue

import (
	"fmt"
	"regexp"
	"strings"

	sdk "github.com/opensourceways/go-gitee/gitee"
	"github.com/sirupsen/logrus"
)

const feedback = `
**二、缺陷分析结构反馈**
影响性分析说明：

缺陷严重等级:(Critical/High/Moderate/Low)

缺陷根因说明:

受影响版本排查(受影响/不受影响):
%v
修复是否涉及abi变化(是/否):
%v
`

const commentFeedback = `
影响性分析说明:
%v
缺陷严重等级:(Critical/High/Moderate/Low)
%v
缺陷根因说明:
%v
受影响版本排查(受影响/不受影响):
%v
修复是否涉及abi变化(是/否):
%v
`

const suspendTip = `
%v
**issue变更为 [已取消/已挂起] 状态时，必须由操作者填写相关原因，现issue被重新打开**
**请按如下格式评论原因后，重新进行操作**
************************************************************************
/reason xxxxxx
`

const commentCopyValue = `
%v
**issue处理注意事项:** 
**1. 当前issue受影响的分支提交pr时, 须在pr描述中填写当前issue编号进行关联, 否则无法关闭当前issue;**
**2. 模板内容需要填写完整, 无论是受影响或者不受影响都需要填写完整内容,未引入的分支不需要填写, 否则无法关闭当前issue;**
**3. 以下为模板中需要填写完整的内容, 请复制到评论区回复, 注: 内容的标题名称(影响性分析说明, 缺陷严重等级, 受影响版本排查(受影响/不受影响), 修复是否涉及abi变化(是/否))不能省略,省略后defect-manager将无法正常解析填写内容.**
**评论区可能使用到的指令说明:**
| 指令  | 指令说明 | 使用权限 |
|:--:|:--:|---------|
|/check-issue|触发defect-manager校验|不限|
|/reason xxx|/reason +挂起或取消条件|不限|
************************************************************************
影响性分析说明: 

缺陷严重等级:(Critical/High/Moderate/Low)

缺陷根因说明:

受影响版本排查(受影响/不受影响): 
%v
abi变化(是/否):
%v
-----------------------------------------------------------------------
缺陷issue处理具体操作请参考: 
%v
pr关联issue具体操作请参考:
%v
`
const rejectTb = `
| issue状态  | 操作者 | 原因 |
|:--:|:--:|---------|
|%v|%v|%v|
`

const rejectComment = `
%v 当前issue状态为: %v，若要追加评论，请先修改issue状态，否则评论无法被识别.
`

const tb1 = `
%v 经过defect-manager解析，已分析的内容如下表所示:
| 状态  | 需分析 | 内容 |
|:--:|:--:|---------|
|已分析|1.影响性分析说明|%v|
|已分析|2.缺陷严重等级|%v|
|已分析|3.缺陷根因定位|%v|
|已分析|4.受影响版本排查|%v|
|已分析|5.abi变化|%v|

**请确认分析内容的准确性，确认无误后，您可以进行后续步骤，否则您可以继续分析**
`

const tb2 = `
%v 经过defect-manager解析，已分析的内容如下表所示:
| 状态  | 需分析 | 内容 |
|:--:|:--:|---------|
|已分析|1.影响性分析说明|%v|
|已分析|2.缺陷严重等级|%v|
|已分析|3.缺陷根因定位|%v|
|已分析|4.受影响版本排查|%v|
|已分析|5.abi变化|%v|

**请确认分析内容的准确性，确认无误后，您可以进行后续步骤，否则您可以继续分析**
`

const reOpenComment = `
@%s
关闭issue前,需要将受影响的分支在合并pr时关联上当前issue编号: #%v
受影响分支: %v
具体操作参考: %v
`

const commentVersionTip = `
%v 请确认分支: %v.
**请确认%v是否填写完整，否则将无法关闭当前issue.**
`

const (
	commentCmd  = "https://gitee.com/Coopermassaki/cve-manager/blob/master/cve-vulner-manager/doc/md/defect-manager-manual.md"
	PrIssueLink = "https://gitee.com/help/articles/4142"
)

// When issue created, add issue body part 2 defect analysis structurecontent
func addAnalysisFeedback(body, name string, maintainVersion []string) sdk.IssueUpdateParam {
	newBody := generateAnalysisFeedbackBody(body, maintainVersion)

	return sdk.IssueUpdateParam{
		Body: newBody,
		Repo: name,
	}
}

func analysisCommentFeedback(body, name string, comment parseCommentResult) sdk.IssueUpdateParam {
	newBody := generateanalysisCommentFeedbackBody(body, comment)

	return sdk.IssueUpdateParam{
		Body: newBody,
		Repo: name,
	}
}

// If issue parse success create first comment
func commentTemplate(maintainVersion, committerList []string) string {
	if committerList == nil {
		return ""
	}

	var affectedVersion string
	for i, version := range maintainVersion {
		affectedVersion += fmt.Sprintf("%d. %s:\n", i+1, version)
	}

	assList := []string{}
	assigneeStr := ""

	for _, v := range committerList {
		assList = append(assList, "@"+v)
	}

	assigneeStr = strings.Join(assList, " , ")

	return fmt.Sprintf(commentCopyValue, assigneeStr, affectedVersion, affectedVersion, commentCmd, PrIssueLink)
}

func analysisComplete(assigner *sdk.UserHook, anlysisComment parseCommentResult) string {
	if assigner == nil {
		return ""
	}

	if anlysisComment.RootCause == "" {
		anlysisComment.RootCause = "无"
	}

	assigning := "@" + assigner.UserName

	return fmt.Sprintf(
		tb1,
		assigning,
		strings.ReplaceAll(anlysisComment.Influence, "\r\n", ""),
		anlysisComment.SeverityLevel,
		anlysisComment.RootCause,
		strings.Join(anlysisComment.AllVersionResult, ","),
		strings.Join(anlysisComment.AllAbiResult, ","),
	)
}

func modifyIssueBodyStyle(labels []sdk.LabelHook, name string) sdk.IssueUpdateParam {
	newLabels := dealLabels(labels, unFixedLabel)

	return sdk.IssueUpdateParam{
		Labels: newLabels,
		Repo:   name,
	}
}

func generateAnalysisFeedbackBody(body string, maintainVersion []string) string {
	var affectedVersion string
	for _, version := range maintainVersion {
		affectedVersion += fmt.Sprintf("%s\n", version)
	}

	return body + fmt.Sprintf(feedback, affectedVersion, affectedVersion)
}

func generateanalysisCommentFeedbackBody(body string, comment parseCommentResult) string {
	analysisBody := fmt.Sprintf(commentFeedback, comment.Influence,
		comment.SeverityLevel, comment.RootCause,
		strings.Join(comment.AllVersionResult, "\n"), strings.Join(comment.AllAbiResult, "\n"))

	regItemFirstPartDefectInfo := regexp.MustCompile(`(\*\*【缺陷描述】：请补充详细的缺陷问题现象描述)([\s\S]*?)\*\*二、缺陷分析结构反馈\*\*`)
	match := regItemFirstPartDefectInfo.FindAllStringSubmatch(body, -1)
	if len(match) == 0 || len(match[0]) == 0 {
		logrus.Error("issue body not match, not find regItemFirstPartDefectInfo")
		return body
	}

	matchBody := match[regMatchResult][regMatchResult]

	return matchBody + analysisBody
}

func dealLabels(labels []sdk.LabelHook, updateLabel string) string {
	if len(labels) == 0 {
		return updateLabel
	}

	var labelNames []string

	for _, v := range labels {
		if v.Name != unAffectedLabel && v.Name != fixedLabel && v.Name != unFixedLabel {
			labelNames = append(labelNames, v.Name)
		}
	}

	labelNames = append(labelNames, updateLabel)

	return strings.Join(labelNames, ",")
}
