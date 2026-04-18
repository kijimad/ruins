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
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/resources"
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
func (st *CharacterJobState) OnStart(world w.World) error {
	st.menuMount = hooks.NewMount[jobMenuProps]()
	st.menuMount.SetProps(st.fetchProps(world))
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

	// Props更新。職業データは静的なのでOnStartでセット済み
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
	menuState, ok := hooks.GetState[hooks.TabMenuState](st.menuMount, "job")
	if !ok {
		return es.Transition[w.World]{}, fmt.Errorf("jobの取得に失敗")
	}
	itemIndex := menuState.ItemIndex

	if itemIndex >= len(props.Items) {
		return es.Transition[w.World]{Type: es.TransNone}, nil
	}

	prof := props.Items[itemIndex].Profession

	// 既存プレイヤーがいれば削除する
	if existing, err := worldhelper.GetPlayerEntity(world); err == nil {
		world.Manager.DeleteEntity(existing)
	}

	player, err := worldhelper.SpawnPlayer(world, 5, 5, "Ash")
	if err != nil {
		return es.Transition[w.World]{}, fmt.Errorf("プレイヤーの生成に失敗: %w", err)
	}
	if err := worldhelper.ApplyProfession(world, player, prof); err != nil {
		return es.Transition[w.World]{}, fmt.Errorf("職業の適用に失敗: %w", err)
	}

	// プレイヤー名を上書き
	name := world.Components.Name.Get(player).(*gc.Name)
	name.Name = st.playerName

	// 操作ガイドを表示する
	gamelog.New(gamelog.FieldLog).System("WASD: 移動する。").Log()
	gamelog.New(gamelog.FieldLog).System("Mキー: 拠点メニューを開く。").Log()
	gamelog.New(gamelog.FieldLog).System("Spaceキー: アクションメニューを開く。").Log()

	st.SetTransition(es.Transition[w.World]{
		Type:          es.TransReplace,
		NewStateFuncs: []es.StateFactory[w.World]{NewTownState()},
	})

	return st.ConsumeTransition(), nil
}

// ================
// buildUI
// ================

func (st *CharacterJobState) buildUI(world w.World) *ebitenui.UI {
	res := world.Resources.UIResources
	props := st.menuMount.GetProps()
	menuState, _ := hooks.GetState[hooks.TabMenuState](st.menuMount, "job")
	itemIndex := menuState.ItemIndex

	// 3行グリッド: タイトル(固定) / メインエリア(伸縮) / フッター(固定)
	rootContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(1),
			widget.GridLayoutOpts.Spacing(0, 10),
			widget.GridLayoutOpts.Stretch([]bool{true}, []bool{false, true, false}),
			widget.GridLayoutOpts.Padding(&widget.Insets{
				Top:    20,
				Bottom: 20,
				Left:   40,
				Right:  40,
			}),
		)),
	)

	// タイトル行
	titleContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	titleLabel := widget.NewText(
		widget.TextOpts.Text("職業", &res.Text.TitleFontFace, consts.PrimaryColor),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
			}),
		),
	)
	titleContainer.AddChild(titleLabel)

	// メインエリア: 左右分割
	leftContainer := styled.NewVerticalContainer()
	for i, item := range props.Items {
		isSelected := i == itemIndex
		itemWidget := styled.NewListItemText(item.Profession.Name, consts.TextColor, isSelected, res)
		leftContainer.AddChild(itemWidget)
	}
	rightContainer := st.buildDetailPanel(props, itemIndex, res)
	mainContainer := styled.NewWSplitContainer(leftContainer, rightContainer)

	// フッター: 説明 + ヒント
	footerContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(4),
		)),
	)
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
		),
	)
	hintLabel := widget.NewText(
		widget.TextOpts.Text(consts.IconArrowUp+consts.IconArrowDown+" 選択 / "+consts.IconKeyEnter+" 決定 / "+consts.IconKeyEsc+" 戻る", &res.Text.SmallFace, consts.SecondaryColor),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionCenter,
			}),
		),
	)
	footerContainer.AddChild(descriptionText)
	footerContainer.AddChild(hintLabel)

	rootContainer.AddChild(titleContainer)
	rootContainer.AddChild(mainContainer)
	rootContainer.AddChild(footerContainer)

	return &ebitenui.UI{Container: rootContainer}
}

// buildDetailPanel は選択中の職業の詳細パネルを構築する
func (st *CharacterJobState) buildDetailPanel(props jobMenuProps, itemIndex int, res *resources.UIResources) *widget.Container {
	container := styled.NewVerticalContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
	)

	if itemIndex >= len(props.Items) {
		return container
	}

	prof := props.Items[itemIndex].Profession

	// 装備
	if len(prof.Equips) > 0 {
		container.AddChild(styled.NewDescriptionText("装備", res))
		for _, equip := range prof.Equips {
			slotLabel := equip.Slot
			if slot, ok := gc.ParseEquipmentSlot(equip.Slot); ok {
				slotLabel = slot.String()
			}
			container.AddChild(styled.NewMenuText(fmt.Sprintf(" %s: %s", slotLabel, equip.Name), res))
		}
	}

	// 所持品
	if len(prof.Items) > 0 {
		container.AddChild(styled.NewDescriptionText("所持品", res))
		for _, item := range prof.Items {
			container.AddChild(styled.NewMenuText(fmt.Sprintf(" %s x%d", item.Name, item.Count), res))
		}
	}

	// スキル
	if len(prof.Skills) > 0 {
		container.AddChild(styled.NewDescriptionText("スキル", res))
		for _, skill := range prof.Skills {
			skillID := gc.SkillID(skill.ID)
			name := skill.ID
			if gc.HasSkillName(skillID) {
				name = gc.SkillName(skillID)
			}
			container.AddChild(styled.NewMenuText(fmt.Sprintf(" %s Lv.%d", name, skill.Value), res))
		}
	}

	return container
}
