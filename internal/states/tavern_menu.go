package states

import (
	"fmt"
	"math/rand/v2"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/menurt"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/menuscreen"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// TavernMenuState は酒場の雇用画面のゲームステート
type TavernMenuState struct {
	es.BaseState[w.World]
	actionWin  menuscreen.ActionWindow // 雇用のアクション選択。overlay として Screen に登録する
	detail     menuscreen.Detail       // 候補の能力・費用を出す詳細モーダル。overlay として Screen に登録する
	screen     menurt.Screen[TavernProps]
	candidates []tavernCandidate
}

var _ es.State[w.World] = &TavernMenuState{}
var _ Configurable = &TavernMenuState{}

// StateConfig は背景のブラーと暗幕を無効にする。後ろのフィールドをそのまま見せる
func (st *TavernMenuState) StateConfig() StateConfig {
	return StateConfig{BlurBackground: false}
}

// OnStart はステートが開始する際に呼ばれる
func (st *TavernMenuState) OnStart(world w.World) error {
	st.actionWin = menuscreen.NewActionWindow(st.actionWindowContent)
	st.detail = menuscreen.NewDetail(st.detailContent)
	st.screen = menurt.NewScreen[TavernProps](st, &st.detail, &st.actionWin)
	st.candidates = generateCandidates(world.Config.RNG)
	return nil
}

// Update はステートの更新処理を行う
func (st *TavernMenuState) Update(world w.World) (es.Transition[w.World], error) {
	return st.screen.Update(world)
}

// Draw はステートの描画処理を行う
func (st *TavernMenuState) Draw(_ w.World, screen *ebiten.Image) error {
	st.screen.Draw(screen)
	return nil
}

// ExtraInput は共通入力に加える独自キーを返す。x で選択中候補の詳細モーダルを開く
func (st *TavernMenuState) ExtraInput() (inputmapper.ActionID, bool) {
	ki := input.GetSharedKeyboardInput()
	if ki.IsKeyJustPressed(ebiten.KeyX) && !ki.IsKeyPressed(ebiten.KeyShift) {
		return inputmapper.ActionOpenItemDetail, true
	}
	return "", false
}

// DoAction はアクションを実行してステート遷移を返す
func (st *TavernMenuState) DoAction(_ w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionOpenItemDetail:
		st.detail.Open()
	case inputmapper.ActionMenuSelect:
		st.actionWin.Open()
	case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown, inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight, inputmapper.ActionMenuTabNext, inputmapper.ActionMenuTabPrev:
		// Dispatchで処理
	default:
		return es.Transition[w.World]{}, fmt.Errorf("tavernMenu: 未対応のアクション: %s", action)
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
	"general",
}

// generateCandidates はランダムな雇用候補を生成する。
func generateCandidates(rng *rand.Rand) []tavernCandidate {
	count := 3 + rng.IntN(3) // 3〜5人
	used := make(map[string]bool)
	var candidates []tavernCandidate

	for range count {
		if len(used) >= len(candidateNamePool) {
			break
		}
		// 名前の重複を避ける
		var name string
		for {
			name = candidateNamePool[rng.IntN(len(candidateNamePool))]
			if !used[name] {
				used[name] = true
				break
			}
		}

		abilities := randomAbilities(rng)
		cost := calculateHiringCost(abilities)
		spriteKey := candidateSpritePool[rng.IntN(len(candidateSpritePool))]

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
func randomAbilities(rng *rand.Rand) gc.Abilities {
	randStat := func() int { return 4 + rng.IntN(8) } // 4〜11
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

// TavernProps は画面の表示 props。menurt.Screen の型引数として渡す
type TavernProps struct {
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

// Fetch は世界から表示 props を構築する。menurt.Model の Model 部にあたる
func (st *TavernMenuState) Fetch(world w.World) TavernProps {
	var currency int
	query.Player(world, func(playerEntity ecs.Entity) {
		currency = query.GetCurrency(world, playerEntity)
	})

	candidates := make([]tavernCandidateData, 0, len(st.candidates))
	for i := range st.candidates {
		c := &st.candidates[i]
		candidates = append(candidates, tavernCandidateData{
			Index:     i,
			Name:      c.Name,
			Stats:     fmt.Sprintf("体%d 力%d 感%d 器%d 敏%d 防%d", c.Abilities.Vitality.Base, c.Abilities.Strength.Base, c.Abilities.Sensation.Base, c.Abilities.Dexterity.Base, c.Abilities.Agility.Base, c.Abilities.Defense.Base),
			Cost:      c.Cost,
			CanAfford: currency >= c.Cost,
		})
	}

	return TavernProps{
		Candidates: candidates,
		Currency:   currency,
	}
}

// ================
// Window
// ================

// Menu は一覧の構成を返す。menurt.Model の Menu 部にあたる
func (st *TavernMenuState) Menu(props TavernProps) menurt.MenuConfig {
	return menurt.MenuConfig{Key: "tavern", TabCount: 1, ItemCounts: []int{len(props.Candidates)}}
}

// actionWindowContent は現在カーソルが当たっている候補の見出しと選択肢を返す。アクション窓の唯一の定義点。
// 雇用の実行内容も Run に閉じ込め、雇用・閉じるを1箇所で定義する
func (st *TavernMenuState) actionWindowContent(_ w.World) (string, []menuscreen.Action, bool) {
	c, ok := st.selectedCandidate()
	if !ok || c.Name == "" {
		return "", nil, false
	}
	var actions []menuscreen.Action
	if c.CanAfford {
		idx := c.Index
		actions = append(actions, menuscreen.Action{Label: TextHire, Run: func(world w.World) error {
			return st.hireCandidate(world, idx)
		}})
	}
	actions = append(actions, menuscreen.Action{Label: TextClose})
	return c.Name, actions, true
}

// selectedCandidate は現在カーソルが当たっている雇用候補を返す
func (st *TavernMenuState) selectedCandidate() (tavernCandidateData, bool) {
	props := st.screen.Props()
	sel := st.screen.Selection()
	if sel.ItemIndex >= len(props.Candidates) {
		return tavernCandidateData{}, false
	}
	return props.Candidates[sel.ItemIndex], true
}

// hireCandidate は idx 番目の候補を雇用し、候補リストから取り除く。所持金が足りなければ何もしない
func (st *TavernMenuState) hireCandidate(world w.World, idx int) error {
	if idx < 0 || idx >= len(st.candidates) {
		return nil
	}
	candidate := st.candidates[idx]

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

	st.candidates = append(st.candidates[:idx], st.candidates[idx+1:]...)
	return nil
}

// ================
// buildUI
// ================

// View は props を UI へ組む純粋な描画。menurt.Model の View 部にあたる
func (st *TavernMenuState) View(_ w.World, props TavernProps, sel menurt.Selection, res resources.UIResources) *ebitenui.UI {
	content := styled.NewVerticalContainer()
	content.AddChild(newCurrencyRow(props.Currency, res))
	content.AddChild(st.buildCandidateTable(props.Candidates, sel.ItemIndex, res))
	return newTabScreenUI(res, tabScreen{Content: content, Footer: menuNavHint(false, "x 詳細")})
}

// buildCandidateTable は雇用候補を名前のみの1カラムで並べる。能力・費用は x の詳細モーダルで見る
func (st *TavernMenuState) buildCandidateTable(candidates []tavernCandidateData, selectedIndex int, res resources.UIResources) *widget.Container {
	rows := make([]menuRow, len(candidates))
	for i, c := range candidates {
		rows[i] = menuRow{Cells: []string{c.Name}}
	}
	return renderMenuList(selectedIndex, rows, []int{menuRowWidth}, []styled.TextAlign{styled.AlignLeft}, menuListOpts{AlwaysIndicator: true, EmptyText: "雇用できる候補がいません"}, res)
}

// detailContent は現在カーソルが当たっている候補の能力と費用を返す。詳細モーダルの唯一の定義点
func (st *TavernMenuState) detailContent(_ w.World) (menuscreen.DetailContent, bool) {
	c, ok := st.selectedCandidate()
	if !ok {
		return menuscreen.DetailContent{}, false
	}
	return menuscreen.DetailContent{
		Name: c.Name,
		Desc: c.Stats,
		Rows: []menuscreen.SpecRow{{Label: "費用", Value: query.FormatCurrency(c.Cost)}},
	}, true
}
