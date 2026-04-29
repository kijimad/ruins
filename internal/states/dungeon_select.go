package states

import (
	"fmt"
	"image"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	w "github.com/kijimaD/ruins/internal/world"
)

// DungeonSelectState はダンジョン選択画面のステート
type DungeonSelectState struct {
	es.BaseState[w.World]
	mount  *hooks.Mount[dungeonSelectProps]
	widget *ebitenui.UI
}

func (st DungeonSelectState) String() string {
	return "DungeonSelect"
}

var _ es.State[w.World] = &DungeonSelectState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *DungeonSelectState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *DungeonSelectState) OnResume(_ w.World) error { return nil }

// OnStop はステートが終了する際に呼ばれる
func (st *DungeonSelectState) OnStop(_ w.World) error { return nil }

// OnStart はステートが開始される際に呼ばれる
func (st *DungeonSelectState) OnStart(_ w.World) error {
	st.mount = hooks.NewMount[dungeonSelectProps]()
	return nil
}

// Update はステートの更新処理
func (st *DungeonSelectState) Update(world w.World) (es.Transition[w.World], error) {
	if action, ok := HandleMenuInput(); ok {
		if transition, err := st.DoAction(world, action); err != nil {
			return es.Transition[w.World]{}, err
		} else if transition.Type != es.TransNone {
			return transition, nil
		}
		st.mount.Dispatch(action)
	}

	props := st.fetchProps(world)
	st.mount.SetProps(props)
	hooks.UseTabMenu(st.mount.Store(), "dselect", hooks.TabMenuConfig{
		TabCount:   1,
		ItemCounts: []int{len(props.Items)},
	})

	if st.mount.Update() || st.widget == nil {
		st.widget = st.buildUI(world)
	}

	st.widget.Update()
	return st.ConsumeTransition(), nil
}

// Draw はステートの描画処理
func (st *DungeonSelectState) Draw(world w.World, screen *ebiten.Image) error {
	// 右半分に選択中ダンジョンの背景画像を描画する
	menuState, ok := hooks.GetState[hooks.TabMenuState](st.mount, "dselect")
	if !ok {
		return fmt.Errorf("dselectステートの取得に失敗")
	}
	props := st.mount.GetProps()
	idx := menuState.ItemIndex

	if idx < len(props.Items) && !props.Items[idx].IsCancel {
		item := props.Items[idx]
		if item.ImageKey == "" {
			return fmt.Errorf("ダンジョンのImageKeyが未設定です: %s", item.Name)
		}
		bgSheet, sheetOK := (*world.Resources.SpriteSheets)["bg"]
		if !sheetOK {
			return fmt.Errorf("bgスプライトシートが存在しない")
		}
		sprite, ok := bgSheet.Sprites[item.ImageKey]
		if !ok {
			return fmt.Errorf("スプライトが見つかりません: %s", item.ImageKey)
		}
		rect := image.Rect(sprite.X, sprite.Y, sprite.X+sprite.Width, sprite.Y+sprite.Height)
		bgImage := bgSheet.Texture.Image.SubImage(rect).(*ebiten.Image)

		// 右パネル位置にパディング付きでスケーリングして描画する
		padding := 12.0
		op := &ebiten.DrawImageOptions{}
		panelX := float64(dungeonSelectLeftWidth) + padding
		panelW := float64(consts.MinGameWidth-dungeonSelectLeftWidth) - padding*2
		panelH := float64(dungeonSelectImageHeight) - padding
		scaleX := panelW / float64(sprite.Width)
		scaleY := panelH / float64(sprite.Height)
		op.GeoM.Scale(scaleX, scaleY)
		op.GeoM.Translate(panelX, padding)
		screen.DrawImage(bgImage, op)
	}

	st.widget.Draw(screen)
	return nil
}

// DoAction はActionを実行する
func (st *DungeonSelectState) DoAction(_ w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionMenuSelect:
		return st.handleSelection()
	case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown,
		inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight,
		inputmapper.ActionMenuTabNext, inputmapper.ActionMenuTabPrev:
		return es.Transition[w.World]{Type: es.TransNone}, nil
	default:
		return es.Transition[w.World]{}, fmt.Errorf("dungeonSelect: 未対応のアクション: %s", action)
	}
}

// ================
// Props
// ================

// レイアウト定数
const (
	dungeonSelectLeftWidth   = 160
	dungeonSelectImageHeight = 480
)

type dungeonSelectProps struct {
	Items []dungeonSelectItem
}

type dungeonSelectItem struct {
	Name        string
	Description string
	Cleared     bool
	ImageKey    string // bgスプライトシート内のキー
	IsCancel    bool   // 「やめる」項目
}

func (st *DungeonSelectState) fetchProps(world w.World) dungeonSelectProps {
	gp := world.Resources.GameProgress
	allDungeons := dungeon.GetAllDungeons()
	items := make([]dungeonSelectItem, 0, len(allDungeons)+1)

	for _, d := range allDungeons {
		items = append(items, dungeonSelectItem{
			Name:        d.Name,
			Description: d.Description,
			Cleared:     gp.IsDungeonCleared(d.Name),
			ImageKey:    d.ImageKey,
		})
	}

	items = append(items, dungeonSelectItem{
		Name:     "戻る",
		IsCancel: true,
	})

	return dungeonSelectProps{Items: items}
}

func (st *DungeonSelectState) handleSelection() (es.Transition[w.World], error) {
	menuState, ok := hooks.GetState[hooks.TabMenuState](st.mount, "dselect")
	if !ok {
		return es.Transition[w.World]{}, fmt.Errorf("dselectの取得に失敗")
	}
	item := st.mount.GetProps().Items[menuState.ItemIndex]
	if item.IsCancel {
		return es.Transition[w.World]{Type: es.TransPop}, nil
	}

	// ダンジョンへ遷移する
	return es.Transition[w.World]{
		Type: es.TransPush,
		NewStateFuncs: []es.StateFactory[w.World]{
			NewFadeoutAnimationState(NewDungeonState(1, WithDefinitionName(item.Name))),
		},
	}, nil
}

// ================
// buildUI
// ================

func (st *DungeonSelectState) buildUI(world w.World) *ebitenui.UI {
	res := world.Resources.UIResources
	props := st.mount.GetProps()
	menuState, _ := hooks.GetState[hooks.TabMenuState](st.mount, "dselect")
	itemIndex := menuState.ItemIndex

	// ルートコンテナ: 横2列（左リスト | 右詳細）
	// 背景画像はDrawで直接描画するため、ルートは透明にする
	root := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(2),
			widget.GridLayoutOpts.Spacing(0, 0),
			widget.GridLayoutOpts.Stretch([]bool{false, true}, []bool{true}),
			widget.GridLayoutOpts.Padding(&widget.Insets{
				Top: 4, Bottom: 4, Left: 4, Right: 4,
			}),
		)),
	)

	root.AddChild(st.buildListPanel(props, itemIndex, res))
	root.AddChild(st.buildDetailPanel(props, itemIndex, res))

	return &ebitenui.UI{Container: root}
}

func (st *DungeonSelectState) buildListPanel(props dungeonSelectProps, itemIndex int, res *resources.UIResources) *widget.Container {
	container := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(4),
			widget.RowLayoutOpts.Padding(&widget.Insets{
				Top: 20, Bottom: 10, Left: 4, Right: 4,
			}),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(dungeonSelectLeftWidth, 0),
		),
	)

	for i, item := range props.Items {
		isSelected := i == itemIndex
		label := item.Name
		if item.Cleared {
			label = consts.IconStar + " " + label
		}
		color := consts.TextColor
		container.AddChild(styled.NewListItemText(label, color, isSelected, res))
	}

	return container
}

func (st *DungeonSelectState) buildDetailPanel(props dungeonSelectProps, itemIndex int, res *resources.UIResources) *widget.Container {
	container := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(4),
			widget.RowLayoutOpts.Padding(&widget.Insets{
				Top: 4, Bottom: 10, Left: 8, Right: 8,
			}),
		)),
	)

	// 画像エリアのスペーサー（Drawで直接描画するので空のコンテナで高さを確保する）
	spacer := widget.NewContainer(
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(0, dungeonSelectImageHeight),
		),
		widget.ContainerOpts.Layout(widget.NewRowLayout()),
	)
	container.AddChild(spacer)

	// 説明文
	if itemIndex < len(props.Items) && !props.Items[itemIndex].IsCancel {
		item := props.Items[itemIndex]
		container.AddChild(styled.NewMenuText(item.Description, res))
	}

	return container
}
