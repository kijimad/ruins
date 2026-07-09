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
	"github.com/kijimaD/ruins/internal/save"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
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
	// ウィジェットを先に描画する
	st.widget.Draw(screen)

	// 画像をパネルの上に描画して、半透明パネルで暗くならないようにする
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
		bgSheet, sheetOK := world.Resources.SpriteSheets["bg"]
		if !sheetOK {
			return fmt.Errorf("bgスプライトシートが存在しない")
		}
		sprite, ok := bgSheet.Sprites[item.ImageKey]
		if !ok {
			return fmt.Errorf("スプライトが見つかりません: %s", item.ImageKey)
		}
		rect := image.Rect(sprite.X, sprite.Y, sprite.X+sprite.Width, sprite.Y+sprite.Height)
		bgImage := bgSheet.Texture.Image.SubImage(rect).(*ebiten.Image)

		scaleX := float64(dungeonSelectImageWidth) / float64(sprite.Width)
		scaleY := float64(dungeonSelectImageHeight) / float64(sprite.Height)

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(scaleX, scaleY)
		op.GeoM.Translate(dungeonSelectImageX, dungeonSelectImageY)
		screen.DrawImage(bgImage, op)
	}

	return nil
}

// DoAction はActionを実行する
func (st *DungeonSelectState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionMenuSelect:
		saveManager, err := save.NewSerializationManager()
		if err != nil {
			return es.Transition[w.World]{}, err
		}
		if err := saveManager.AutoSave(world); err != nil {
			return es.Transition[w.World]{}, err
		}
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
	dungeonSelectListWidth   = 160
	dungeonSelectImageWidth  = 320
	dungeonSelectImageHeight = 240
	dungeonSelectWindowPad   = 20
)

// ウィンドウ中央配置から算出した画像描画位置。
// ウィンドウ幅 = pad + listWidth + pad(spacing) + imageWidth + pad
const (
	dungeonSelectWindowWidth = dungeonSelectWindowPad*3 + dungeonSelectListWidth + dungeonSelectImageWidth
	// ウィンドウ左端X = (screenW - windowW) / 2
	// 画像左端X = ウィンドウ左端 + pad + listWidth + pad(spacing)
	dungeonSelectImageX = float64(consts.GameWidth-dungeonSelectWindowWidth)/2 + dungeonSelectWindowPad*2 + dungeonSelectListWidth
	dungeonSelectImageY = float64(consts.GameHeight-dungeonSelectImageHeight) / 2
)

type dungeonSelectProps struct {
	Items []dungeonSelectItem
}

type dungeonSelectItem struct {
	Name     string
	Cleared  bool
	ImageKey string // bgスプライトシート内のキー
	IsCancel bool   // 「やめる」項目
}

func (st *DungeonSelectState) fetchProps(world w.World) dungeonSelectProps {
	gp := query.GetGameProgress(world)
	allDungeons := dungeon.GetAllDungeons()
	items := make([]dungeonSelectItem, 0, len(allDungeons)+1)

	for i := range allDungeons {
		d := &allDungeons[i]
		items = append(items, dungeonSelectItem{
			Name:     d.Name,
			Cleared:  gp.IsDungeonCleared(d.Name),
			ImageKey: d.ImageKey,
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

	root := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	// ウィンドウコンテナ: 左に選択肢、右に画像スペーサーを横並びにする
	windowContainer := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(2),
			widget.GridLayoutOpts.Spacing(dungeonSelectWindowPad, 0),
			widget.GridLayoutOpts.Stretch([]bool{false, true}, []bool{true}),
			widget.GridLayoutOpts.Padding(&widget.Insets{
				Top: dungeonSelectWindowPad, Bottom: dungeonSelectWindowPad,
				Left: dungeonSelectWindowPad, Right: dungeonSelectWindowPad,
			}),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}),
			widget.WidgetOpts.MinSize(dungeonSelectWindowWidth, 0),
		),
	)

	// 左: 選択肢リスト
	windowContainer.AddChild(st.buildListItems(props, itemIndex, res))

	// 右: 画像エリアのスペーサー（Drawで直接描画する）
	spacer := widget.NewContainer(
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(dungeonSelectImageWidth, dungeonSelectImageHeight),
		),
		widget.ContainerOpts.Layout(widget.NewRowLayout()),
	)
	windowContainer.AddChild(spacer)

	root.AddChild(windowContainer)
	return &ebitenui.UI{Container: root}
}

func (st *DungeonSelectState) buildListItems(props dungeonSelectProps, itemIndex int, res resources.UIResources) *widget.Container {
	container := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(theme.Space2),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(dungeonSelectListWidth, 0),
		),
	)

	for i, item := range props.Items {
		isSelected := i == itemIndex
		label := item.Name
		if item.Cleared {
			label = consts.IconStar + " " + label
		}
		color := theme.TextPrimary
		container.AddChild(styled.NewListItemText(label, color, isSelected, res))
	}

	return container
}
