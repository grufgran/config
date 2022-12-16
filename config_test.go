package config

import (
	config "config/context"
	"testing"
)

func TestNewConfigFromFile(t *testing.T) {
	basePaths := map[string]string{"site": "/site"}
	ctx := config.NewContext(basePaths, "test2", "recuired_role")
	ctx.SetConfRoot("testdata/")

	conf, err := NewConfigFromFile(ctx, "testdata/test1.conf", nil)
	t.Log(conf, err)
}
