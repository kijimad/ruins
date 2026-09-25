package interior

import (
	"testing"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/require"
)

// TestFlavorContent_専用フィールドから引ける は、全施設共通のフレーバー装飾が専用フィールド flavorContent から
// 引けることを固定する。施設レシピと直交する大域レイヤなので、id プールでなく専用フィールドで持つ。
func TestFlavorContent_専用フィールドから引ける(t *testing.T) {
	t.Parallel()

	c, err := flavorContent(testRaws())
	require.NoError(t, err)
	require.Equal(t, "flavor", c.ID)
}

// TestFlavorContent_未設定はerror は、flavorContent が無い raw を引くと生成を落とさず error を返すことを固定する。
func TestFlavorContent_未設定はerror(t *testing.T) {
	t.Parallel()

	_, err := flavorContent(oapi.Raws{})
	require.ErrorIs(t, err, errFlavorContentMissing)
}

// TestContentByID_未定義idはerror は、存在しない content id を引くと生成を落とさず error を返すことを固定する。
func TestContentByID_未定義idはerror(t *testing.T) {
	t.Parallel()

	_, err := contentByID(testRaws(), "no_such_content")
	require.ErrorIs(t, err, errContentNotFound)
}

// TestFacilityContent_未登録施設はerror は、facilityContents に無い施設種別を引くと生成を落とさず error を
// 返すことを固定する。
func TestFacilityContent_未登録施設はerror(t *testing.T) {
	t.Parallel()

	_, err := facilityContent(oapi.Raws{}, FacilityKind("no_such_facility"), 0)
	require.ErrorIs(t, err, errFacilityNotRegistered)
}

// TestBackRoomContent_未登録施設はerror は、facilityRooms に無い施設種別の奥室既定を引くと error を返すことを
// 固定する。
func TestBackRoomContent_未登録施設はerror(t *testing.T) {
	t.Parallel()

	_, err := backRoomContent(oapi.Raws{}, FacilityKind("no_such_facility"))
	require.ErrorIs(t, err, errFacilityNotRegistered)
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
