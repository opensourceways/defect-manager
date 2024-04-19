package issue

type Config struct {
	RobotToken string `json:"robot_token"      required:"true"`
	IssueType  string `json:"issue_type"       required:"true"`
	//EnterpriseToken string           `json:"enterprise_token" required:"true"`
	//EnterpriseId    string           `json:"enterprise_id"    required:"true"`
	MaintainVersion []string `json:"maintain_version" required:"true"`
	//PkgPolicy       []map[string]int `json:"pkg_policy"     required:"true"`
}
