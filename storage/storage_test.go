package storage

import (
	"bytes"
	"os"
	"testing"

	"github.com/xescugc/rebost/config"
)

func TestNew(t *testing.T) {
	c := &config.Config{
		Volumes: []string{"./volume1", "./volume2"},
	}

	s := New(c)
	defer s.Clean()

	if len(s.localVolumes) != 2 {
		t.Errorf("Expected to have length of 2 and had %d", len(s.localVolumes))
	}

	for _, v := range []string{"./volume1", "./volume2"} {
		if _, err := os.Stat(v); err != nil {
			ExpectedNoError(t, err)
		}
	}
}

func TestImplementsVolumeInterface(t *testing.T) {
	c := &config.Config{
		Volumes: []string{"./volume1", "./volume2"},
	}

	s := New(c)
	defer s.Clean()

	var v Volume = s
	_ = v

}

func TestAddFileWithMultipleVolumes(t *testing.T) {
	c := &config.Config{
		Volumes: []string{"./volume1", "./volume2"},
	}

	s := New(c)
	defer s.Clean()

	for _, v := range s.localVolumes {
		ok, err := v.HasFile("test")
		ExpectedNoError(t, err)
		if ok {
			t.Errorf("Expected to find no File but found one")
		}
	}

	content := []byte("test body")
	f, err := s.AddFile("test", bytes.NewBuffer(content))
	ExpectedNoError(t, err)

	for _, v := range s.localVolumes {
		if v.(*volume).rootDir == f.volume.rootDir {
			ok, err := v.HasFile("test")
			ExpectedNoError(t, err)
			if !ok {
				t.Errorf("Expected to find File but found no one")
			}
		} else {
			ok, err := v.HasFile("test")
			ExpectedNoError(t, err)
			if ok {
				t.Errorf("Expected to find no File but found one")
			}
		}
	}

}

func TestHasFileWithMultipleVolumes(t *testing.T) {
	c := &config.Config{
		Volumes: []string{"./volume1", "./volume2"},
	}

	s := New(c)
	defer s.Clean()

	ok, err := s.HasFile("test")
	ExpectedNoError(t, err)

	if ok {
		t.Errorf("Expected to find no File but found one")
	}

	content := []byte("test body")
	_, err = s.AddFile("test", bytes.NewBuffer(content))
	ExpectedNoError(t, err)

	ok, err = s.HasFile("test")
	ExpectedNoError(t, err)

	if !ok {
		t.Errorf("Expected to find File but found no one")
	}
}
