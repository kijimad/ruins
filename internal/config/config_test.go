package config

import (
	"os"
	"testing"

	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyProfileDefaults_Production(t *testing.T) {
	t.Parallel()

	cfg := &Config{Profile: ProfileProduction}
	cfg.ApplyProfileDefaults()

	// 環境変数が未設定の場合にデフォルト値が適用される
	// テスト実行時に一部の環境変数が設定されている可能性があるため、
	// 環境変数に依存しないフィールドのみ検証する
	assert.Equal(t, 960, cfg.User.WindowWidth)
	assert.Equal(t, 720, cfg.User.WindowHeight)
	assert.False(t, cfg.SkipOpening)
	assert.Equal(t, 60, cfg.TargetFPS)
}

func TestApplyProfileDefaults_Development(t *testing.T) {
	t.Parallel()

	cfg := &Config{Profile: ProfileDevelopment}
	cfg.ApplyProfileDefaults()

	assert.Equal(t, 960, cfg.User.WindowWidth)
	assert.Equal(t, 720, cfg.User.WindowHeight)
	assert.True(t, cfg.SkipOpening)
}

func TestApplyProfileDefaults_UnknownProfile(t *testing.T) {
	t.Parallel()

	cfg := &Config{Profile: Profile("unknown")}
	cfg.ApplyProfileDefaults()

	// 不明なプロファイルは本番設定にフォールバックする
	assert.False(t, cfg.Debug)
	assert.False(t, cfg.SkipOpening)
}

func TestConfig_StringGolden(t *testing.T) {
	t.Parallel()

	c := &Config{
		Profile:     ProfileDevelopment,
		User:        UserConfig{WindowWidth: 800, WindowHeight: 600},
		Debug:       true,
		LogLevel:    "debug",
		Seed:        42,
		TargetFPS:   30,
		PProfPort:   6060,
		ProfilePath: ".",
	}

	assertGoldenText(t, "config_string", c.String())
}

// assertGoldenText は文字列を testdata/<name>.golden と比較する。
// GOLDIE_UPDATE=1 で golden を更新する。make updategolden から拾えるようテスト名には Golden を含める。
func assertGoldenText(t *testing.T, name, actual string) {
	t.Helper()

	g := goldie.New(t, goldie.WithFixtureDir("testdata"), goldie.WithNameSuffix(".golden"))
	if v := os.Getenv("GOLDIE_UPDATE"); v == "1" || v == "true" {
		require.NoError(t, g.Update(t, name, []byte(actual)))
		return
	}
	g.Assert(t, name, []byte(actual))
}

func TestLoad(t *testing.T) {
	t.Parallel()

	cfg, err := Load()
	require.NoError(t, err)

	assert.NotNil(t, cfg)
	assert.NotNil(t, cfg.RNG)
	assert.Positive(t, cfg.User.WindowWidth)
	assert.Positive(t, cfg.User.WindowHeight)
	assert.Positive(t, cfg.TargetFPS)
}
func TestValidate(t *testing.T) {
	t.Parallel()

	// Validate は最初に見つかった不正値でエラーを返すため、各ケースを「1フィールドだけ不正」に
	// する必要がある。妥当なベースラインを返して1箇所だけ変異させることで、変異したフィールドが
	// 唯一のエラー要因になり、そのフィールドを正しく検証できる。
	valid := func() *Config {
		return &Config{
			User:      UserConfig{WindowWidth: 1920, WindowHeight: 1080},
			TargetFPS: 144,
			PProfPort: 8080,
		}
	}

	t.Run("有効な値はエラーを返さない", func(t *testing.T) {
		t.Parallel()

		assert.NoError(t, valid().Validate())
	})

	t.Run("不正な値はエラーを返す", func(t *testing.T) {
		t.Parallel()

		// wantErr で「変異したフィールドがまさにそのエラーを起こす」ことを検証する
		cases := []struct {
			name    string
			mutate  func(*Config)
			wantErr error
		}{
			{"ウィンドウ幅が最小値未満", func(c *Config) { c.User.WindowWidth = 100 }, errWindowWidthTooSmall},
			{"ウィンドウ高さが最小値未満", func(c *Config) { c.User.WindowHeight = 50 }, errWindowHeightTooSmall},
			{"目標FPSが1未満", func(c *Config) { c.TargetFPS = 0 }, errTargetFPSInvalid},
			{"pprofポートが下限未満", func(c *Config) { c.PProfPort = 80 }, errPProfPortOutOfRange},
			{"pprofポートが上限超過", func(c *Config) { c.PProfPort = 70000 }, errPProfPortOutOfRange},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				cfg := valid()
				tc.mutate(cfg)
				assert.ErrorIs(t, cfg.Validate(), tc.wantErr)
			})
		}
	})
}
