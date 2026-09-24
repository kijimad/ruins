package interior

import (
	"testing"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/require"
)

// TestActiveContents_コード必須のidが存在する は、データでなくコードが直接引く content id が raw.toml に
// あることを固定する。generic は未知施設のフォールバック、flavor は flavor machine が引く。validate は
// データ間の参照しか見ないので、これらのコード必須 id は誤削除に気づけるようテストで守る。
func TestActiveContents_コード必須のidが存在する(t *testing.T) {
	t.Parallel()

	cs := activeContents()
	for _, id := range []string{"generic", "flavor"} {
		require.Containsf(t, cs.byID, id, "コードが必須とする content %q が raw.toml に無い", id)
	}
}

// TestLoadContents_不正なダイスはerrorを返す は、amount のダイス表記が壊れたレシピをロードしたとき
// LoadContents が握りつぶさず error を返すことを固定する。golden は正常データしか通らないので、この
// 失敗経路はここでしか検証できない。
func TestLoadContents_不正なダイスはerrorを返す(t *testing.T) {
	t.Parallel()

	raws := oapi.Raws{
		InteriorContents: &[]oapi.InteriorContent{{
			Id: "bad",
			Groups: &[]oapi.ContentGroup{{
				Style: "pick_each",
				Items: []oapi.ContentStuff{{Kind: "furniture", Ref: "table", Amount: "notdice"}},
			}},
		}},
	}

	_, err := LoadContents(raws)
	require.Error(t, err)
	require.ErrorContains(t, err, "amount")
}
