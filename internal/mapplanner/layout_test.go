package mapplanner

import (
	"testing"

	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/require"
)

// TestNewStart_全ての最初の段の行から段を作れる は、raw に足した最初の段の行が Go の分岐に届くことを固定する。
// 絵の最初の段は newLayoutChain が別に作るので除く
func TestNewStart_全ての最初の段の行から段を作れる(t *testing.T) {
	t.Parallel()
	raws := testutil.InitTestWorld(t).Resources.RawMaster
	var ids []string
	ids = appendIDs(ids, raws.RectRoomStarts, func(r *rawschema.RectRoomStart) string { return r.Id })
	ids = appendIDs(ids, raws.NoiseStarts, func(r *rawschema.NoiseStart) string { return r.Id })
	ids = appendIDs(ids, raws.BuildingStarts, func(r *rawschema.BuildingStart) string { return r.Id })
	ids = appendIDs(ids, raws.ClearingStarts, func(r *rawschema.ClearingStart) string { return r.Id })
	ids = appendIDs(ids, raws.SimpleStarts, func(r *rawschema.SimpleStart) string { return r.Id })
	require.NotEmpty(t, ids)
	for _, id := range ids {
		_, err := newStart(raws, id, testLayoutTiles)
		require.NoError(t, err, id)
	}
}

// TestNewStep_全ての中の段の行から段を作れる は、raw に足した中の段の行が Go の分岐に届くことを固定する
func TestNewStep_全ての中の段の行から段を作れる(t *testing.T) {
	t.Parallel()
	raws := testutil.InitTestWorld(t).Resources.RawMaster
	var ids []string
	ids = appendIDs(ids, raws.CorridorSteps, func(r *rawschema.CorridorStep) string { return r.Id })
	ids = appendIDs(ids, raws.DoorSteps, func(r *rawschema.DoorStep) string { return r.Id })
	ids = appendIDs(ids, raws.BigRoomSteps, func(r *rawschema.BigRoomStep) string { return r.Id })
	ids = appendIDs(ids, raws.AutomataSteps, func(r *rawschema.AutomataStep) string { return r.Id })
	ids = appendIDs(ids, raws.WidenSteps, func(r *rawschema.WidenStep) string { return r.Id })
	ids = appendIDs(ids, raws.ScatterWallSteps, func(r *rawschema.ScatterWallStep) string { return r.Id })
	ids = appendIDs(ids, raws.RuinedWallSteps, func(r *rawschema.RuinedWallStep) string { return r.Id })
	ids = appendIDs(ids, raws.RuinedPathSteps, func(r *rawschema.RuinedPathStep) string { return r.Id })
	ids = appendIDs(ids, raws.TreeSteps, func(r *rawschema.TreeStep) string { return r.Id })
	ids = appendIDs(ids, raws.TrailSteps, func(r *rawschema.TrailStep) string { return r.Id })
	ids = appendIDs(ids, raws.GladeSteps, func(r *rawschema.GladeStep) string { return r.Id })
	ids = appendIDs(ids, raws.SimpleSteps, func(r *rawschema.SimpleStep) string { return r.Id })
	require.NotEmpty(t, ids)
	for _, id := range ids {
		_, err := newStep(raws, id, testLayoutTiles)
		require.NoError(t, err, id)
	}
}

// TestNewSimpleStep_全ての種類から段を作れる は、列挙に足した種類が switch に届くことを固定する
func TestNewSimpleStep_全ての種類から段を作れる(t *testing.T) {
	t.Parallel()
	for _, k := range rawschema.SimpleStepKindValues() {
		_, err := newSimpleStep(rawschema.SimpleStep{Id: "x", Kind: k}, testLayoutTiles)
		require.NoError(t, err, string(k))
	}
	for _, k := range rawschema.SimpleStartKindValues() {
		raws := rawschema.Raws{SimpleStarts: &[]rawschema.SimpleStart{{Id: "x", Kind: k}}}
		_, err := newStart(raws, "x", testLayoutTiles)
		require.NoError(t, err, string(k))
	}
}

func appendIDs[T any](ids []string, rows *[]T, id func(*T) string) []string {
	rs := raw.PtrSlice(rows)
	for i := range rs {
		ids = append(ids, id(&rs[i]))
	}
	return ids
}
