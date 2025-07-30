package config_test

import (
	"testing"

	"github.com/katallaxie/natz-operator/pkg/config"
	"github.com/katallaxie/pkg/cast"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	cfg := config.New()
	require.NotNil(t, cfg)
}

func TestDefault(t *testing.T) {
	t.Parallel()

	cfg := config.Default()
	require.NotNil(t, cfg)

	json, err := cfg.Marshal()
	require.NoError(t, err)
	require.JSONEq(t, `{"host":"0.0.0.0","port":4222}`, string(json))
}

func TestUnmarshal(t *testing.T) {
	t.Parallel()

	cfg := config.New()
	require.NotNil(t, cfg)

	err := cfg.Unmarshal([]byte(`{"host": "localhost", "port": 4223}`))
	require.NoError(t, err)
	require.Equal(t, "localhost", *cfg.Host)
	require.Equal(t, 4223, *cfg.Port)
}

func TestMarshal(t *testing.T) {
	t.Parallel()

	cfg := config.New()
	require.NotNil(t, cfg)

	cfg.Host = cast.Ptr("localhost")
	cfg.Port = cast.Ptr(4223)

	json, err := cfg.Marshal()
	require.NoError(t, err)
	require.JSONEq(t, `{"host":"localhost","port":4223}`, string(json))
}
