package storage

import (
	"bytes"
	"os"
	"path"
	"reflect"
	"testing"
)

var (
	rootDir = "./data"
)

func createVolumne() *volume {
	v := NewVolume(rootDir)
	return v.(*volume)
}

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

func TestAddFile(t *testing.T) {
	v := createVolumne()
	defer v.Clean()

	content := []byte("test body")
	ef := v.newFile("test", "a140e4eff89659e835b76f8fef7da83e096a91ff")

	f, err := v.AddFile("test", bytes.NewBuffer(content))

	ExpectedNoError(t, err)

	if !reflect.DeepEqual(f, ef) {
		t.Errorf("Expected %#v but found %#v", ef, f)
	}
}

func TestGetFile(t *testing.T) {
	v := createVolumne()
	defer v.Clean()

	content := []byte("test body")
	ef := v.newFile("test", "a140e4eff89659e835b76f8fef7da83e096a91ff")

	f, err := v.GetFile("test")
	ExpectedNoError(t, err)

	if f != nil {
		t.Errorf("Expected no File but found %#v", f)
	}

	f, err = v.AddFile("test", bytes.NewBuffer(content))
	ExpectedNoError(t, err)

	if !reflect.DeepEqual(f, ef) {
		t.Errorf("Expected %#v but found %#v", ef, f)
	}
}

func TestDeleteFile(t *testing.T) {
	v := createVolumne()
	defer v.Clean()

	content := []byte("test body")
	_, err := v.AddFile("test", bytes.NewBuffer(content))
	ExpectedNoError(t, err)

	ok, err := v.HasFile("test")
	ExpectedNoError(t, err)

	if !ok {
		t.Errorf("Expected to find File but found no one")
	}

	err = v.DeleteFile("test")

	ok, err = v.HasFile("test")
	ExpectedNoError(t, err)

	if ok {
		t.Errorf("Expected to find no File but found one")
	}

}

func TestHasFile(t *testing.T) {
	v := createVolumne()
	defer v.Clean()

	ok, err := v.HasFile("test")
	ExpectedNoError(t, err)

	if ok {
		t.Errorf("Expected not to find File but found one")
	}

	content := []byte("test body")
	_, err = v.AddFile("test", bytes.NewBuffer(content))
	ExpectedNoError(t, err)

	ok, err = v.HasFile("test")
	ExpectedNoError(t, err)

	if !ok {
		t.Errorf("Expected to File but found none")
	}
}
