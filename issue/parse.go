package issue

import (
	"fmt"
	"regexp"
	"strings"

	sdk "github.com/opensourceways/go-gitee/gitee"
	"github.com/opensourceways/server-common-lib/utils"
	"k8s.io/apimachinery/pkg/util/sets"

	localutils "github.com/opensourceways/defect-manager/utils"
)

const (
	cmdCheck        = "/check-issue"
	unAffectedLabel = "DEFECT/UNAFFECTED"
	fixedLabel      = "DEFECT/FIXED"
	unFixedLabel    = "DEFECT/UNFIXED"

	itemDescription                   = "description"
	itemOS                            = "os"
	itemKernel                        = "kernel"
	itemComponents                    = "components"
	itemProblemReproductionSteps      = "problemReproductionSteps"
	itemTitleDescription              = "descriptionTitle"
	itemTitleOS                       = "osTitle"
	itemTitleKernel                   = "kernelTitle"
	itemTitleComponents               = "componentsTitle"
	itemTitleProblemReproductionSteps = "problemReproductionStepsTitle"
	itemReferenceAndGuidanceUrl       = "referenceAndGuidanceUrl"
	itemInfluence                     = "influence"
	itemSeverityLevel                 = "severityLevel"
	itemAffectedVersion               = "affectedVersion"
	itemAbi                           = "abi"

	severityLevelLow      = "Low"
	severityLevelModerate = "Moderate"
	severityLevelHigh     = "High"
	severityLevelCritical = "Critical"

	regMatchResult = 0
	regMatchItem   = 2
	oneMonth       = 1
	TickerInterval = 24
	base           = 10

	influence = "影响性分析说明"
	notice    = "issue处理注意事项"
	parse     = "经过defect-manager解析"
	failParse = "解析失败"
	notNull   = "不允许为空"

	StatusTodo      = "待办的"
	StatusRepairing = "修复中"
	StatusConfirm   = "已确认"
	StatusFinished  = "已完成"
	StatusAccept    = "已验收"
	StatusSuspend   = "已挂起"
	StatusCancel    = "已取消"
)

var (
	itemName = map[string]string{
		itemDescription:                   "缺陷描述",
		itemOS:                            "缺陷所属的os版本",
		itemKernel:                        "内核版本",
		itemComponents:                    "缺陷所属软件及版本号",
		itemProblemReproductionSteps:      "问题复现步骤",
		itemTitleDescription:              "缺陷描述",
		itemTitleOS:                       "缺陷所属的os版本",
		itemTitleKernel:                   "内核版本",
		itemTitleComponents:               "缺陷所属软件及版本号",
		itemTitleProblemReproductionSteps: "问题复现步骤",
		itemReferenceAndGuidanceUrl:       "详情及分析指导参考链接",
		itemInfluence:                     "影响性分析说明",
		itemSeverityLevel:                 "缺陷严重等级",
		itemAffectedVersion:               "受影响版本",
		itemAbi:                           "abi",
	}

	regexpOfItems = map[string]*regexp.Regexp{
		itemDescription:                   regexp.MustCompile(`(缺陷描述)[】][(（]必填[)）][:：]请补充详细的缺陷问题现象描述\*\*([\s\S]*?)\*\*一、缺陷信息`),
		itemOS:                            regexp.MustCompile(`(缺陷所属的os版本)[】][(（]必填，如openEuler-22.03-LTS[)）]\*\*([\s\S]*?)\*\*【内核版本`),
		itemKernel:                        regexp.MustCompile(`(内核版本)[】][(（]必填，如kernel-4.19[)）]\*\*([\s\S]*?)\*\*【缺陷所属软件及版本号`),
		itemComponents:                    regexp.MustCompile(`(缺陷所属软件及版本号)[】][(（]必填，如kernel-4.19[)）]\*\*([\s\S]*?)\*\*【环境信息`),
		itemProblemReproductionSteps:      regexp.MustCompile(`(问题复现步骤)[】][(（]必填[)）][:：]请描述具体的操作步骤\*\*([\s\S]*?)\*\*【实际结果`),
		itemTitleDescription:              regexp.MustCompile(`(缺陷描述)[】][(（]必填[)）][:：]请补充详细的缺陷问题现象描述([\s\S]*?)\*\*一、缺陷信息`),
		itemTitleOS:                       regexp.MustCompile(`(缺陷所属的os版本)[】][(（]必填，如openEuler-22.03-LTS[)）]([\s\S]*?)### 【内核版本`),
		itemTitleKernel:                   regexp.MustCompile(`(内核版本)[】][(（]必填，如kernel-4.19[)）]([\s\S]*?)### 【缺陷所属软件及版本号`),
		itemTitleComponents:               regexp.MustCompile(`(缺陷所属软件及版本号)[】][(（]必填，如kernel-4.19[)）]([\s\S]*?)\*\*【环境信息`),
		itemTitleProblemReproductionSteps: regexp.MustCompile(`(问题复现步骤)[】][(（]必填[)）][:：]请描述具体的操作步骤([\s\S]*?)\*\*【实际结果`),
		itemInfluence:                     regexp.MustCompile(`(影响性分析说明)[:：]([\s\S]*?)缺陷严重等级`),
		itemSeverityLevel:                 regexp.MustCompile(`(缺陷严重等级)[:：]\(Critical/High/Moderate/Low\)([\s\S]*?)受影响版本排查`),
		itemAffectedVersion:               regexp.MustCompile(`(受影响版本排查)\(受影响/不受影响\)[:：]([\s\S]*?)abi变化`),
		itemAbi:                           regexp.MustCompile(`(abi变化)\(是/否\)[:：]([\s\S]*?)$`),
	}

	sortOfIssueItems = []string{
		itemDescription,
		itemOS,
		itemKernel,
		itemComponents,
		itemProblemReproductionSteps,
	}

	sortOfIssueTitleItems = []string{
		itemTitleDescription,
		itemTitleOS,
		itemTitleKernel,
		itemTitleComponents,
		itemTitleProblemReproductionSteps,
	}

	sortOfCommentItems = []string{
		itemInfluence,
		itemSeverityLevel,
		itemAffectedVersion,
		itemAbi,
	}

	noTrimItem = map[string]bool{
		itemDescription: true,
		itemInfluence:   true,
	}

	severityLevelMap = map[string]bool{
		severityLevelLow:      true,
		severityLevelModerate: true,
		severityLevelHigh:     true,
		severityLevelCritical: true,
	}
)

type parseIssueResult struct {
	Kernel           string
	Component        string
	ComponentVersion string
	OS               string
	Description      string
}

type parseCommentResult struct {
	Influence        string
	SeverityLevel    string
	AllVersionResult []string
	AllAbiResult     []string
	AffectedVersion  []string
	Abi              []string
}

func (impl eventHandler) parseIssue(assigner *sdk.UserHook, body string) (parseIssueResult, error) {
	var parseIssueParam = sortOfIssueItems
	if strings.Contains(body, "###") {
		parseIssueParam = sortOfIssueTitleItems
	}

	result, err := impl.parse(parseIssueParam, assigner, body)
	if err != nil {
		return parseIssueResult{}, err
	}

	var ret parseIssueResult
	if v, ok := result[itemKernel]; ok {
		ret.Kernel = v
	}

	if v, ok := result[itemComponents]; ok {
		split := strings.Split(v, "-")

		ret.Component = strings.Join(split[:len(split)-1], "-")
		ret.ComponentVersion = split[len(split)-1]
	}

	if v, ok := result[itemOS]; ok {
		ret.OS = v
	}

	if v, ok := result[itemDescription]; ok {
		ret.Description = v
	}

	return ret, nil
}

func (impl eventHandler) parseComment(assigner *sdk.UserHook, body string) (parseCommentResult, error) {
	result, err := impl.parse(sortOfCommentItems, assigner, body)
	if err != nil {
		return parseCommentResult{}, err
	}

	var ret parseCommentResult
	if v, ok := result[itemInfluence]; ok {
		ret.Influence = v
	}

	if v, ok := result[itemSeverityLevel]; ok {
		ret.SeverityLevel = v
	}

	if v, ok := result[itemAffectedVersion]; ok {
		allVersionResult, verison, err := impl.parseVersion(v, assigner)
		if err != nil {
			return parseCommentResult{}, err
		}

		ret.AllVersionResult = allVersionResult
		ret.AffectedVersion = verison
	}

	if v, ok := result[itemAbi]; ok {
		AllAbiResult, abi, err := impl.parseVersion(v, assigner)
		if err != nil {
			return parseCommentResult{}, err
		}

		ret.AllAbiResult = AllAbiResult
		ret.Abi = abi
	}

	return ret, nil
}

func (impl eventHandler) parse(items []string, assigner *sdk.UserHook, body string) (map[string]string, error) {
	var assign string
	if assigner != nil {
		assign = "@" + assigner.Name
	}

	mr := utils.NewMultiErrors()
	genErr := func(item string) string {
		return fmt.Sprintf("%s %s=> 没有按正确格式填写", assign, item)
	}

	parseResult := make(map[string]string)
	for _, item := range items {
		match := regexpOfItems[item].FindAllStringSubmatch(body, -1)
		if len(match) < 1 || len(match[regMatchResult]) < 3 {
			mr.Add(fmt.Sprintf("%s %s=> 没有按正确格式填写", assign, itemName[item]))
			continue
		}

		matchItem := match[regMatchResult][regMatchItem]
		trimItemInfo := localutils.TrimString(matchItem)
		if trimItemInfo == "" {
			mr.Add(fmt.Sprintf("%s 不允许为空", itemName[item]))
			continue
		}

		if _, ok := noTrimItem[item]; ok {
			parseResult[item] = matchItem
		} else {
			parseResult[item] = trimItemInfo
		}

		switch item {
		case itemSeverityLevel:
			if _, exist := severityLevelMap[parseResult[item]]; !exist {
				mr.Add(genErr(itemName[itemSeverityLevel]))
			}
		case itemOS:
			maintainVersion := sets.NewString(impl.cfg.MaintainVersion...)
			if !maintainVersion.Has(parseResult[item]) {
				mr.Add(genErr(itemName[itemOS]))
			}
		case itemComponents:
			split := strings.Split(parseResult[item], "-")
			if len(split) < 2 {
				mr.Add(genErr(itemName[itemComponents]))
			}
		}
	}

	return parseResult, mr.Err()
}

func (impl eventHandler) parseVersion(s string, assigner *sdk.UserHook) (versionAnalysisResult, affectedVersion []string, err error) {
	var assign string
	if assigner != nil {
		assign = "@" + assigner.Name
	}

	reg := regexp.MustCompile(`(openEuler.*?)[:：]\s*([受影响|不受影响|是|否]+)`)
	matches := reg.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return nil, nil, fmt.Errorf(fmt.Sprintf("@%s 请对受影响版本排查/abi变化进行分析", assigner.Name))
	}

	var allVersion []string
	for _, v := range matches {
		allVersion = append(allVersion, v[1])

		if v[2] == "受影响" || v[2] == "不受影响" || v[2] == "是" || v[2] == "否" {
			versionAnalysisResult = append(versionAnalysisResult, v[0])
		}

		if v[2] == "受影响" || v[2] == "是" {
			affectedVersion = append(affectedVersion, v[1])
		}
	}

	reg2 := regexp.MustCompile("受影响(.+)$")
	for i, v := range allVersion {
		if strings.Contains(v, "受影响") {
			matches := reg2.FindAllStringSubmatch(v, -1)

			if len(matches) > 0 {
				lastMatch := matches[len(matches)-1]
				allVersion[i] = lastMatch[1]
			}
		}
	}

	av := sets.NewString(allVersion...)

	var missingVersions []string

	for _, version := range impl.cfg.MaintainVersion {
		if !av.Has(version) {
			missingVersions = append(missingVersions, version)
		}
	}

	missingVersionsString := strings.Join(missingVersions, ",")

	if len(missingVersions) > 0 {
		return nil, nil, fmt.Errorf(fmt.Sprintf(commentVersionTip, assign, missingVersionsString))
	}

	return versionAnalysisResult, affectedVersion, nil
}
