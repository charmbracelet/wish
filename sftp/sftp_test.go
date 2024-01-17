package sftp

import "testing"

func TestSubsystem(t *testing.T) {
	if ss := Subsystem(); ss == nil {
		t.Error("returned subsystem should not be nil")
	}
}
