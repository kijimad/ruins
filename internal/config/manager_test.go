package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 以下は t.Setenv でプロセス環境変数を書き換えて Load の分岐を検証するため、
// t.Setenv の制約上 t.Parallel は呼ばない。

func TestLoad_RUINS_PROFILEを指定するとプロファイルへ反映される(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("RUINS_PROFILE", "development")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, ProfileDevelopment, cfg.Profile)
	assert.True(t, cfg.SkipOpening) // 開発プロファイル固有のデフォルト値
}

func TestLoad_環境変数の型が不正だとエラーを返す(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("RUINS_PPROF_PORT", "not-a-number")

	_, err := Load()

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to load config")
}

func TestLoad_Validateに失敗するとエラーを返す(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("RUINS_WINDOW_WIDTH", "100") // Validate の最小値320を下回る

	_, err := Load()

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to validate config")
}
