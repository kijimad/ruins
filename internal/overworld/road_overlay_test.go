package overworld

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRoadOverlay_表示範囲に道が現れ端点が集落に一致する(t *testing.T) {
	t.Parallel()
	const runSeed uint64 = 12345
	const cols consts.Chunk = 5
	// 集落は Spacing チャンクごとに1つ当たる。数リージョン跨ぐ表示範囲なら必ず道が出る。道は Y に沿って走る
	area := MacroRange{OriginX: 0, OriginY: 0, Cols: cols, Rows: 3 * placementOf(testutil.InitTestWorld(t).Resources.RawMaster, rawschema.PointFeatureKindSettlement).Spacing}
	overlay := buildRoadOverlay(testutil.InitTestWorld(t).Resources.RawMaster, runSeed, area, cols, 0)

	assert.NotEmpty(t, overlay, "複数リージョンを覆う表示範囲には道が出る")
	for c, dir := range overlay {
		assert.NotZero(t, dir, "道ありのキーはビットが立つ")
		assert.Zero(t, dir&^(RoadN|RoadS|RoadE|RoadW), "未定義ビットは立たない")
		assert.GreaterOrEqual(t, c.X, consts.Chunk(0), "列は帯の内側")
		assert.Less(t, c.X, cols, "列は帯の内側")
	}

	// 道は集落のチャンクに入らず、集落の脇の道ばたのチャンクを通る
	a := placementOf(testutil.InitTestWorld(t).Resources.RawMaster, rawschema.PointFeatureKindSettlement).WinnerOf(runSeed, 1, cols)
	roadside, ok := roadsideOf(a, cols)
	require.True(t, ok)
	assert.Zero(t, overlay[a], "集落のチャンクに道は入らない")
	assert.NotZero(t, overlay[roadside], "集落の道ばたを道が通る")
}
