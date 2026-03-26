package model_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/storing/model"
)

func TestConfigRoundTrip(t *testing.T) {
	cfg := &config.Config{
		Port:    3805,
		Name:    "n1",
		Replica: 3,
		Tags:    map[string]string{"rack": "us-east-1", "env": "prod"},
	}
	m := model.ConfigToModel(cfg)
	assert.Equal(t, 3, m.Replica) // verifies the json:"replica" bug fix
	assert.Equal(t, cfg.Tags, m.Tags)

	back := model.ToConfig(m)
	assert.Equal(t, cfg.Replica, back.Replica)
	assert.Equal(t, cfg.Tags, back.Tags)
}
