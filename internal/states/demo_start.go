package states

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/action"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
)

// DemoStartState はデモ開始時のプレイヤー初期化を行うステート。
// OnStartでプレイヤー生成と職業適用を行い、最初のUpdateでTownStateに遷移する
type DemoStartState struct {
	es.BaseState[w.World]
}

func (st DemoStartState) String() string {
	return "DemoStart"
}

// OnStart はステート開始時にデフォルトプレイヤーを生成し、TownStateへの遷移を設定する
func (st *DemoStartState) OnStart(world w.World) error {
	player, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
	if err != nil {
		return fmt.Errorf("プレイヤーの生成に失敗: %w", err)
	}

	professions := raw.PtrSlice(world.Resources.RawMaster.Professions)
	if len(professions) > 0 {
		if err := action.ApplyProfession(world, player, professions[0]); err != nil {
			return fmt.Errorf("職業の適用に失敗: %w", err)
		}
	}

	st.SetTransition(es.Transition[w.World]{
		Type:          es.TransReplace,
		NewStateFuncs: []es.StateFactory[w.World]{NewTownState()},
	})

	return nil
}

// OnStop はステートが停止される際に呼ばれる
func (st *DemoStartState) OnStop(_ w.World) error { return nil }

// OnPause はステートが一時停止される際に呼ばれる
func (st *DemoStartState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *DemoStartState) OnResume(_ w.World) error { return nil }

// Update はOnStartで設定された遷移を消費してTownStateに遷移する
func (st *DemoStartState) Update(_ w.World) (es.Transition[w.World], error) {
	return st.ConsumeTransition(), nil
}

// Draw はスクリーンに描画する
func (st *DemoStartState) Draw(_ w.World, screen *ebiten.Image) error {
	screen.Fill(theme.ScreenBackground)
	return nil
}
