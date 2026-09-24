package interior

import (
	"testing"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/require"
)

// TestContentByID_コード必須のidが引ける は、データでなくコードが直接引く content id が raw.toml にあることを
// 固定する。generic は未知施設のフォールバック、flavor は flavor machine が引く。validate はデータ間の参照
// しか見ないので、これらのコード必須 id は誤削除に気づけるようテストで守る。欠ければ contentByID が panic する。
func TestContentByID_コード必須のidが引ける(t *testing.T) {
	t.Parallel()

	raws := testRaws()
	for _, id := range []string{"generic", "flavor"} {
		require.Equalf(t, id, contentByID(raws, id).ID, "コードが必須とする content %q が raw.toml に無い", id)
	}
}

// TestToContent_不正なダイスはerrorを返す は、amount のダイス表記が壊れたレシピを変換したとき toContent が
// 握りつぶさず error を返すことを固定する。golden は正常データしか通らないので、この失敗経路はここでしか
// 検証できない。生成時は contentByID がこの error を panic へ昇格させる。
func TestToContent_不正なダイスはerrorを返す(t *testing.T) {
	t.Parallel()

	ic := oapi.InteriorContent{
		Id: "bad",
		Groups: &[]oapi.ContentGroup{{
			Style: "pick_each",
			Items: []oapi.ContentStuff{{Kind: "furniture", Ref: "table", Amount: "notdice"}},
		}},
	}

	_, err := toContent(ic)
	require.Error(t, err)
	require.ErrorContains(t, err, "amount")
}
