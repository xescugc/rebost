package storage

import (
	"os"
	"path"
	"reflect"
	"testing"
)

var (
	rootDir = "./data"
)

func createVolumne() *volume { return NewVolume(rootDir) }

func TestNewVolumeAndClean(t *testing.T) {
	var err error
	if _, err = os.Stat(rootDir); err != nil && !os.IsNotExist(err) {
		t.Errorf("Expected to find a %s, found a %s", os.ErrNotExist, err)
	}

	if _, err = os.Stat(path.Join(rootDir, "tmps")); err != nil && !os.IsNotExist(err) {
		t.Errorf("Expected to find a %s, found a %s", os.ErrNotExist, err)
	}

	if _, err = os.Stat(path.Join(rootDir, "file")); err != nil && !os.IsNotExist(err) {
		t.Errorf("Expected to find a %s, found a %s", os.ErrNotExist, err)
	}

	if _, err = os.Stat(path.Join(rootDir, "volume.index")); err != nil && !os.IsNotExist(err) {
		t.Errorf("Expected to find a %s, found a %s", os.ErrNotExist, err)
	}

	v := createVolumne()

	if _, err = os.Stat(rootDir); err != nil {
		t.Errorf("Expected to find no errors, found %s", err)
	}

	if _, err = os.Stat(path.Join(rootDir, "tmps")); err != nil {
		t.Errorf("Expected to find no errors, found %s", err)
	}

	if _, err = os.Stat(path.Join(rootDir, "file")); err != nil {
		t.Errorf("Expected to find no errors, found %s", err)
	}

	if _, err = os.Stat(path.Join(rootDir, "volume.index")); err != nil {
		t.Errorf("Expected to find no errors, found %s", err)
	}

	v.Clean()

	if _, err = os.Stat(rootDir); err != nil && !os.IsNotExist(err) {
		t.Errorf("Expected to find a %s, found a %s", os.ErrNotExist, err)
	}

	if _, err = os.Stat(path.Join(rootDir, "tmps")); err != nil && !os.IsNotExist(err) {
		t.Errorf("Expected to find a %s, found a %s", os.ErrNotExist, err)
	}

	if _, err = os.Stat(path.Join(rootDir, "file")); err != nil && !os.IsNotExist(err) {
		t.Errorf("Expected to find a %s, found a %s", os.ErrNotExist, err)
	}

	if _, err = os.Stat(path.Join(rootDir, "volume.index")); err != nil && !os.IsNotExist(err) {
		t.Errorf("Expected to find a %s, found a %s", os.ErrNotExist, err)
	}
}

func Test_newFile(t *testing.T) {
	v := createVolumne()
	defer v.Clean()

	k := "key"
	s := "signature"

	tests := []struct {
		v *File
		e *File
	}{
		{
			v.newFile(k, s),
			&File{key: k, Signature: s, volume: v},
		},
		{
			v.newFileFromKey(k),
			&File{key: k, volume: v},
		},
		{
			v.newFileFromSignature(s),
			&File{Signature: s, volume: v},
		},
	}

	for _, test := range tests {
		if !reflect.DeepEqual(test.v, test.e) {
			t.Errorf("Expected\n%#v\nto be\n%#v\n", test.v, test.e)
		}
	}

}
