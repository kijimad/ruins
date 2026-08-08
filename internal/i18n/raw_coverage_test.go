package i18n

import (
	"sort"
	"strings"
	"testing"

	"github.com/kijimaD/ruins/internal/raw"
	"github.com/stretchr/testify/require"
)

// TestRawTranslationCoverage は raw.toml の表示される name/description が英語原文の msgid で ja.po に
// 訳を持つことを検証する完全性ゲート。既定言語が ja のため、日本語のまま残す、または英語化しても
// ja.po へ訳を入れ忘れると日本語プレイヤーへ英語が漏れる。それを fail-closed で止める。
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

	// 表示される文字列は全て英語原文の msgid で ja.po に訳を持たねばならない。日本語のままの name は
	// msgid にならず ja.po に無いのでここで落ちる。英語化して訳を入れ忘れた場合も同様に落ちる。
	seen := map[string]bool{}
	var missing []string
	for _, s := range texts {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		if !defined[s] {
			missing = append(missing, s)
		}
	}
	sort.Strings(missing)

	require.Empty(t, missing,
		"表示される raw の name/description は英語原文で ja.po に訳を持つこと。未英語化か訳漏れ:\n%s",
		strings.Join(missing, "\n"))
}
