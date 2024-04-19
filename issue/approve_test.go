package issue

import (
	"testing"
)

func TestAssigner(t *testing.T) {
	InitCommitterInstance()

	CommitterInstance.InitCommitterCache()
	b := CommitterInstance.isCommitter("src-openeuler/A-Ops", "luanjianhai")
	if !b {
		t.Failed()
	}
}
