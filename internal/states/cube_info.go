package states

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
	"github.com/kijimaD/ruins/internal/menuloop"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/menuframe"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

const cubeInfoMenuKey = "cube_info"

// CubeInfoState は移動拠点キューブの情報をタブで見る読み取り専用画面。
// いまは基本タブだけで、収納の総重量と燃料残量を出す。タブは後で増やす。
type CubeInfoState struct {
	es.BaseState[w.World]

	cube   ecs.Entity
	screen *menuloop.Screen[CubeInfoProps]
}

// CubeInfoProps は情報画面の表示 props
type CubeInfoProps struct {
	Tabs []statsTab
}

var _ es.State[w.World] = &CubeInfoState{}

// NewCubeInfoState はキューブの情報画面を作る
func NewCubeInfoState(cube ecs.Entity) (es.State[w.World], error) {
	return &CubeInfoState{cube: cube}, nil
}

// OnStart は Screen を組む。overlay は持たない読み取り専用画面
func (st *CubeInfoState) OnStart(_ w.World) error {
	st.screen = menuloop.NewScreen[CubeInfoProps](st)
	return nil
}

// Update はステートの更新を Screen へ委譲する
func (st *CubeInfoState) Update(world w.World) (es.Transition[w.World], error) {
	return st.screen.Update(world)
}

// Draw はステートの描画を Screen へ委譲する
func (st *CubeInfoState) Draw(_ w.World, screen *ebiten.Image) error {
	st.screen.Draw(screen)
	return nil
}

// DoAction は読み取り専用なので閉じる操作だけ扱う
func (st *CubeInfoState) DoAction(_ w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionMenuSelect:
		return es.Transition[w.World]{Type: es.TransNone}, nil
	default:
		return es.Transition[w.World]{}, fmt.Errorf("cubeInfo: unsupported action: %s", action)
	}
}

// Fetch は表示するタブを組む。いまは基本タブだけ
func (st *CubeInfoState) Fetch(world w.World) (CubeInfoProps, error) {
	return CubeInfoProps{Tabs: []statsTab{
		{Label: query.T(world, "Basic"), Items: cubeInfoItems(world, st.cube)},
	}}, nil
}

// Menu はタブ構成を返す。見出し行が無いのでスキップは不要
func (st *CubeInfoState) Menu(props CubeInfoProps) menuloop.MenuConfig {
	itemCounts := make([]int, len(props.Tabs))
	for i, tab := range props.Tabs {
		itemCounts[i] = len(tab.Items)
	}
	return menuloop.MenuConfig{Key: cubeInfoMenuKey, TabCount: len(props.Tabs), ItemCounts: itemCounts}
}

// ViewUI はタブ帯つきの情報表を組む
func (st *CubeInfoState) ViewUI(world w.World, props CubeInfoProps, cursor menuloop.Selection, res resources.UIResources) uicore.Drawable {
	labels := make([]string, len(props.Tabs))
	for i, tab := range props.Tabs {
		labels[i] = tab.Label
	}
	tabIndex := cursor.TabIndex
	if tabIndex >= len(props.Tabs) {
		tabIndex = 0
	}
	content, pager := buildStatsTableUI(world, props.Tabs[tabIndex].Items, cursor.ItemIndex, res)
	return menuframe.TabScreen(world, res, query.T(world, "Cube info"), labels, tabIndex, content, keybind.HelpHint(world), pager)
}

// cubeInfoItems はキューブの基本情報を表の行に組む。収納の総重量と燃料残量。
// 値は表示時に都度算出する。死んだキューブには空を返す
func cubeInfoItems(world w.World, cube ecs.Entity) []statusItemData {
	if !world.ECS.Alive(cube) {
		return nil
	}
	weight := query.CubeWeight(world, cube)
	fuel := query.CubeFuelTotal(world, cube)
	return []statusItemData{
		{Label: query.T(world, "Total weight"), Value: weight.KgString()},
		{Label: query.T(world, "Fuel"), Value: fuel.String()},
	}
}
