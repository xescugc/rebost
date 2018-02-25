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
