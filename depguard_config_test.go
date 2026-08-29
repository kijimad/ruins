package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// depguard はパターンがどのパッケージにもマッチしなくても黙って通す。リネームや移動で
// 設定のパスが実体から外れると、違反コードが残ったまま検査だけが消える。設定が指す
// パスの実在を確かめて、追随漏れをテスト赤にする
type golangciConfig struct {
	Linters struct {
		Settings struct {
			Depguard struct {
				Rules map[string]struct {
					Files []string `yaml:"files"`
					Allow []string `yaml:"allow"`
					Deny  []struct {
						Pkg string `yaml:"pkg"`
					} `yaml:"deny"`
				} `yaml:"rules"`
			} `yaml:"depguard"`
		} `yaml:"settings"`
	} `yaml:"linters"`
}

// staleDepguardPaths は設定が指すパスのうち実在しないものを列挙する。
// exists はリポジトリ相対パスの実在判定で、テストから差し替える
func staleDepguardPaths(config []byte, module string, exists func(string) bool) ([]string, error) {
	var cfg golangciConfig
	if err := yaml.Unmarshal(config, &cfg); err != nil {
		return nil, err
	}

	var stale []string
	for name, rule := range cfg.Linters.Settings.Depguard.Rules {
		for _, pattern := range rule.Files {
			// 先頭の ! は除外指定、前後の ** は任意階層のワイルドカード。実在確認にかける
			// リポジトリ相対パスだけを取り出す。ディレクトリ指定は前後に ** が付き、
			// ファイル直指定は先頭だけに付く。$all・$test・$gostd は golangci の特殊トークン
			rel := strings.TrimPrefix(pattern, "!")
			if strings.HasPrefix(rel, "$") {
				continue
			}
			rel = strings.TrimSuffix(strings.TrimPrefix(rel, "**/"), "/**")
			if !exists(rel) {
				stale = append(stale, name+".files: "+pattern)
			}
		}

		// 他モジュールと $gostd は自リポジトリに実体が無いので確認できない。
		// 末尾の $ は正規表現のアンカーなので外す
		checkPkg := func(pkg string) {
			if !strings.HasPrefix(pkg, module) {
				return
			}
			rel := strings.Trim(strings.TrimPrefix(strings.TrimSuffix(pkg, "$"), module), "/")
			if rel != "" && !exists(rel) {
				stale = append(stale, name+".pkg: "+pkg)
			}
		}
		for _, pkg := range rule.Allow {
			checkPkg(pkg)
		}
		for _, deny := range rule.Deny {
			checkPkg(deny.Pkg)
		}
	}

	return stale, nil
}

// modulePath は go.mod の module 行からモジュールパスを読む
func modulePath(t *testing.T) string {
	t.Helper()

	body, err := os.ReadFile("go.mod")
	require.NoError(t, err)
	for line := range strings.Lines(string(body)) {
		if after, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
			return strings.TrimSpace(after)
		}
	}
	require.Fail(t, "go.mod に module 行が無い")

	return ""
}

func TestStaleDepguardPaths_設定が指すパスがすべて実在する(t *testing.T) {
	t.Parallel()

	config, err := os.ReadFile(".golangci.yml")
	require.NoError(t, err)

	stale, err := staleDepguardPaths(config, modulePath(t), func(rel string) bool {
		_, err := os.Stat(filepath.FromSlash(rel))

		return err == nil
	})
	require.NoError(t, err)
	assert.Empty(t, stale, "depguard が実在しないパスを指している。リネームや移動への追随漏れを疑う")
}

func TestStaleDepguardPaths_実在しないパスを列挙する(t *testing.T) {
	t.Parallel()

	config := []byte(`
linters:
  settings:
    depguard:
      rules:
        typo_guard:
          files:
            - "$all"
            - "!**/internal/widgets/uicoer/**"
          allow:
            - "$gostd"
            - "github.com/hajimehoshi/ebiten/v2"
          deny:
            - pkg: "github.com/kijimaD/ruins/internal/widgets/uicoer$"
            - pkg: "github.com/kijimaD/ruins/internal/widgets/uicore"
`)
	exists := func(rel string) bool { return rel == "internal/widgets/uicore" }

	stale, err := staleDepguardPaths(config, "github.com/kijimaD/ruins", exists)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"typo_guard.files: !**/internal/widgets/uicoer/**",
		"typo_guard.pkg: github.com/kijimaD/ruins/internal/widgets/uicoer$",
	}, stale)
}
