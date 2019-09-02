package file_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xescugc/rebost/file"
)

func TestFilePath(t *testing.T) {
	f := file.File{Signature: "1231231232"}
	assert.Equal(t, "root/12/31/23/12/32", f.Path("root"))
}

func TestPath(t *testing.T) {
	assert.Equal(t, "root/12/31/23/12/32", file.Path("root", "1231231232"))
}

func TestFileDeleteVolumeID(t *testing.T) {
	tests := []struct {
		Name       string
		VolumeIDs  []string
		VolumeID   string
		EVolumeIDs []string
	}{
		{
			Name:       "Success2WithBeginning",
			VolumeIDs:  []string{"a", "b"},
			VolumeID:   "a",
			EVolumeIDs: []string{"b"},
		},
		{
			Name:       "Success2WithEnd",
			VolumeIDs:  []string{"a", "b"},
			VolumeID:   "b",
			EVolumeIDs: []string{"a"},
		},
		{
			Name:       "Success3WithBeginning",
			VolumeIDs:  []string{"a", "b", "c"},
			VolumeID:   "a",
			EVolumeIDs: []string{"b", "c"},
		},
		{
			Name:       "Success3WithMiddle",
			VolumeIDs:  []string{"a", "b", "c"},
			VolumeID:   "b",
			EVolumeIDs: []string{"a", "c"},
		},
		{
			Name:       "Success3WithEnd",
			VolumeIDs:  []string{"a", "b", "c"},
			VolumeID:   "c",
			EVolumeIDs: []string{"a", "b"},
		},
		{
			Name:       "SuccessNotFound",
			VolumeIDs:  []string{"a", "b", "c"},
			VolumeID:   "d",
			EVolumeIDs: []string{"a", "b", "c"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			f := file.File{VolumeIDs: tt.VolumeIDs}
			f.DeleteVolumeID(tt.VolumeID)
			assert.Equal(t, tt.EVolumeIDs, f.VolumeIDs)
		})
	}
}
