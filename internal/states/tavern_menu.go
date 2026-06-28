package states

import (
	"fmt"
	"math/rand"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/config"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// tavernSubState はサブステート
type tavernSubState int

const (
	tavernSubStateMenu   tavernSubState = iota // 候補一覧
	tavernSubStateWindow                       // アクションウィンドウ
)

// TavernMenuState は酒場の雇用画面のゲームステート
type TavernMenuState struct {
	es.BaseState[w.World]
	subState    tavernSubState
	menuMount   *hooks.Mount[tavernProps]
	windowMount *hooks.Mount[tavernWindowProps]
	widget      *ebitenui.UI
	candidates  []tavernCandidate
}

func (st TavernMenuState) String() string {
	return "TavernMenu"
}

var _ es.State[w.World] = &TavernMenuState{}
var _ es.ActionHandler[w.World] = &TavernMenuState{}

func (st *TavernMenuState) OnPause(_ w.World) error  { return nil }
func (st *TavernMenuState) OnResume(_ w.World) error  { return nil }
func (st *TavernMenuState) OnStop(_ w.World) error    { return nil }
func (st *TavernMenuState) OnStart(_ w.World) error {
	st.subState = tavernSubStateMenu
	st.menuMount = hooks.NewMount[tavernProps]()
	st.windowMount = hooks.NewMount[tavernWindowProps]()
	st.candidates = generateCandidates()
	return nil
}

func (st *TavernMenuState) Update(world w.World) (es.Transition[w.World], error) {
	if action, ok := st.HandleInput(world.Config); ok {
		if transition, err := st.DoAction(world, action); err != nil {
			return es.Transition[w.World]{}, err
		} else if transition.Type != es.TransNone {
			return transition, nil
		}
		switch st.subState {
		case tavernSubStateMenu:
			st.menuMount.Dispatch(action)
		case tavernSubStateWindow:
			st.windowMount.Dispatch(action)
		}
	}

	props := st.fetchProps(world)
	st.menuMount.SetProps(props)

	hooks.UseTabMenu(st.menuMount.Store(), "tavern", hooks.TabMenuConfig{
		TabCount:   1,
		ItemCounts: []int{len(props.Candidates)},
	})

	st.setupWindowState()

	menuDirty := st.menuMount.Update()
	windowDirty := st.windowMount.Update()
	if menuDirty || windowDirty || st.widget == nil {
		st.widget = st.buildUI(world)
	}

	st.widget.Update()
	return st.ConsumeTransition(), nil
}

func (st *TavernMenuState) Draw(_ w.World, screen *ebiten.Image) error {
	st.widget.Draw(screen)
	return nil
}

func (st *TavernMenuState) HandleInput(_ *config.Config) (inputmapper.ActionID, bool) {
	switch st.subState {
	case tavernSubStateMenu:
		return HandleMenuInput()
	case tavernSubStateWindow:
		return HandleWindowInput()
	}
	return "", false
}

func (st *TavernMenuState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch st.subState {
	case tavernSubStateWindow:
		switch action {
		case inputmapper.ActionWindowConfirm:
			if err := st.executeWindowAction(world); err != nil {
				return es.Transition[w.World]{}, err
			}
		case inputmapper.ActionWindowCancel:
			st.subState = tavernSubStateMenu
		case inputmapper.ActionWindowUp, inputmapper.ActionWindowDown:
			// Dispatchで処理
		default:
			return es.Transition[w.World]{}, fmt.Errorf("tavernSubStateWindow: 未対応のアクション: %s", action)
		}

	case tavernSubStateMenu:
		switch action {
		case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
			return es.Transition[w.World]{Type: es.TransPop}, nil
		case inputmapper.ActionMenuSelect:
			st.handleCandidateSelection()
		case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown, inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight, inputmapper.ActionMenuTabNext, inputmapper.ActionMenuTabPrev:
			// Dispatchで処理
		default:
			return es.Transition[w.World]{}, fmt.Errorf("tavernSubStateMenu: 未対応のアクション: %s", action)
		}
	}
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

// ================
// 候補生成
// ================

// tavernCandidate は雇用候補のデータ
type tavernCandidate struct {
	Name      string
	Abilities gc.Abilities
	SpriteKey string
	Cost      int
}

// candidateNamePool は候補名のプール
var candidateNamePool = []string{
	"ジン", "カイ", "レン", "ミラ", "セイ",
	"ノア", "リク", "ユウ", "ハル", "ソラ",
}

// candidateSpritePool は候補スプライトのプール
var candidateSpritePool = []string{
	"player", "player",
}

// generateCandidates はランダムな雇用候補を生成する
func generateCandidates() []tavernCandidate {
	count := 3 + rand.Intn(3) // 3〜5人
	used := make(map[string]bool)
	var candidates []tavernCandidate

	for range count {
		if len(used) >= len(candidateNamePool) {
			break
		}
		// 名前の重複を避ける
		var name string
		for {
			name = candidateNamePool[rand.Intn(len(candidateNamePool))]
			if !used[name] {
				used[name] = true
				break
			}
		}

		abilities := randomAbilities()
		cost := calculateHiringCost(abilities)
		spriteKey := candidateSpritePool[rand.Intn(len(candidateSpritePool))]

		candidates = append(candidates, tavernCandidate{
			Name:      name,
			Abilities: abilities,
			SpriteKey: spriteKey,
			Cost:      cost,
		})
	}

	return candidates
}

// randomAbilities はランダムな能力値を生成する
func randomAbilities() gc.Abilities {
	randStat := func() int { return 4 + rand.Intn(8) } // 4〜11
	return gc.Abilities{
		Vitality:  gc.Ability{Base: randStat()},
		Strength:  gc.Ability{Base: randStat()},
		Sensation: gc.Ability{Base: randStat()},
		Dexterity: gc.Ability{Base: randStat()},
		Agility:   gc.Ability{Base: randStat()},
		Defense:   gc.Ability{Base: randStat()},
	}
}

// calculateHiringCost は能力値から雇用コストを算出する
func calculateHiringCost(a gc.Abilities) int {
	total := a.Vitality.Base + a.Strength.Base + a.Sensation.Base +
		a.Dexterity.Base + a.Agility.Base + a.Defense.Base
	return total * 30
}

// ================
// Props
// ================

type tavernProps struct {
	Candidates []tavernCandidateData
	Currency   int
}

type tavernCandidateData struct {
	Index     int
	Name      string
	Stats     string
	Cost      int
	CanAfford bool
}

type tavernWindowProps struct {
	Candidate tavernCandidateData
}

func (st *TavernMenuState) fetchProps(world w.World) tavernProps {
	var currency int
	query.Player(world, func(playerEntity ecs.Entity) {
		currency = query.GetCurrency(world, playerEntity)
	})

	var candidates []tavernCandidateData
	for i, c := range st.candidates {
		candidates = append(candidates, tavernCandidateData{
			Index:     i,
			Name:      c.Name,
			Stats:     fmt.Sprintf("体%d 力%d 感%d 器%d 敏%d 防%d", c.Abilities.Vitality.Base, c.Abilities.Strength.Base, c.Abilities.Sensation.Base, c.Abilities.Dexterity.Base, c.Abilities.Agility.Base, c.Abilities.Defense.Base),
			Cost:      c.Cost,
			CanAfford: currency >= c.Cost,
		})
	}

	return tavernProps{
		Candidates: candidates,
		Currency:   currency,
	}
}

// ================
// Window
// ================

func (st *TavernMenuState) setupWindowState() {
	actionItems := st.getActionItems()

	hooks.UseState(st.windowMount.Store(), "tavern_window_index", 0, func(v int, action inputmapper.ActionID) int {
		switch action {
		case inputmapper.ActionWindowUp:
			if v > 0 {
				return v - 1
			}
			return len(actionItems) - 1
		case inputmapper.ActionWindowDown:
			if v < len(actionItems)-1 {
				return v + 1
			}
			return 0
		default:
			return v
		}
	})
}

func (st *TavernMenuState) getActionItems() []string {
	windowProps := st.windowMount.GetProps()
	if windowProps.Candidate.Name == "" {
		return []string{TextClose}
	}
	items := []string{}
	if windowProps.Candidate.CanAfford {
		items = append(items, "雇用する")
	}
	items = append(items, TextClose)
	return items
}

func (st *TavernMenuState) handleCandidateSelection() {
	props := st.menuMount.GetProps()
	menuState, ok := hooks.GetState[hooks.TabMenuState](st.menuMount, "tavern")
	if !ok {
		return
	}
	itemIndex := menuState.ItemIndex
	if itemIndex >= len(props.Candidates) {
		return
	}

	st.subState = tavernSubStateWindow
	st.windowMount = hooks.NewMount[tavernWindowProps]()
	st.windowMount.SetProps(tavernWindowProps{
		Candidate: props.Candidates[itemIndex],
	})
}

func (st *TavernMenuState) executeWindowAction(world w.World) error {
	windowProps := st.windowMount.GetProps()
	actionIndex, ok := hooks.GetState[int](st.windowMount, "tavern_window_index")
	if !ok {
		return nil
	}
	actionItems := st.getActionItems()
	if actionIndex >= len(actionItems) {
		return nil
	}

	selectedAction := actionItems[actionIndex]
	switch selectedAction {
	case "雇用する":
		candidate := st.candidates[windowProps.Candidate.Index]

		playerEntity, err := query.GetPlayerEntity(world)
		if err != nil {
			return err
		}

		if !query.ConsumeCurrency(world, playerEntity, candidate.Cost) {
			return nil
		}

		if _, err := lifecycle.SpawnSquadMember(world, playerEntity, candidate.Name, candidate.Abilities, candidate.SpriteKey); err != nil {
			return fmt.Errorf("雇用に失敗: %w", err)
		}

		// 候補リストから削除する
		st.candidates = append(st.candidates[:windowProps.Candidate.Index], st.candidates[windowProps.Candidate.Index+1:]...)
		st.subState = tavernSubStateMenu

	case TextClose:
		st.subState = tavernSubStateMenu
	}

	return nil
}

// ================
// buildUI
// ================

func (st *TavernMenuState) buildUI(world w.World) *ebitenui.UI {
	res := world.Resources.UIResources
	props := st.menuMount.GetProps()
	menuState, _ := hooks.GetState[hooks.TabMenuState](st.menuMount, "tavern")
	itemIndex := menuState.ItemIndex

	root := styled.NewVerticalContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
	)

	root.AddChild(styled.NewTitleText("酒場", res))
	root.AddChild(st.buildCurrencyRow(props.Currency, res))
	root.AddChild(st.buildCandidateTable(props.Candidates, itemIndex, res))

	eui := &ebitenui.UI{Container: root}

	if st.subState == tavernSubStateWindow {
		window := st.buildActionWindow(world)
		eui.AddWindow(window)
	}

	return eui
}

func (st *TavernMenuState) buildCurrencyRow(currency int, res resources.UIResources) *widget.Container {
	container := styled.NewRowContainer()
	container.AddChild(styled.NewMenuText(query.FormatCurrency(currency), res))
	return container
}

func (st *TavernMenuState) buildCandidateTable(candidates []tavernCandidateData, selectedIndex int, res resources.UIResources) *widget.Container {
	container := styled.NewVerticalContainer()

	if len(candidates) == 0 {
		container.AddChild(styled.NewDescriptionText("雇用できる候補がいません", res))
		return container
	}

	columnWidths := []int{20, 60, 180, 80}
	aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignLeft, styled.AlignLeft, styled.AlignRight}

	table := styled.NewTableContainer(columnWidths, res)
	styled.NewTableHeaderRow(table, columnWidths, []string{"", "名前", "能力", "費用"}, res)

	for i, c := range candidates {
		isSelected := i == selectedIndex
		costStr := query.FormatCurrency(c.Cost)
		styled.NewTableRow(table, columnWidths, []string{"", c.Name, c.Stats, costStr}, aligns, &isSelected, res)
	}

	container.AddChild(table)
	return container
}

func (st *TavernMenuState) buildActionWindow(world w.World) *widget.Window {
	res := world.Resources.UIResources
	actionIndex, _ := hooks.GetState[int](st.windowMount, "tavern_window_index")
	actionItems := st.getActionItems()
	windowProps := st.windowMount.GetProps()

	windowContainer := styled.NewWindowContainer(res)
	titleContainer := styled.NewWindowHeaderContainer(windowProps.Candidate.Name, res)
	window := styled.NewSmallWindow(titleContainer, windowContainer)

	for i, action := range actionItems {
		isSelected := i == actionIndex
		windowContainer.AddChild(styled.NewListItemText(action, theme.TextSecondary, isSelected, res))
	}

	window.SetLocation(getCenterWinRect(world))
	return window
}
