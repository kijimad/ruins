package states

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
	"github.com/kijimaD/ruins/internal/menuloop"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/ui"
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
	screen     *menuloop.Screen[JobMenuProps]
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
	st.screen = menuloop.NewScreen[JobMenuProps](st)
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
	default:
		return es.Transition[w.World]{}, fmt.Errorf("characterJob: unsupported action: %s", action)
	}
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

// Fetch は世界から表示 props を構築する。menuloop.Model の Model 部にあたる
func (st *CharacterJobState) Fetch(world w.World) (JobMenuProps, error) {
	professions := raw.PtrSlice(world.Resources.RawMaster.Professions)
	items := make([]jobMenuItem, len(professions))
	for i := range professions {
		items[i] = jobMenuItem{Profession: professions[i]}
	}
	return JobMenuProps{Items: items}, nil
}

// Menu は一覧の構成を返す。menuloop.Model の Menu 部にあたる
func (st *CharacterJobState) Menu(props JobMenuProps) menuloop.MenuConfig {
	return menuloop.MenuConfig{Key: "job", TabCount: 1, ItemCounts: []int{len(props.Items)}}
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

// ViewUI は View の internal/ui 版。見出し・左の職業一覧・右の詳細パネル・下の説明を Group で絶対配置する。
func (st *CharacterJobState) ViewUI(world w.World, props JobMenuProps, cursor menuloop.Selection, res resources.UIResources) ui.Widget {
	sd := world.Resources.ScreenDimensions
	itemIndex := cursor.ItemIndex
	face := res.Text.BodyFace
	children := make([]ui.Widget, 0, 5)

	title := ui.NewText(query.T(world, "Profession"), res.Text.TitleFontFace, theme.TextPrimary)
	title.Align = ui.AlignCenter
	title.Layout(image.Rect(0, 24, sd.Width, 60))
	children = append(children, title)

	rows := make([]menuRow, len(props.Items))
	for i := range props.Items {
		rows[i] = menuRow{Cells: styled.TextCells(query.T(world, props.Items[i].Profession.Name))}
	}
	listRows := renderMenuListUI(itemIndex, rows, []int{160}, []styled.TextAlign{styled.AlignLeft}, menuListOpts{Spaced: true}, res)
	list := ui.VBox(panelScreenRowH, listRows...)
	list.Layout(image.Rect(40, 80, 40+180, sd.Height-72))
	children = append(children, list)

	detailRows := buildJobDetailRowsUI(world, props, itemIndex, face)
	detail := panelBackground(ui.Panel(ui.BoxStyle{}, panelScreenRowH, detailRows...), res).SetPadding(panelScreenPad)
	detail.Layout(image.Rect(240, 80, sd.Width-40, sd.Height-72))
	children = append(children, detail)

	description := ""
	if itemIndex < len(props.Items) {
		description = query.T(world, props.Items[itemIndex].Profession.Description)
	}
	desc := ui.NewText(description, res.Text.SmallFace, theme.TextAccent)
	desc.Align = ui.AlignCenter
	desc.Layout(image.Rect(0, sd.Height-56, sd.Width, sd.Height-40))
	hint := ui.NewText(keybind.HelpHint(world), res.Text.SmallFace, theme.TextAccent)
	hint.Align = ui.AlignCenter
	hint.Layout(image.Rect(0, sd.Height-32, sd.Width, sd.Height-16))
	children = append(children, desc, hint)

	root := ui.NewGroup(children...)
	root.Layout(image.Rect(0, 0, sd.Width, sd.Height))
	return root
}

// buildJobDetailRowsUI は buildDetailPanel の internal/ui 版。装備・所持品・スキルの見出しと行を返す。
func buildJobDetailRowsUI(world w.World, props JobMenuProps, itemIndex int, face text.Face) []ui.Widget {
	if itemIndex >= len(props.Items) {
		return nil
	}
	prof := props.Items[itemIndex].Profession
	var items []ui.Widget
	section := func(title string) {
		items = append(items, ui.NewText(title, face, theme.TextSecondary))
	}
	line := func(s string) {
		items = append(items, ui.NewText(s, face, theme.TextPrimary))
	}
	if len(prof.Equips) > 0 {
		section(query.T(world, "Equipment"))
		for _, equip := range prof.Equips {
			slotLabel := string(equip.Slot)
			if slot, ok := gc.ParseEquipmentSlot(string(equip.Slot)); ok {
				slotLabel = query.T(world, slot.String())
			}
			line(fmt.Sprintf(" %s %s", slotLabel, query.T(world, raw.ItemName(world.Resources.RawMaster, equip.Name))))
		}
	}
	if len(prof.Items) > 0 {
		section(query.T(world, "Items"))
		for _, item := range prof.Items {
			line(fmt.Sprintf(" %s x%d", query.T(world, raw.ItemName(world.Resources.RawMaster, item.Name)), item.Count))
		}
	}
	if profSkills := raw.PtrSlice(prof.Skills); len(profSkills) > 0 {
		section(query.T(world, "Skills"))
		for _, skill := range profSkills {
			skillID := gc.SkillID(skill.Id)
			name := skill.Id
			if gc.HasSkillName(skillID) {
				name = query.T(world, gc.SkillName(skillID))
			}
			line(fmt.Sprintf(" %s Lv.%d", name, skill.Value))
		}
	}
	return items
}
