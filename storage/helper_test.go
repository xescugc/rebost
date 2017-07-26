package storage

import "testing"

func ExpectedNoError(t *testing.T, err error) {
	if err != nil {
		t.Errorf("Expected no errors when storing and found %q", err)
	}
}
