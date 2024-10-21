package issue

import (
	"fmt"
	"regexp"
	"strings"

	sdk "github.com/opensourceways/go-gitee/gitee"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/opensourceways/defect-manager/utils"
)

const (
	cmdCheck        = "/check-issue"
	unAffectedLabel = "DEFECT/UNAFFECTED"
	fixedLabel      = "DEFECT/FIXED"
	unFixedLabel    = "DEFECT/UNFIXED"

	itemDescription              = "description"
	itemOS                       = "os"
	itemKernel                   = "kernel"
	itemComponents               = "components"
	itemProblemReproductionSteps = "problemReproductionSteps"
	itemReferenceAndGuidanceUrl  = "referenceAndGuidanceUrl"
	itemInfluence                = "influence"
	itemSeverityLevel            = "severityLevel"
	itemRootCause                = "rootCause"
	itemAffectedVersion          = "affectedVersion"
	itemAbi                      = "abi"

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
	pVersion  = "受影响分支"
	pAbi      = "abi变化"

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
		itemDescription:              "缺陷描述",
		itemOS:                       "缺陷所属的os版本",
		itemKernel:                   "内核版本",
		itemComponents:               "缺陷所属软件及版本号",
		itemProblemReproductionSteps: "问题复现步骤",
		itemReferenceAndGuidanceUrl:  "详情及分析指导参考链接",
		itemInfluence:                "影响性分析说明",
		itemSeverityLevel:            "缺陷严重等级",
		itemRootCause:                "缺陷根因说明",
		itemAffectedVersion:          "受影响版本",
		itemAbi:                      "abi",
	}

	regexpOfItems = map[string]*regexp.Regexp{
		itemDescription:              regexp.MustCompile(`(缺陷描述)[】][:：]请补充详细的缺陷问题现象描述\*\*([\s\S]*?)\*\*一、缺陷信息`),
		itemOS:                       regexp.MustCompile(`(缺陷所属的os版本)[】][(（]如openEuler-22.03-LTS，参考命令"cat /etc/os-release"结果[)）]\*\*([\s\S]*?)\*\*【内核版本`),
		itemKernel:                   regexp.MustCompile(`(内核版本)[】][(（]如kernel-5.10.0-60.138.0.165，参考命令"uname -r"结果[)）]\*\*([\s\S]*?)\*\*【缺陷所属软件及版本号`),
		itemComponents:               regexp.MustCompile(`(缺陷所属软件及版本号)[】][(（]如kernel-5.10.0-60.138.0.165，参考命令"rpm -q 包名"结果[)）]\*\*([\s\S]*?)\*\*【环境信息`),
		itemProblemReproductionSteps: regexp.MustCompile(`(问题复现步骤)[】][:：]请描述具体的操作步骤\*\*([\s\S]*?)\*\*【实际结果`),
		itemReferenceAndGuidanceUrl:  regexp.MustCompile(`(缺陷详情及分析指导参考链接)[】]\*\*([\s\S]*?)\*\*二、缺陷分析结构反馈`),
		itemInfluence:                regexp.MustCompile(`(影响性分析说明)[:：]([\s\S]*?)缺陷严重等级`),
		itemSeverityLevel:            regexp.MustCompile(`(缺陷严重等级)[:：]\(Critical/High/Moderate/Low\)([\s\S]*?)缺陷根因说明`),
		itemRootCause:                regexp.MustCompile(`(缺陷根因说明)[:：]([\s\S]*?)受影响版本排查`),
		itemAffectedVersion:          regexp.MustCompile(`(受影响版本排查)\(受影响/不受影响\)[:：]([\s\S]*?)abi变化`),
		itemAbi:                      regexp.MustCompile(`(abi变化)\(是/否\)[:：]([\s\S]*?)$`),
	}

	sortOfIssueItems = []string{
		itemDescription,
		itemOS,
		itemKernel,
		itemComponents,
		itemProblemReproductionSteps,
		itemReferenceAndGuidanceUrl,
	}

	sortOfCommentItems = []string{
		itemInfluence,
		itemSeverityLevel,
		itemRootCause,
		itemAffectedVersion,
		itemAbi,
	}

	noTrimItem = map[string]bool{
		itemDescription: true,
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
	ReferenceURL     string
}

type parseCommentResult struct {
	Influence        string
	SeverityLevel    string
	RootCause        string
	AllVersionResult []string
	AllAbiResult     []string
	AffectedVersion  []string
	Abi              []string
}

func (impl eventHandler) parseIssue(assigner *sdk.UserHook, body string) (parseIssueResult, error) {
	var parseIssueParam = sortOfIssueItems

	result, err := impl.parse(parseIssueParam, assigner, body)
	if err != nil {
		logrus.Errorf("when parse issue error occurred: %v", err)
	}

	var ret parseIssueResult
	if v, ok := result[itemKernel]; ok {
		ret.Kernel = v
	}

	if v, ok := result[itemComponents]; ok {
		split := strings.Split(v, "-")
		if len(split) > 2 {
			ret.Component = strings.Join(split[:len(split)-1], "-")
			ret.ComponentVersion = split[len(split)-1]
		}
	}

	if v, ok := result[itemOS]; ok {
		ret.OS = v
	}

	if v, ok := result[itemDescription]; ok {
		ret.Description = v
	}

	if v, ok := result[itemReferenceAndGuidanceUrl]; ok {
		ret.ReferenceURL = v
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

	if v, ok := result[itemRootCause]; ok {
		ret.RootCause = v
	}

	if v, ok := result[itemAffectedVersion]; ok {
		allVersionResult, verison, err := impl.parseVersion(v, itemAffectedVersion, assigner)
		if err != nil {
			return parseCommentResult{}, err
		}

		ret.AllVersionResult = allVersionResult
		ret.AffectedVersion = verison
	}

	if v, ok := result[itemAbi]; ok {
		AllAbiResult, abi, err := impl.parseVersion(v, itemAbi, assigner)
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
		return fmt.Sprintf("%s %s=>没有按正确格式填写", assign, item)
	}

	parseResult := make(map[string]string)
	for _, item := range items {
		match := regexpOfItems[item].FindAllStringSubmatch(body, -1)
		if len(match) < 1 || len(match[regMatchResult]) < 3 {
			mr.Add(genErr(itemName[item]))
			continue
		}

		matchItem := match[regMatchResult][regMatchItem]
		trimItemInfo := utils.TrimString(matchItem)
		if trimItemInfo == "" {
			if item == itemRootCause && impl.isExistAffectedVersion(body) {
				mr.Add(fmt.Sprintf("%s %s=>不允许为空", assign, itemName[item]))
				continue
			}

			mr.Add(fmt.Sprintf("%s %s=>不允许为空", assign, itemName[item]))
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
		}
	}

	return parseResult, mr.Err()
}

func (impl eventHandler) parseVersion(s, item string, assigner *sdk.UserHook) (versionAnalysisResult, affectedVersion []string, err error) {
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
		if item == itemAffectedVersion {
			return nil, nil, fmt.Errorf(fmt.Sprintf(commentVersionTip, assign, missingVersionsString, pVersion))
		}
		return nil, nil, fmt.Errorf(fmt.Sprintf(commentVersionTip, assign, missingVersionsString, pAbi))
	}

	return versionAnalysisResult, affectedVersion, nil
}

func (impl eventHandler) isExistAffectedVersion(s string) bool {
	reg := regexp.MustCompile(`(openEuler.*?)[:：]\s*受影响`)
	matches := reg.FindAllStringSubmatch(s, -1)

	return len(matches) > 0
}
