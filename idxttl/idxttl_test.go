package idxttl_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/xescugc/rebost/idxttl"
)

func TestNew(t *testing.T) {
	eittl := &idxttl.IDXTTL{
		ExpiresAt:  time.Now(),
		Signatures: []string{"1", "2"},
	}

	ittl := idxttl.New(eittl.ExpiresAt, eittl.Signatures...)
	assert.Equal(t, eittl, ittl)
}

func TestAddSignature(t *testing.T) {
	ittl := &idxttl.IDXTTL{
		ExpiresAt:  time.Now(),
		Signatures: []string{"1", "2"},
	}
	eittl := &idxttl.IDXTTL{
		ExpiresAt:  ittl.ExpiresAt,
		Signatures: []string{"1", "2", "3", "4"},
	}

	ittl.AddSignatures("2", "3", "4")

	assert.Equal(t, eittl, ittl)
}
