package states

import (
	"fmt"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/config"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
)

// CharacterJobState はキャラクター職業選択画面のステート
type CharacterJobState struct {
	es.BaseState[w.World]
	menuMount  *hooks.Mount[jobMenuProps]
	widget     *ebitenui.UI
	playerName string // TODO: どうにかする。キャラメイクは複数のstateで構成され、前の決定事項を保持する必要がある...
}

// NewCharacterJobState は職業選択ステートのファクトリを返す
func NewCharacterJobState(playerName string) es.StateFactory[w.World] {
	return func() es.State[w.World] {
		return &CharacterJobState{
			playerName: playerName,
		}
	}
}

func (st CharacterJobState) String() string {
	return "CharacterJob"
}

// State interface ================

var _ es.State[w.World] = &CharacterJobState{}
var _ es.ActionHandler[w.World] = &CharacterJobState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *CharacterJobState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *CharacterJobState) OnResume(_ w.World) error { return nil }

// OnStart はステート開始時の処理を行う
func (st *CharacterJobState) OnStart(_ w.World) error {
	st.menuMount = hooks.NewMount[jobMenuProps]()
	return nil
}

// OnStop はステートが停止される際に呼ばれる
func (st *CharacterJobState) OnStop(_ w.World) error { return nil }

// Update はゲームステートの更新処理を行う
func (st *CharacterJobState) Update(world w.World) (es.Transition[w.World], error) {
	// 入力処理
	if action, ok := st.HandleInput(world.Config); ok {
		if transition, err := st.DoAction(world, action); err != nil {
			return es.Transition[w.World]{}, err
		} else if transition.Type != es.TransNone {
			return transition, nil
		}
		st.menuMount.Dispatch(action)
	}

	// Props更新
	st.menuMount.SetProps(st.fetchProps(world))
	props := st.menuMount.GetProps()
	hooks.UseTabMenu(st.menuMount.Store(), "job", hooks.TabMenuConfig{
		TabCount:   1,
		ItemCounts: []int{len(props.Items)},
	})

	// dirty判定とUI再構築
	if st.menuMount.Update() || st.widget == nil {
		st.widget = st.buildUI(world)
	}

	st.widget.Update()
	return st.ConsumeTransition(), nil
}

// Draw はスクリーンに描画する
func (st *CharacterJobState) Draw(_ w.World, screen *ebiten.Image) error {
	screen.Fill(consts.BlackColor)
	st.widget.Draw(screen)
	return nil
}

// HandleInput はキー入力をActionに変換する
func (st *CharacterJobState) HandleInput(_ *config.Config) (inputmapper.ActionID, bool) {
	return HandleMenuInput()
}

// DoAction はActionを実行する
func (st *CharacterJobState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionMenuSelect:
		return st.handleSelection(world)
	case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown, inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight, inputmapper.ActionMenuTabNext, inputmapper.ActionMenuTabPrev:
		// Dispatchで処理される
	default:
		return es.Transition[w.World]{}, fmt.Errorf("characterJob: 未対応のアクション: %s", action)
	}
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

// ================
// Props
// ================

// jobMenuProps は職業選択メニューのProps
type jobMenuProps struct {
	Items []jobMenuItem
}

// jobMenuItem は職業メニューの項目
type jobMenuItem struct {
	Profession raw.Profession
}

func (st *CharacterJobState) fetchProps(world w.World) jobMenuProps {
	professions := world.Resources.RawMaster.Raws.Professions
	items := make([]jobMenuItem, len(professions))
	for i, p := range professions {
		items[i] = jobMenuItem{Profession: p}
	}
	return jobMenuProps{Items: items}
}

func (st *CharacterJobState) handleSelection(world w.World) (es.Transition[w.World], error) {
	props := st.menuMount.GetProps()
	itemIndex, ok := hooks.GetState[int](st.menuMount, "job_itemIndex")
	if !ok {
		return es.Transition[w.World]{}, fmt.Errorf("job_itemIndexの取得に失敗")
	}

	if itemIndex >= len(props.Items) {
		return es.Transition[w.World]{Type: es.TransNone}, nil
	}

	prof := props.Items[itemIndex].Profession
	st.selectProfession(world, prof)
	return st.ConsumeTransition(), nil
}

// selectProfession は職業を選択してゲームを開始する
func (st *CharacterJobState) selectProfession(world w.World, prof raw.Profession) {
	// TOMLの"Ash"定義でエンティティを生成し、名前と属性を上書きする
	player, _ := worldhelper.SpawnPlayer(world, 5, 5, "Ash")

	// プレイヤー名を上書き
	name := world.Components.Name.Get(player).(*gc.Name)
	name.Name = st.playerName

	// 職業の属性値で上書き
	attrs := world.Components.Attributes.Get(player).(*gc.Attributes)
	attrs.Strength = gc.Attribute{Base: prof.Attributes.Strength}
	attrs.Sensation = gc.Attribute{Base: prof.Attributes.Sensation}
	attrs.Dexterity = gc.Attribute{Base: prof.Attributes.Dexterity}
	attrs.Agility = gc.Attribute{Base: prof.Attributes.Agility}
	attrs.Vitality = gc.Attribute{Base: prof.Attributes.Vitality}
	attrs.Defense = gc.Attribute{Base: prof.Attributes.Defense}

	// 職業のスキル初期値を設定
	if len(prof.Skills) > 0 {
		skills := world.Components.Skills.Get(player).(*gc.Skills)
		for _, ps := range prof.Skills {
			if s, ok := skills.Data[gc.SkillID(ps.ID)]; ok {
				s.Value = ps.Value
			}
		}
		skills.RecalculateEffects()
	}

	// 属性値変更後にHP/SP/EP/APを再計算
	_ = worldhelper.FullRecover(world, player)

	// 初期装備を付与
	for _, item := range prof.Items {
		_, _ = worldhelper.SpawnItem(world, item.Name, item.Count, gc.ItemLocationInPlayerBackpack)
	}

	st.SetTransition(es.Transition[w.World]{
		Type:          es.TransReplace,
		NewStateFuncs: []es.StateFactory[w.World]{NewTownState()},
	})
}

// ================
// buildUI
// ================

func (st *CharacterJobState) buildUI(world w.World) *ebitenui.UI {
	res := world.Resources.UIResources
	props := st.menuMount.GetProps()
	itemIndex, _ := hooks.GetState[int](st.menuMount, "job_itemIndex")

	rootContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	centerContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(20),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}),
		),
	)

	titleLabel := widget.NewText(
		widget.TextOpts.Text("職業", &res.Text.TitleFontFace, consts.PrimaryColor),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionCenter,
			}),
		),
	)

	// メニューコンテナを構築
	menuContainer := styled.NewVerticalContainer()
	for i, item := range props.Items {
		isSelected := i == itemIndex
		itemWidget := styled.NewListItemText(item.Profession.Name, consts.TextColor, isSelected, res)
		menuContainer.AddChild(itemWidget)
	}

	// 職業説明テキスト
	description := ""
	if itemIndex < len(props.Items) {
		description = props.Items[itemIndex].Profession.Description
	}
	descriptionText := widget.NewText(
		widget.TextOpts.Text(description, &res.Text.SmallFace, consts.SecondaryColor),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionCenter,
			}),
			widget.WidgetOpts.MinSize(300, 0),
		),
	)

	// 操作ヒント
	hintLabel := widget.NewText(
		widget.TextOpts.Text(consts.IconArrowUp+consts.IconArrowDown+" 選択 / "+consts.IconKeyEnter+" 決定 / "+consts.IconKeyEsc+" 戻る", &res.Text.SmallFace, consts.SecondaryColor),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionCenter,
			}),
		),
	)

	centerContainer.AddChild(titleLabel)
	centerContainer.AddChild(menuContainer)
	centerContainer.AddChild(descriptionText)
	centerContainer.AddChild(hintLabel)

	rootContainer.AddChild(centerContainer)

	return &ebitenui.UI{Container: rootContainer}
}
