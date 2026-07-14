package states

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	gc "github.com/kijimaD/ruins/internal/components"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/route"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/gameaction"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// macroMapDebugSeed はデバッグ入口でランを自動生成する際の固定シード。
// 分布が代表的（原野が主役・POI は疎）に見えるシードを選んでいる。
const macroMapDebugSeed = 16

// MacroMapState はマクロ移動（広域グリッドマップ）の画面のステート。
// 矢印でセル単位に移動（マクロ横断のみ）、遺跡・村セルで「決定」を押すとミクロ（潜行・交易）へ。
// 寒波前線が背後の列から迫り、追いつかれるとラン失敗。
type MacroMapState struct {
	es.BaseState[w.World]
}

func (st MacroMapState) String() string {
	return "MacroMap"
}

var _ es.State[w.World] = &MacroMapState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *MacroMapState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *MacroMapState) OnResume(_ w.World) error { return nil }

// OnStop はステートが終了する際に呼ばれる
func (st *MacroMapState) OnStop(_ w.World) error { return nil }

// OnStart はステートが開始される際に呼ばれる。ラン未開始なら最小の実ラン世界を用意する（デバッグ入口用）。
func (st *MacroMapState) OnStart(world w.World) error {
	if query.GetCaravanRun(world) == nil {
		if err := setupDebugRun(world); err != nil {
			return err
		}
	}
	return nil
}

// setupDebugRun はデバッグ入口用に最小の実ラン世界（プレイヤー・隊員・ラン）を用意する。
func setupDebugRun(world w.World) error {
	if err := ensureDebugParty(world); err != nil {
		return err
	}
	query.SetCaravanRun(world, gc.NewCaravanRun(macroMapDebugSeed, route.ExpeditionDeepVault))
	return nil
}

// ensureDebugParty はプレイヤー・隊員が未生成なら生成する（デバッグ入口用。正式には母港で用意済み）。
func ensureDebugParty(world w.World) error {
	if playerExists(world) {
		return nil
	}
	player, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
	if err != nil {
		return fmt.Errorf("プレイヤーの生成に失敗: %w", err)
	}
	professions := raw.PtrSlice(world.Resources.RawMaster.Professions)
	if len(professions) > 0 {
		if err := gameaction.ApplyProfession(world, player, professions[0]); err != nil {
			return fmt.Errorf("職業の適用に失敗: %w", err)
		}
	}
	if _, err := lifecycle.SpawnDefaultSquadMember(world, player); err != nil {
		return fmt.Errorf("初期隊員の生成に失敗: %w", err)
	}
	return nil
}

// playerExists は World にプレイヤーが既に存在するかを返す。
func playerExists(world w.World) bool {
	exists := false
	query.Player(world, func(_ ecs.Entity) { exists = true })
	return exists
}

// Update はステートの更新処理。矢印で移動、決定で現在セルをエンゲージ、キャンセルで離脱。
func (st *MacroMapState) Update(world w.World) (es.Transition[w.World], error) {
	run := query.GetCaravanRun(world)
	if run == nil || run.Grid == nil {
		return es.Transition[w.World]{}, nil
	}
	action, ok := HandleMenuInput()
	if !ok {
		return es.Transition[w.World]{}, nil
	}
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionMenuUp:
		return st.move(world, run, route.Coord{X: 0, Y: -1})
	case inputmapper.ActionMenuDown:
		return st.move(world, run, route.Coord{X: 0, Y: 1})
	case inputmapper.ActionMenuLeft:
		return st.move(world, run, route.Coord{X: -1, Y: 0})
	case inputmapper.ActionMenuRight:
		return st.move(world, run, route.Coord{X: 1, Y: 0})
	case inputmapper.ActionMenuSelect:
		return st.engageCell(world, run)
	default:
		return es.Transition[w.World]{}, nil
	}
}

// move は方向 dir へ1セル移動を試みる。移動はマクロ横断のみ（ミクロには入らない）。
func (st *MacroMapState) move(world w.World, run *gc.CaravanRun, dir route.Coord) (es.Transition[w.World], error) {
	target := route.Coord{X: run.Pos.X + dir.X, Y: run.Pos.Y + dir.Y}
	if !run.CanMoveTo(target) {
		return es.Transition[w.World]{}, nil // 枠外・凍結・非隣接は無視
	}
	run.MoveTo(target)
	if run.Swallowed() {
		return st.failSwallowed(world)
	}
	if run.Pos == run.Grid.Goal {
		return st.reachGoal(world)
	}
	return es.Transition[w.World]{}, nil
}

// engageCell は現在セルの地形をエンゲージする（決定を押したときだけ）。
// 遺跡＝潜行、村/専門店/前哨＝交易、野営＝休息、目標＝達成。平原/山脈は跨ぐだけでエンゲージ対象外。
func (st *MacroMapState) engageCell(world w.World, run *gc.CaravanRun) (es.Transition[w.World], error) {
	switch run.Grid.At(run.Pos) {
	case route.NodeRuin:
		run.Dawdle(gc.RuinFrontCost)
		if run.Swallowed() {
			return st.failSwallowed(world)
		}
		gamelog.New(query.GetGameLog(world)).System(fmt.Sprintf(
			"遺跡に潜行する（寒波が %d 列詰める）。", gc.RuinFrontCost)).Log()
		return es.Transition[w.World]{
			Type:          es.TransPush,
			NewStateFuncs: []es.StateFactory[w.World]{NewRuinState(WithEscapePop())},
		}, nil
	case route.NodeMarket, route.NodeShop, route.NodeOutpost:
		gamelog.New(query.GetGameLog(world)).System(fmt.Sprintf(
			"%sに立ち寄る。", nodeTypeJP(run.Grid.At(run.Pos)))).Log()
		return es.Transition[w.World]{
			Type:          es.TransPush,
			NewStateFuncs: []es.StateFactory[w.World]{NewMarketState(WithEscapePop())},
		}, nil
	case route.NodeCamp:
		const campFoodRestore = 15
		run.Supply.Food += campFoodRestore
		run.Dawdle(gc.CampFrontCost)
		gamelog.New(query.GetGameLog(world)).System(fmt.Sprintf(
			"野営した。糧食を %d 回復したが、寒波が %d 列詰めた。", campFoodRestore, gc.CampFrontCost)).Log()
		if run.Swallowed() {
			return st.failSwallowed(world)
		}
		return es.Transition[w.World]{}, nil
	case route.NodeGoal:
		return st.reachGoal(world)
	default:
		// 平原/山脈/母港 は跨ぐ地形（エンゲージ対象外）
		return es.Transition[w.World]{}, nil
	}
}

// reachGoal は目標到達＝遠征達成の処理。ランを終了してスコア画面へ。
func (st *MacroMapState) reachGoal(world w.World) (es.Transition[w.World], error) {
	gamelog.New(query.GetGameLog(world)).System("目標地点に到達した。目標物を納品して遠征達成。").Log()
	summary := goalSummary()
	query.SetCaravanRun(world, nil)
	return es.Transition[w.World]{
		Type:          es.TransSwitch,
		NewStateFuncs: []es.StateFactory[w.World]{NewGoalResultState(summary)},
	}, nil
}

// failSwallowed は寒波前線に呑まれてラン失敗した際の処理。ランを終了して道中を閉じる。
func (st *MacroMapState) failSwallowed(world w.World) (es.Transition[w.World], error) {
	gamelog.New(query.GetGameLog(world)).System("寒波前線に呑まれた。ラン失敗。").Log()
	query.SetCaravanRun(world, nil)
	return es.Transition[w.World]{Type: es.TransPop}, nil
}

// goalSummary は遠征達成時の結果メッセージを返す。
func goalSummary() string {
	return "遠征達成！\n\n目標地点に到達し、目標物を納品した。\nキャラバンは役目を果たした。"
}

// Draw はグリッド広域マップを描画する。
func (st *MacroMapState) Draw(world w.World, screen *ebiten.Image) error {
	run := query.GetCaravanRun(world)
	if run == nil || run.Grid == nil {
		return nil
	}
	st.drawGrid(world, screen, run)
	return nil
}

// ================
// 描画（グリッド広域マップ）
// ================

func (st *MacroMapState) drawGrid(world w.World, screen *ebiten.Image, run *gc.CaravanRun) {
	g := run.Grid
	face := world.Resources.UIResources.Text.BodyFace
	sw := float64(screen.Bounds().Dx())
	sh := float64(screen.Bounds().Dy())

	screen.Fill(colorMacroBG)
	vector.FillCircle(screen, float32(sw*0.5), float32(sh*0.5), float32(sw*0.45), colorMacroBGCenter, true)

	const (
		marginX      = 60.0
		marginTop    = 118.0
		marginBottom = 66.0
	)
	mapW := sw - 2*marginX
	mapH := sh - marginTop - marginBottom
	cellW := mapW / float64(g.W)
	cellH := mapH / float64(g.H)
	cellCenter := func(c route.Coord) (float32, float32) {
		return float32(marginX + (float64(c.X)+0.5)*cellW), float32(marginTop + (float64(c.Y)+0.5)*cellH)
	}

	// セル（地形タイル）を描画
	for y := range g.H {
		for x := range g.W {
			c := route.Coord{X: x, Y: y}
			cx, cy := cellCenter(c)
			drawCell(screen, face, cx, cy, float32(cellW), float32(cellH), g.At(c), x <= run.FrontCol)
		}
	}

	// 寒波前線の縁（明るいシアン線）
	if run.FrontCol >= 0 && run.FrontCol < g.W {
		fx := float32(marginX + (float64(run.FrontCol)+1)*cellW)
		vector.StrokeLine(screen, fx, float32(marginTop), fx, float32(marginTop+mapH), 2.5, colorMacroCold, true)
	}

	// キャラバン（現在地）
	px, py := cellCenter(run.Pos)
	mr := float32(minF(cellW, cellH))
	vector.FillCircle(screen, px, py, mr*0.30, colorMacroGlowWhite, true)
	vector.FillCircle(screen, px, py, mr*0.16, colorMacroCurrent, true)
	vector.StrokeCircle(screen, px, py, mr*0.24, 2.5, colorMacroCurrent, true)

	st.drawGridOverlay(screen, face, run)
}

// drawCell は1セルを地形タイルとして描く。特別セル（遺跡・村・目標等）はリングとラベルを付ける。
func drawCell(screen *ebiten.Image, face text.Face, cx, cy, cw, ch float32, t route.NodeType, frozen bool) {
	pad := float32(3)
	fill := dimColor(nodeRingColor(t), 0.22)
	if frozen {
		fill = colorMacroFrozen
	}
	vector.FillRect(screen, cx-cw/2+pad, cy-ch/2+pad, cw-2*pad, ch-2*pad, fill, false)

	if !isSpecialCell(t) {
		return
	}
	ring := nodeRingColor(t)
	if frozen {
		ring = dimColor(ring, 0.45)
	}
	r := minF32(cw, ch) * 0.24
	vector.FillCircle(screen, cx, cy, r*0.7, colorMacroNodeFill, true)
	vector.StrokeCircle(screen, cx, cy, r, 2, ring, true)

	// 短いラベルをタイル下端に
	label := nodeTypeShort(t)
	lw, lh := text.Measure(label, face, 0)
	lx := float64(cx) - lw/2
	ly := float64(cy) + float64(r) + 2
	vector.FillRect(screen, float32(lx-3), float32(ly-1), float32(lw+6), float32(lh+2), colorMacroLabelBG, false)
	op := &text.DrawOptions{}
	op.GeoM.Translate(lx, ly)
	var labelColor color.Color = colorMacroLabel
	if frozen {
		labelColor = dimColor(colorMacroLabel, 0.5)
	}
	op.ColorScale.ScaleWithColor(labelColor)
	text.Draw(screen, label, face, op)
}

// drawGridOverlay は上部パネル（現在地・供給・寒波リード）と下部ヒントを描く。
func (st *MacroMapState) drawGridOverlay(screen *ebiten.Image, face text.Face, run *gc.CaravanRun) {
	sw := screen.Bounds().Dx()
	sh := screen.Bounds().Dy()

	const textX = 22.0
	drawText := func(s string, y int, c color.Color) {
		op := &text.DrawOptions{}
		op.GeoM.Translate(textX, float64(y))
		op.ColorScale.ScaleWithColor(c)
		text.Draw(screen, s, face, op)
	}

	styled.DrawFramedBackground(screen, 10, 10, sw-20, 92, styled.PanelStyle())
	drawText("広域マップ ── 凍原をどう横断するか", 18, theme.TextPrimary)
	drawText(fmt.Sprintf("現在地: %s　　糧食 %d ／ 燃料 %d ／ 積載 %d",
		nodeTypeJP(run.Grid.At(run.Pos)), run.Supply.Food, run.Supply.Fuel, int(run.Supply.Cargo)), 42, theme.TextPrimary)
	lead := fmt.Sprintf("❄ 寒波リード %d 列", run.FrontLead())
	if run.IsStarving() {
		lead += "　⚠飢餓：足が鈍り寒波が加速する"
	}
	drawText(lead, 66, colorMacroCold)

	// 現在セルに応じた行動プロンプト
	prompt := "↑↓←→: 移動　　キャンセル: 戻る"
	switch run.Grid.At(run.Pos) {
	case route.NodeRuin:
		prompt = "決定: 潜行する　　↑↓←→: 移動　　キャンセル: 戻る"
	case route.NodeMarket, route.NodeShop, route.NodeOutpost:
		prompt = "決定: 立ち寄る　　↑↓←→: 移動　　キャンセル: 戻る"
	case route.NodeCamp:
		prompt = "決定: 野営する　　↑↓←→: 移動　　キャンセル: 戻る"
	default:
		// 平原/山脈/母港/目標 は移動ヒントのまま
	}
	drawText(prompt, sh-26, colorMacroLabel)
}

// isSpecialCell は地形タイルに POI マーカー（リング・ラベル）を付けるかを返す。
// 平原/山脈は跨ぐだけの地形なのでタイル色だけ、それ以外は目印を付ける。
func isSpecialCell(t route.NodeType) bool {
	switch t {
	case route.NodePlain, route.NodeMountain:
		return false
	default:
		return true
	}
}

// dimColor は色の明度を factor 倍に落とす（タイル背景など暗く敷く用途）。
func dimColor(c color.Color, factor float64) color.Color {
	r, g, b, _ := c.RGBA()
	return color.RGBA{
		uint8(float64(r>>8) * factor),
		uint8(float64(g>>8) * factor),
		uint8(float64(b>>8) * factor),
		255,
	}
}

func minF(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func minF32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

// マップ描画の色（モックの暗地＋発光リングに寄せる）
var (
	colorMacroBG        = color.RGBA{10, 16, 23, 255}    // 地図背景（外側）
	colorMacroBGCenter  = color.RGBA{16, 24, 34, 130}    // 中央のビネット
	colorMacroNodeFill  = color.RGBA{17, 24, 33, 255}    // POI の暗い塗り
	colorMacroFrozen    = color.RGBA{22, 34, 52, 255}    // 凍結した後方セル（氷の青）
	colorMacroCurrent   = color.RGBA{245, 245, 245, 255} // 現在地（白）
	colorMacroLabel     = color.RGBA{220, 231, 240, 255} // ラベル文字
	colorMacroLabelBG   = color.RGBA{10, 16, 24, 225}    // ラベル背景チップ
	colorMacroCold      = color.RGBA{127, 214, 255, 255} // 寒波表示
	colorMacroGlowWhite = color.RGBA{240, 244, 250, 60}  // 白グロー（低α）
)

// nodeRingColor はノード種別の明色を返す（タイル色・POI リングに使う）。
func nodeRingColor(t route.NodeType) color.Color {
	switch t {
	case route.NodeHome:
		return color.RGBA{229, 198, 117, 255} // 金
	case route.NodeMarket:
		return color.RGBA{95, 208, 255, 255} // 集落＝クリスタル青
	case route.NodeShop:
		return color.RGBA{201, 160, 255, 255} // 専門店＝紫
	case route.NodeRuin:
		return color.RGBA{255, 138, 95, 255} // 遺跡＝橙
	case route.NodePlain:
		return color.RGBA{143, 209, 79, 255} // 平原＝緑
	case route.NodeMountain:
		return color.RGBA{143, 176, 214, 255} // 山脈＝冷たい青灰
	case route.NodeCamp:
		return color.RGBA{255, 157, 60, 255} // 野営＝火の橙
	case route.NodeOutpost:
		return color.RGBA{127, 214, 255, 255} // 前哨＝青
	case route.NodeGoal:
		return color.RGBA{255, 211, 92, 255} // 目標＝金
	default:
		return color.RGBA{160, 160, 160, 255}
	}
}

// nodeTypeJP はノード種別の表示名を返す。
func nodeTypeJP(t route.NodeType) string {
	switch t {
	case route.NodeHome:
		return "母港"
	case route.NodeMarket:
		return "集落マーケット"
	case route.NodeShop:
		return "専門店"
	case route.NodeRuin:
		return "遺跡"
	case route.NodeCamp:
		return "野営地"
	case route.NodePlain:
		return "平原"
	case route.NodeMountain:
		return "山脈"
	case route.NodeOutpost:
		return "前哨"
	case route.NodeGoal:
		return "目標地点"
	default:
		return "地点"
	}
}

// nodeTypeShort は広域マップ上の POI ラベル用の短い表示名を返す。
func nodeTypeShort(t route.NodeType) string {
	switch t {
	case route.NodeHome:
		return "母港"
	case route.NodeMarket:
		return "集落"
	case route.NodeShop:
		return "専門店"
	case route.NodeRuin:
		return "遺跡"
	case route.NodeCamp:
		return "野営"
	case route.NodeOutpost:
		return "前哨"
	case route.NodeGoal:
		return "目標"
	default:
		return ""
	}
}
