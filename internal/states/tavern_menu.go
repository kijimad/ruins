package states

import (
	"fmt"
	"math/rand/v2"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/config"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
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
	screen     Screen[tavernProps]
	candidates []tavernCandidate
}

var _ es.State[w.World] = &TavernMenuState{}
var _ Configurable = &TavernMenuState{}

// StateConfig は背景のブラーと暗幕を無効にする。後ろのフィールドをそのまま見せる
func (st *TavernMenuState) StateConfig() StateConfig {
	return StateConfig{BlurBackground: false}
}

var _ es.ActionHandler[w.World] = &TavernMenuState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *TavernMenuState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *TavernMenuState) OnResume(_ w.World) error { return nil }

// OnStop はステートが終了する際に呼ばれる
func (st *TavernMenuState) OnStop(_ w.World) error { return nil }

// OnStart はステートが開始する際に呼ばれる
func (st *TavernMenuState) OnStart(world w.World) error {
	st.actionWin = menuscreen.NewActionWindow(st.actionWindowContent)
	st.screen = NewScreen[tavernProps](&st.actionWin)
	st.candidates = generateCandidates(world.Config.RNG)
	return nil
}

// Update はステートの更新処理を行う
func (st *TavernMenuState) Update(world w.World) (es.Transition[w.World], error) {
	return st.screen.Update(world, st)
}

// Draw はステートの描画処理を行う
func (st *TavernMenuState) Draw(_ w.World, screen *ebiten.Image) error {
	st.screen.Draw(screen)
	return nil
}

// HandleInput は入力を処理してアクションIDを返す
func (st *TavernMenuState) HandleInput(_ *config.Config) (inputmapper.ActionID, bool) {
	return HandleMenuInput()
}

// DoAction はアクションを実行してステート遷移を返す
func (st *TavernMenuState) DoAction(_ w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionMenuSelect:
		st.actionWin.Open()
		st.screen.MarkDirty()
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

func (st *TavernMenuState) fetch(world w.World) tavernProps {
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

	return tavernProps{
		Candidates: candidates,
		Currency:   currency,
	}
}

// ================
// Window
// ================

// actionWindowContent は現在カーソルが当たっている候補の見出しと選択肢を返す。アクション窓の唯一の定義点。
// 雇用の実行内容も Run に閉じ込め、雇用・閉じるを1箇所で定義する
func (st *TavernMenuState) menu(props tavernProps) MenuConfig {
	return MenuConfig{Key: "tavern", TabCount: 1, ItemCounts: []int{len(props.Candidates)}}
}

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

func (st *TavernMenuState) view(_ w.World, props tavernProps, sel Selection, res resources.UIResources) *ebitenui.UI {
	content := styled.NewVerticalContainer()
	content.AddChild(newCurrencyRow(props.Currency, res))
	content.AddChild(st.buildCandidateTable(props.Candidates, sel.ItemIndex, res))
	return newTabScreenUI(res, tabScreen{Content: content, Footer: menuNavHint(false)})
}

func (st *TavernMenuState) buildCandidateTable(candidates []tavernCandidateData, selectedIndex int, res resources.UIResources) *widget.Container {
	columnWidths := []int{20, 60, 180, 80}
	aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignLeft, styled.AlignLeft, styled.AlignRight}
	rows := make([]menuRow, len(candidates))
	for i, c := range candidates {
		rows[i] = menuRow{Cells: []string{"", c.Name, c.Stats, query.FormatCurrency(c.Cost)}}
	}
	// 名前・能力・費用の列見出しを固定で置く。選択やページ送りの対象には含めない
	return renderMenuList(selectedIndex, rows, columnWidths, aligns, menuListOpts{HeaderRow: []string{"", "名前", "能力", "費用"}, EmptyText: "雇用できる候補がいません"}, res)
}
