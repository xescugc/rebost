package mock

import (
	"os"
	"time"

	"github.com/spf13/afero"
)

type Fs struct {
	CreateFn         func(name string) (afero.File, error)
	CreateInvoked    bool
	MkdirFn          func(name string, perm os.FileMode) error
	MkdirInvoked     bool
	MkdirAllFn       func(path string, perm os.FileMode) error
	MkdirAllInvoked  bool
	OpenFn           func(name string) (afero.File, error)
	OpenInvoked      bool
	OpenFileFn       func(name string, flag int, perm os.FileMode) (afero.File, error)
	OpenFileInvoked  bool
	RemoveFn         func(name string) error
	RemoveInvoked    bool
	RemoveAllFn      func(path string) error
	RemoveAllInvoked bool
	RenameFn         func(oldname, newname string) error
	RenameInvoked    bool
	StatFn           func(name string) (os.FileInfo, error)
	StatInvoked      bool
	NameFn           func() string
	NameInvoked      bool
	ChmodFn          func(name string, mode os.FileMode) error
	ChmodInvoked     bool
	ChtimesFn        func(name string, atime time.Time, mtime time.Time) error
	ChtimesInvoked   bool
}

func (m *Fs) Create(name string) (afero.File, error) {
	m.CreateInvoked = true
	return m.CreateFn(name)
}
func (m *Fs) Mkdir(name string, perm os.FileMode) error {
	m.MkdirInvoked = true
	return m.MkdirFn(name, perm)
}
func (m *Fs) MkdirAll(path string, perm os.FileMode) error {
	m.MkdirAllInvoked = true
	return m.MkdirAllFn(path, perm)
}
func (m *Fs) Open(name string) (afero.File, error) {
	m.OpenInvoked = true
	return m.OpenFn(name)
}
func (m *Fs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	m.OpenFileInvoked = true
	return m.OpenFileFn(name, flag, perm)
}
func (m *Fs) Remove(name string) error {
	m.RemoveInvoked = true
	return m.RemoveFn(name)
}
func (m *Fs) RemoveAll(path string) error {
	m.RemoveAllInvoked = true
	return m.RemoveAllFn(path)
}
func (m *Fs) Rename(oldname, newname string) error {
	m.RenameInvoked = true
	return m.RenameFn(oldname, newname)
}
func (m *Fs) Stat(name string) (os.FileInfo, error) {
	m.StatInvoked = true
	return m.StatFn(name)
}
func (m *Fs) Name() string {
	m.NameInvoked = true
	return m.NameFn()
}
func (m *Fs) Chmod(name string, mode os.FileMode) error {
	m.ChmodInvoked = true
	return m.ChmodFn(name, mode)
}
func (m *Fs) Chtimes(name string, atime time.Time, mtime time.Time) error {
	m.ChtimesInvoked = true
	return m.ChtimesFn(name, atime, mtime)
}
