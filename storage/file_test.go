package storage

import (
	"bytes"
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func TestPath(t *testing.T) {
	v := createVolumne()
	defer v.Clean()

	f := v.newFileFromSignature("aaabbbcccddd")

	ep := path.Join(v.fileDir, "aaa", "bbb", "ccc", "ddd")
	if f.Path() != ep {
		t.Errorf("Expected %q to be %q", f.Path(), ep)
	}
}

func Test_ensurePath(t *testing.T) {
	v := createVolumne()
	defer v.Clean()

	f := v.newFileFromSignature("aaabbbcccddd")

	fp := path.Join(v.fileDir, "aaa", "bbb", "ccc", "ddd")
	dir := path.Join(v.fileDir, "aaa", "bbb", "ccc")

	if _, err := os.Stat(dir); err != nil && !os.IsNotExist(err) {
		t.Errorf("Expected to find error %q, found %q", os.ErrNotExist, err)
	}

	p, err := f.ensurePath()

	ExpectedNoError(t, err)

	if p != fp {
		t.Errorf("Expected path to be %q, found %q", fp, p)
	}

	if _, err = os.Stat(dir); err != nil {
		t.Errorf("Expected to find dir and no errors, found %s", err)
	}
}

func Test_store(t *testing.T) {
	v := createVolumne()
	defer v.Clean()

	content := []byte("test body")
	f := v.newFileFromKey("test")
	ef := v.newFileFromSignature("a140e4eff89659e835b76f8fef7da83e096a91ff")

	if _, err := os.Stat(ef.Path()); !os.IsNotExist(err) {
		t.Errorf("Expected to find error %q, found %q", os.ErrNotExist, err)
	}

	err := f.store(bytes.NewBuffer(content))

	ExpectedNoError(t, err)

	if f.Signature != ef.Signature {
		t.Errorf("Expected signature to be %q, found %q", ef.Signature, f.Signature)
	}

	if _, err = os.Stat(ef.Path()); err != nil {
		t.Errorf("Expected to find file and no errors, found %s", err)
	}

	b, err := ioutil.ReadFile(f.Path())

	ExpectedNoError(t, err)

	if string(b) != string(content) {
		t.Errorf("Expected the content of the file to be %q and found %q", b, content)
	}

}

func Test_remove(t *testing.T) {
	v := createVolumne()
	defer v.Clean()

	content := []byte("test body")
	f := v.newFileFromKey("test")
	ef := v.newFileFromSignature("a140e4eff89659e835b76f8fef7da83e096a91ff")

	if _, err := os.Stat(ef.Path()); !os.IsNotExist(err) {
		t.Errorf("Expected to find error %q, found %q", os.ErrNotExist, err)
	}

	err := f.store(bytes.NewBuffer(content))

	ExpectedNoError(t, err)

	if _, err = os.Stat(f.Path()); err != nil {
		t.Errorf("Expected to find file and no errors, found %s", err)
	}

	err = f.remove()

	ExpectedNoError(t, err)

	if _, err := os.Stat(f.Path()); !os.IsNotExist(err) {
		t.Errorf("Expected to find error %q, found %q", os.ErrNotExist, err)
	}
}
