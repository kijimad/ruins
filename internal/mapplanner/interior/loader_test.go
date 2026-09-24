package interior

import (
	"testing"

	"github.com/kijimaD/ruins/internal/raw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadContents_GoレシピとrawTomlが一致する は、raw.toml へ移植した内装レシピが Go の content_catalog の
// 関数とバイト等価であることを固定する。移植の取りこぼしをここで捕まえ、FurnishBuilding を raw 駆動へ
// 切り替えたとき golden が動かないことを保証する。レシピを移すたびに want へ1行足す。
func TestLoadContents_GoレシピとrawTomlが一致する(t *testing.T) {
	t.Parallel()

	master, err := raw.LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)
	cs, err := LoadContents(master)
	require.NoError(t, err)

	want := map[string]Content{
		"clinic": clinicContent(),
	}
	for id, goContent := range want {
		got, ok := cs.byID[id]
		require.Truef(t, ok, "レシピ %q が raw.toml に移植済み", id)
		assert.Equalf(t, goContent, got, "レシピ %q が Go と raw.toml で一致する", id)
	}
}
