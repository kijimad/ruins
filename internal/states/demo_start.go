package states

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/gameaction"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
)

// DemoStartState はデモ開始時のプレイヤー初期化を行うステート。
// OnStartでプレイヤー生成と職業適用を行い、最初のUpdateで街を含むオーバーワールドへ遷移する
type DemoStartState struct {
	es.BaseState[w.World]
}

// OnStart はステート開始時にデフォルトプレイヤーを生成し、オーバーワールドへの遷移を設定する
func (st *DemoStartState) OnStart(world w.World) error {
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	if err != nil {
		return fmt.Errorf("failed to spawn player: %w", err)
	}

	professions := raw.PtrSlice(world.Resources.RawMaster.Professions)
	if len(professions) > 0 {
		if err := gameaction.ApplyProfession(world, player, professions[0]); err != nil {
			return fmt.Errorf("failed to apply profession: %w", err)
		}
	}

	if err := spawnDebugBiscuits(world); err != nil {
		return err
	}

	st.SetTransition(es.Transition[w.World]{
		Type:          es.TransReplace,
		NewStateFuncs: []es.StateFactory[w.World]{newGameOverworldState(world)},
	})

	return nil
}

// spawnDebugBiscuits はデバッグ用に鮮度の違うビスケットを初期所持へ積む。
// 鮮度でスタックが分かれることを手元で確認するための仕込みで、新鮮・劣化・腐敗を各2個そろえる。
// 開始直後は実効劣化が RotAccrued に等しいので、StageLength の倍数で置けば段階が決まり raw の変更にも追随する。
func spawnDebugBiscuits(world w.World) error {
	// 新鮮 [0, SL) 劣化 [SL, 2SL) 腐敗 [2SL, ) の各段階を作る RotAccrued の倍率
	for _, mul := range []float64{0, 1, 2} {
		for range 2 {
			entity, err := lifecycle.SpawnBackpackItem(world, "biscuit", 1)
			if err != nil {
				return fmt.Errorf("failed to spawn debug biscuit: %w", err)
			}
			perishable := world.Components.Perishable.Get(entity)
			perishable.RotAccrued = consts.Turn(float64(perishable.StageLength) * mul)
		}
	}
	return nil
}

// OnStop はステートが停止される際に呼ばれる
func (st *DemoStartState) OnStop(_ w.World) error { return nil }

// OnPause はステートが一時停止される際に呼ばれる
func (st *DemoStartState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *DemoStartState) OnResume(_ w.World) error { return nil }

// Update はOnStartで設定された遷移を消費してオーバーワールドに遷移する
func (st *DemoStartState) Update(_ w.World) (es.Transition[w.World], error) {
	return st.ConsumeTransition(), nil
}

// Draw はスクリーンに描画する
func (st *DemoStartState) Draw(_ w.World, screen *ebiten.Image) error {
	screen.Fill(theme.ScreenBackground)
	return nil
}
