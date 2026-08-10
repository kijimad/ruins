package states

import (
	"fmt"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/menurt"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/gameaction"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
)

// CharacterJobState はキャラクター職業選択画面のステート
type CharacterJobState struct {
	es.BaseState[w.World]
	screen     *menurt.Screen[JobMenuProps]
	playerName string // TODO: どうにかする。キャラメイクは複数のstateで構成され、前の決定事項を保持する必要がある...
}

// NewCharacterJobState は職業選択ステートのファクトリを返す
func NewCharacterJobState(playerName string) es.StateFactory[w.World] {
	return func() (es.State[w.World], error) {
		return &CharacterJobState{
			playerName: playerName,
		}, nil
	}
}

// State interface ================

var _ es.State[w.World] = &CharacterJobState{}

// OnStart はステート開始時の処理を行う
func (st *CharacterJobState) OnStart(_ w.World) error {
	st.screen = menurt.NewScreen[JobMenuProps](st)
	return nil
}

// Update はゲームステートの更新処理を行う
func (st *CharacterJobState) Update(world w.World) (es.Transition[w.World], error) {
	return st.screen.Update(world)
}

// Draw はスクリーンに描画する
func (st *CharacterJobState) Draw(_ w.World, screen *ebiten.Image) error {
	screen.Fill(theme.ScreenBackground)
	st.screen.Draw(screen)
	return nil
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
		return es.Transition[w.World]{}, fmt.Errorf("characterJob: unsupported action: %s", action)
	}
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

// ================
// Props
// ================

// JobMenuProps は職業選択メニューのProps
type JobMenuProps struct {
	Items []jobMenuItem
}

// jobMenuItem は職業メニューの項目
type jobMenuItem struct {
	Profession oapi.Profession
}

// Fetch は世界から表示 props を構築する。menurt.Model の Model 部にあたる
func (st *CharacterJobState) Fetch(world w.World) JobMenuProps {
	professions := raw.PtrSlice(world.Resources.RawMaster.Professions)
	items := make([]jobMenuItem, len(professions))
	for i := range professions {
		items[i] = jobMenuItem{Profession: professions[i]}
	}
	return JobMenuProps{Items: items}
}

// Menu は一覧の構成を返す。menurt.Model の Menu 部にあたる
func (st *CharacterJobState) Menu(props JobMenuProps) menurt.MenuConfig {
	return menurt.MenuConfig{Key: "job", TabCount: 1, ItemCounts: []int{len(props.Items)}}
}

func (st *CharacterJobState) handleSelection(world w.World) (es.Transition[w.World], error) {
	props := st.screen.Props()
	itemIndex := st.screen.Selection().ItemIndex
	if itemIndex >= len(props.Items) {
		return es.Transition[w.World]{Type: es.TransNone}, nil
	}

	prof := props.Items[itemIndex].Profession

	// 既存プレイヤーがいれば削除する
	if existing, err := query.GetPlayerEntity(world); err == nil {
		world.ECS.RemoveEntity(existing)
	}

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	if err != nil {
		return es.Transition[w.World]{}, fmt.Errorf("failed to spawn player: %w", err)
	}
	if err := gameaction.ApplyProfession(world, player, prof); err != nil {
		return es.Transition[w.World]{}, fmt.Errorf("failed to apply profession: %w", err)
	}

	if _, err := lifecycle.SpawnDefaultSquadMember(world, player); err != nil {
		return es.Transition[w.World]{}, fmt.Errorf("failed to spawn initial member: %w", err)
	}

	// プレイヤー名を上書き
	name := world.Components.Name.Get(player)
	name.Name = st.playerName

	// 操作ガイドを表示する
	gamelog.New(query.GetGameLog(world)).Markup(gamelog.Tag("system", query.T(world, "Arrows: Move."))).Log()
	gamelog.New(query.GetGameLog(world)).Markup(gamelog.Tag("system", query.T(world, "M key: Open base menu."))).Log()
	gamelog.New(query.GetGameLog(world)).Markup(gamelog.Tag("system", query.T(world, "Space key: Open action menu."))).Log()

	st.SetTransition(es.Transition[w.World]{
		Type:          es.TransReplace,
		NewStateFuncs: []es.StateFactory[w.World]{newGameOverworldState(world)},
	})

	return st.ConsumeTransition(), nil
}

// ================
// View
// ================

// View は props を UI へ組む純粋な描画。menurt.Model の View 部にあたる
func (st *CharacterJobState) View(world w.World, props JobMenuProps, cursor menurt.Selection, res resources.UIResources) *ebitenui.UI {
	itemIndex := cursor.ItemIndex

	// 3行グリッド: タイトル(固定) / メインエリア(伸縮) / フッター(固定)
	rootContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(1),
			widget.GridLayoutOpts.Spacing(0, theme.Space4),
			widget.GridLayoutOpts.Stretch([]bool{true}, []bool{false, true, false}),
			widget.GridLayoutOpts.Padding(&widget.Insets{
				Top:    theme.Space6,
				Bottom: theme.Space6,
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
		widget.TextOpts.Text(query.T(world, "Profession"), &res.Text.TitleFontFace, theme.TextPrimary),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
			}),
		),
	)
	titleContainer.AddChild(titleLabel)

	// メインエリア: 左右分割。左の一覧は他メニューと同じ密なテーブル行で縦幅と行間を揃える
	leftContainer := styled.NewVerticalContainer()
	rows := make([]menuRow, len(props.Items))
	for i := range props.Items {
		rows[i] = menuRow{Cells: styled.TextCells(query.T(world, props.Items[i].Profession.Name))}
	}
	leftContainer.AddChild(renderMenuList(itemIndex, rows, []int{160}, []styled.TextAlign{styled.AlignLeft}, menuListOpts{Spaced: true}, res))
	rightContainer := st.buildDetailPanel(world, props, itemIndex, res)
	mainContainer := styled.NewWSplitContainer(leftContainer, rightContainer)

	// フッター: 説明 + ヒント
	footerContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(theme.Space2),
		)),
	)
	description := ""
	if itemIndex < len(props.Items) {
		description = query.T(world, props.Items[itemIndex].Profession.Description)
	}
	descriptionText := widget.NewText(
		widget.TextOpts.Text(description, &res.Text.SmallFace, theme.TextAccent),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionCenter,
			}),
		),
	)
	hintLabel := widget.NewText(
		widget.TextOpts.Text(consts.IconArrowUp+consts.IconArrowDown+" "+query.T(world, "Select")+" / "+consts.IconKeyEnter+" "+query.T(world, "Confirm")+" / "+consts.IconKeyEsc+" "+query.T(world, "Back"), &res.Text.SmallFace, theme.TextAccent),
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
func (st *CharacterJobState) buildDetailPanel(world w.World, props JobMenuProps, itemIndex int, res resources.UIResources) *widget.Container {
	container := styled.NewVerticalContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.Image),
	)

	if itemIndex >= len(props.Items) {
		return container
	}

	prof := props.Items[itemIndex].Profession

	// 装備
	if len(prof.Equips) > 0 {
		container.AddChild(styled.NewDescriptionText(query.T(world, "Equipment"), res))
		for _, equip := range prof.Equips {
			slotLabel := string(equip.Slot)
			if slot, ok := gc.ParseEquipmentSlot(string(equip.Slot)); ok {
				slotLabel = query.T(world, slot.String())
			}
			container.AddChild(styled.NewMenuText(fmt.Sprintf(" %s %s", slotLabel, query.T(world, raw.ItemName(world.Resources.RawMaster, equip.Name))), res))
		}
	}

	// 所持品
	if len(prof.Items) > 0 {
		container.AddChild(styled.NewDescriptionText(query.T(world, "Items"), res))
		for _, item := range prof.Items {
			container.AddChild(styled.NewMenuText(fmt.Sprintf(" %s x%d", query.T(world, raw.ItemName(world.Resources.RawMaster, item.Name)), item.Count), res))
		}
	}

	// スキル
	if profSkills := raw.PtrSlice(prof.Skills); len(profSkills) > 0 {
		container.AddChild(styled.NewDescriptionText(query.T(world, "Skills"), res))
		for _, skill := range profSkills {
			skillID := gc.SkillID(skill.Id)
			name := skill.Id
			if gc.HasSkillName(skillID) {
				name = query.T(world, gc.SkillName(skillID))
			}
			container.AddChild(styled.NewMenuText(fmt.Sprintf(" %s Lv.%d", name, skill.Value), res))
		}
	}

	return container
}
