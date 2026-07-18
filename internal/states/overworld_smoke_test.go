package states_test

import (
	"testing"

	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/mapplanner"
	gs "github.com/kijimaD/ruins/internal/states"
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/stretchr/testify/require"
)

// TestOverworldState_Updateスモーク は OverworldState を実ワールドで複数フレーム回し、
// システム列（ターン/視界/カメラ/HUD 等）＋ maybeShift が panic・エラーなく回ることを確認する。
// 入力は無いのでシフトは起きないが、初期帯生成後の Update 経路の健全性を守る。
func TestOverworldState_Updateスモーク(t *testing.T) {
	t.Parallel()

	world := vrt.InitVRTWorld(t)

	factory := gs.NewOverworldState(mapplanner.PlannerTypeOverworldField, &gs.NewGameParams{RunSeed: 1, ChunkW: 50, ChunkH: 50, K: 3})
	state, err := factory()
	require.NoError(t, err)

	sm, err := es.Init(state, world)
	require.NoError(t, err)

	for range 10 {
		require.NoError(t, sm.Update(world))
	}
}
