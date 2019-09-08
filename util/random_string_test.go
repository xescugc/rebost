package util_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xescugc/rebost/util"
)

func TestRandomString(t *testing.T) {
	l := 3
	s := util.RandomString(l)
	s2 := util.RandomString(l)

	assert.Len(t, s, l)
	assert.Len(t, s2, l)

	assert.NotEqual(t, s, s2)
}
