package i18n

import (
	"bufio"
	"os"
	"sort"
	"strings"
	"testing"
	"unicode"

	"github.com/kijimaD/ruins/internal/raw"
	"github.com/stretchr/testify/require"
)

// debtAllowlistPath は英語化済みだが未訳の name を暫定的に許す allowlist のパス。
const debtAllowlistPath = "testdata/raw_i18n_debt.txt"

// TestRawTranslationCoverage は raw.toml の表示される name/description が英語化済みなら ja.po に
// 訳を持つことを検証する。既定言語が ja のため、英語化して ja.po を書き忘れると日本語プレイヤーへ
// 英語が漏れる。それを fail-closed で止める。CDDA の抽出ツール相当だが、既定 en の CDDA と違い
// 既定 ja の ruins は未訳を許さないので完全性ゲートも兼ねる。
//
// 既存の未訳債務は debtAllowlistPath へ退避し、新規の英語化漏れだけを止めるラチェットにする。
// 訳を足したら allowlist から削除する。allowlist == 現在の債務、を両方向で強制して確実に縮める。
func TestRawTranslationCoverage(t *testing.T) {
	t.Parallel()

	defined := parsePoMsgids(t)
	raws, err := raw.LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)

	// プレイヤーに表示される文字列だけを収集する。Prop/Tile の Description は spawn 時に
	// コンポーネントへ入るが読み出す表示経路が無いので対象外。テーブル名も内部キーで非表示。
	items, members, props := raw.PtrSlice(raws.Items), raw.PtrSlice(raws.Members), raw.PtrSlice(raws.Props)
	tiles, profs := raw.PtrSlice(raws.Tiles), raw.PtrSlice(raws.Professions)
	texts := make([]string, 0, len(items)*2+len(members)+len(props)+len(tiles)+len(profs)*2)
	for _, it := range items {
		texts = append(texts, it.Name, it.Description)
	}
	for _, m := range members {
		texts = append(texts, m.Name)
	}
	for _, p := range props {
		texts = append(texts, p.Name)
	}
	for _, tl := range tiles {
		texts = append(texts, tl.Name)
	}
	for _, pr := range profs {
		texts = append(texts, pr.Name, pr.Description)
	}

	// 現在の債務: 英語化済みだが ja.po に無い表示文字列。日本語のままのものは移行前なので数えるだけ。
	seen := map[string]bool{}
	missing := map[string]bool{}
	unmigrated := 0
	for _, s := range texts {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		if hasJapanese(s) {
			unmigrated++
			continue
		}
		if !defined[s] {
			missing[s] = true
		}
	}

	allow := loadDebtAllowlist(t)

	// 新規債務: allowlist に無い未訳。英語化して ja.po を書き忘れた回帰。
	var unexpected []string
	for s := range missing {
		if !allow[s] {
			unexpected = append(unexpected, s)
		}
	}
	// 解消済み: allowlist にあるが既に訳あり、または表示文字列でなくなったもの。縮小を強制する。
	var stale []string
	for s := range allow {
		if !missing[s] {
			stale = append(stale, s)
		}
	}
	sort.Strings(unexpected)
	sort.Strings(stale)

	t.Logf("未英語化(日本語) %d, 債務 allowlist %d, 新規債務 %d, 解消済み %d",
		unmigrated, len(allow), len(unexpected), len(stale))
	require.Empty(t, unexpected,
		"英語化した raw の name/description で ja.po に訳が無い。ja.po に訳を足すか、暫定なら %s に追記する:\n%s",
		debtAllowlistPath, strings.Join(unexpected, "\n"))
	require.Empty(t, stale,
		"%s に不要な行がある。訳済みか表示対象外になったので該当行を削除する:\n%s",
		debtAllowlistPath, strings.Join(stale, "\n"))
}

// loadDebtAllowlist は debtAllowlistPath を読み、# 始まりと空行を除いた集合を返す。
func loadDebtAllowlist(t *testing.T) map[string]bool {
	t.Helper()
	f, err := os.Open(debtAllowlistPath)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	allow := map[string]bool{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), " \t")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		allow[line] = true
	}
	require.NoError(t, sc.Err())
	return allow
}

// hasJapanese は文字列に日本語(漢字・ひらがな・カタカナ)が含まれるかを返す。
func hasJapanese(s string) bool {
	for _, r := range s {
		if unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana) {
			return true
		}
	}
	return false
}
