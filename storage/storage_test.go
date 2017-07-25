package storage

import (
	"os"
	"testing"

	"github.com/xescugc/rebost/config"
)

func TestNew(t *testing.T) {
	c := &config.Config{
		Volumes: []string{"./volume1", "./volume2"},
	}

	s := New(c)
	defer func() {
		for _, v := range s.volumes {
			v.(*volume).Clean()
		}
	}()

	if len(s.volumes) != 2 {
		t.Errorf("Expected to have length of 2 and had %d", len(s.volumes))
	}

	for _, v := range []string{"./volume1", "./volume2"} {
		if _, err := os.Stat(v); err != nil {
			t.Errorf("Expected to find no errors, found %s", err)
		}
	}
}

//func TestGetFileWithMultiplesVolumes(t *testing.T) {
//c := &config.Config{
//Volumes: []string{"./volume1", "./volume2"},
//}

//s := New(c)
//defer func() {
//for _, v := range s.volumes {
//v.(*volume).Clean()
//}
//}()
//}
