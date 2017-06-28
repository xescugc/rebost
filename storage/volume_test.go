package storage

import (
	"os"
	"path"
	"testing"
)

func TestNewVolume(t *testing.T) {
	// Root dir
	var err error
	rootDir := "./data"
	if _, err = os.Stat(rootDir); err != nil && !os.IsNotExist(err) {
		t.Errorf("Expected to find a %s, found a %s", os.ErrNotExist, err)
	}

	if _, err = os.Stat(path.Join(rootDir, "temps")); err != nil && !os.IsNotExist(err) {
		t.Errorf("Expected to find a %s, found a %s", os.ErrNotExist, err)
	}

	if _, err = os.Stat(path.Join(rootDir, "file")); err != nil && !os.IsNotExist(err) {
		t.Errorf("Expected to find a %s, found a %s", os.ErrNotExist, err)
	}

	_ := NewVolume(rootDir)

	if _, err = os.Stat(rootDir); err == nil {
		t.Errorf("Expected to find no errors, found %s", err)
	}

	if _, err = os.Stat(path.Join(rootDir, "temps")); err == nil {
		t.Errorf("Expected to find no errors, found %s", err)
	}

	if _, err = os.Stat(path.Join(rootDir, "file")); err == nil {
		t.Errorf("Expected to find no errors, found %s", err)
	}
}
