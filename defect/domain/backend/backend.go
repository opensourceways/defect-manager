package backend

type CveBackend interface {
	MaxBulletinID() (int, error)
	PublishedDefects() ([]IssueNumAndVersion, error)
}

type IssueNumAndVersion struct {
	IssueNum string   `json:"issue_num"`
	Versions []string `json:"versions"`
}
