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
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/messagedata"
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
// イベント種別が揃って見えるシードを選んでいる。
const macroMapDebugSeed = 15

// MacroMapState はマクロ移動（FTL 風の停留点マップ）の画面のステート。
// 次の停留点を選んでジャンプし、到達先のイベント（遺跡=潜行/村=交易/吹雪=選択/…）を解決する。
// 寒波前線が背後の列から迫り、追いつかれるとラン失敗。
type MacroMapState struct {
	es.BaseState[w.World]
	sel int // 選択中の次停留点インデックス（Outgoing の中）
	// divingRuin は直近の Push が遺跡潜行かを示す（戻り時の物資回収判定に使う）
	divingRuin bool
	// cargoValueBeforeDive は潜行開始時の背嚢価値（戻り時に稼いだ差分を出す）
	cargoValueBeforeDive int
}

func (st MacroMapState) String() string {
	return "MacroMap"
}

var _ es.State[w.World] = &MacroMapState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *MacroMapState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる。サブステート（潜行・交易）から戻ったとき、
// 背嚢の戦利品を広域のラン状態へ帰結させる（micro→macro の密結合）。
func (st *MacroMapState) OnResume(world w.World) error {
	run := query.GetCaravanRun(world)
	if run == nil {
		return nil
	}
	// 戦利品は積載重量になり、以後のジャンプ供給消費を増やす（物量で頂点＝稼ぐほど重く飢える）
	run.Supply.Cargo = backpackWeight(world)
	// 潜行から戻ったら、稼いだ戦利品価値の一部を糧食/燃料として回収する（潜ると供給が延びる）
	if st.divingRuin {
		st.divingRuin = false
		gained := backpackValue(world) - st.cargoValueBeforeDive
		if gained > 0 {
			food := gained / 8
			fuel := gained / 16
			run.Supply.Food += food
			run.Supply.Fuel += fuel
			gamelog.New(query.GetGameLog(world)).System(fmt.Sprintf(
				"遺跡から物資を回収した（糧食 +%d ／ 燃料 +%d）。", food, fuel)).Log()
		}
	}
	return nil
}

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

// Update はステートの更新処理。上下で次停留点を選び、決定でジャンプ、キャンセルで離脱。
func (st *MacroMapState) Update(world w.World) (es.Transition[w.World], error) {
	run := query.GetCaravanRun(world)
	if run == nil || run.Beacons == nil {
		return es.Transition[w.World]{}, nil
	}

	// デバッグ: G キーで目標地点へ即到達する（cfg.Debug 時のみ。到達フローの動作確認用）
	if world.Config.Debug && input.GetSharedKeyboardInput().IsKeyJustPressed(ebiten.KeyG) {
		run.Current = run.Beacons.Goal
		return st.reachGoal(world)
	}

	next := run.Beacons.Outgoing(run.Current)

	action, ok := HandleMenuInput()
	if !ok {
		return es.Transition[w.World]{}, nil
	}
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionMenuUp:
		st.sel = cycle(st.sel-1, len(next))
		return es.Transition[w.World]{}, nil
	case inputmapper.ActionMenuDown:
		st.sel = cycle(st.sel+1, len(next))
		return es.Transition[w.World]{}, nil
	case inputmapper.ActionMenuSelect:
		if len(next) == 0 {
			return es.Transition[w.World]{}, nil
		}
		to := next[cycle(st.sel, len(next))]
		st.sel = 0
		return st.jump(world, run, to)
	default:
		return es.Transition[w.World]{}, nil
	}
}

// jump は次の停留点へジャンプし、到達先のイベントを解決する。
func (st *MacroMapState) jump(world w.World, run *gc.CaravanRun, to route.NodeID) (es.Transition[w.World], error) {
	run.JumpTo(to)
	if run.Swallowed() {
		return st.failSwallowed(world)
	}
	b := run.Beacons.BeaconByID(run.Current)
	if b == nil {
		return es.Transition[w.World]{}, nil
	}
	return st.resolveEvent(world, run, b.Event)
}

// resolveEvent は到達した停留点のイベントを解決する。
// 遺跡=潜行、村/専門店/前哨=交易、野営=休息、山脈=吹雪(選択)、平原=無風、目標=達成。
func (st *MacroMapState) resolveEvent(world w.World, run *gc.CaravanRun, ev route.NodeType) (es.Transition[w.World], error) {
	log := gamelog.New(query.GetGameLog(world))
	switch ev {
	case route.NodeGoal:
		return st.reachGoal(world)
	case route.NodeRuin:
		run.Dawdle(gc.RuinFrontCost)
		if run.Swallowed() {
			return st.failSwallowed(world)
		}
		// 戻り時に「稼いだ戦利品」を出せるよう、潜行前の背嚢価値を控える
		st.divingRuin = true
		st.cargoValueBeforeDive = backpackValue(world)
		log.System(fmt.Sprintf("遺跡に潜行する（寒波が %d 列詰める）。", gc.RuinFrontCost)).Log()
		return es.Transition[w.World]{
			Type:          es.TransPush,
			NewStateFuncs: []es.StateFactory[w.World]{NewRuinState(WithEscapePop())},
		}, nil
	case route.NodeMarket, route.NodeShop, route.NodeOutpost:
		log.System(fmt.Sprintf("%sに立ち寄る。", nodeTypeJP(ev))).Log()
		return es.Transition[w.World]{
			Type:          es.TransPush,
			NewStateFuncs: []es.StateFactory[w.World]{NewMarketState(WithEscapePop())},
		}, nil
	case route.NodeCamp:
		const campFoodRestore = 20
		run.Supply.Food += campFoodRestore
		run.Dawdle(gc.CampFrontCost)
		log.System(fmt.Sprintf("野営した。糧食を %d 回復したが、寒波が %d 列詰めた。", campFoodRestore, gc.CampFrontCost)).Log()
		if run.Swallowed() {
			return st.failSwallowed(world)
		}
		return es.Transition[w.World]{}, nil
	case route.NodeMountain:
		// 吹雪：燃料を焚くか凍えるかの選択イベント
		return es.Transition[w.World]{
			Type:          es.TransPush,
			NewStateFuncs: []es.StateFactory[w.World]{NewBlizzardEventState()},
		}, nil
	default:
		// 平原など：無風（小さな採集）
		run.Supply.Food += 4
		log.System("穏やかな雪原を越えた。わずかに糧を得た（糧食 +4）。").Log()
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

// NewBlizzardEventState は吹雪の選択イベント（燃料を焚く / 凍えて進む）を作る。
func NewBlizzardEventState() es.StateFactory[w.World] {
	return func() (es.State[w.World], error) {
		ms := &MessageState{}
		ms.messageData = messagedata.NewSystemMessage("吹雪に見舞われた。凍てつく風が隊列を叩く。どうする？").
			WithChoice("燃料を焚いて凌ぐ（燃料 -8）", func(world w.World) error {
				if run := query.GetCaravanRun(world); run != nil {
					run.Supply.Fuel = clampMin0(run.Supply.Fuel - 8)
				}
				ms.SetTransition(es.Transition[w.World]{Type: es.TransPop})
				return nil
			}).
			WithChoice("凍えて進む（糧食 -12）", func(world w.World) error {
				if run := query.GetCaravanRun(world); run != nil {
					run.Supply.Food = clampMin0(run.Supply.Food - 12)
				}
				ms.SetTransition(es.Transition[w.World]{Type: es.TransPop})
				return nil
			})
		return ms, nil
	}
}

// backpackWeight はキャラバンが背嚢に持つ全アイテムの総重量を返す（＝積載）。
func backpackWeight(world w.World) route.Weight {
	total := 0.0
	q := ecs.NewFilter1[gc.LocationInBackpack](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		total += query.GetEntityWeight(world, e) // GetEntityWeight は個数を含む
	}
	return route.Weight(total)
}

// backpackValue はキャラバンが背嚢に持つ全アイテムの価値合計を返す（潜行で稼いだ差分の算出に使う）。
func backpackValue(world w.World) int {
	total := 0
	q := ecs.NewFilter1[gc.LocationInBackpack](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		total += query.GetItemValue(world, e) * query.GetEntityCount(world, e)
	}
	return total
}

func clampMin0(n int) int {
	if n < 0 {
		return 0
	}
	return n
}

func cycle(i, n int) int {
	if n <= 0 {
		return 0
	}
	return ((i % n) + n) % n
}

// Draw は停留点マップを描画する。
func (st *MacroMapState) Draw(world w.World, screen *ebiten.Image) error {
	run := query.GetCaravanRun(world)
	if run == nil || run.Beacons == nil {
		return nil
	}
	st.drawMap(world, screen, run)
	return nil
}

// ================
// 描画（停留点マップ）
// ================

func (st *MacroMapState) drawMap(world w.World, screen *ebiten.Image, run *gc.CaravanRun) {
	m := run.Beacons
	face := world.Resources.UIResources.Text.BodyFace
	sw := float64(screen.Bounds().Dx())
	sh := float64(screen.Bounds().Dy())

	screen.Fill(colorMacroBG)
	vector.FillCircle(screen, float32(sw*0.5), float32(sh*0.5), float32(sw*0.45), colorMacroBGCenter, true)

	const (
		marginX      = 70.0
		marginTop    = 120.0
		marginBottom = 70.0
	)
	mapW := sw - 2*marginX
	mapH := sh - marginTop - marginBottom

	// 列ごとに集めて位置を計算
	maxCol := 0
	colBeacons := map[int][]route.NodeID{}
	for _, b := range m.Beacons {
		if b.Column > maxCol {
			maxCol = b.Column
		}
		colBeacons[b.Column] = append(colBeacons[b.Column], b.ID)
	}
	if maxCol == 0 {
		maxCol = 1
	}
	pos := make(map[route.NodeID][2]float64, len(m.Beacons))
	for col, ids := range colBeacons {
		x := marginX + float64(col)/float64(maxCol)*mapW
		for i, id := range ids {
			y := marginTop + float64(i+1)/float64(len(ids)+1)*mapH
			pos[id] = [2]float64{x, y}
		}
	}

	// 選択中の次停留点
	next := m.Outgoing(run.Current)
	selectedTo := route.NodeID(-1)
	if len(next) > 0 {
		selectedTo = next[cycle(st.sel, len(next))]
	}

	// 辺を描画（現在地からの選択肢を強調）
	for _, b := range m.Beacons {
		p1, ok1 := pos[b.ID]
		if !ok1 {
			continue
		}
		for _, to := range m.Outgoing(b.ID) {
			p2, ok2 := pos[to]
			if !ok2 {
				continue
			}
			col := colorMacroEdge
			width := float32(1.8)
			if b.ID == run.Current {
				col = colorMacroReachEdge
				width = 2.5
				if to == selectedTo {
					col = colorMacroSelected
					width = 4.0
				}
			}
			vector.StrokeLine(screen, float32(p1[0]), float32(p1[1]), float32(p2[0]), float32(p2[1]), width, col, true)
		}
	}

	// 停留点を描画
	for _, b := range m.Beacons {
		p := pos[b.ID]
		x, y := float32(p[0]), float32(p[1])
		r := beaconRadius(b.Event)
		isCurrent := b.ID == run.Current

		switch {
		case isCurrent:
			vector.FillCircle(screen, x, y, r+13, colorMacroGlowWhite, true)
		case b.Event == route.NodeGoal:
			vector.FillCircle(screen, x, y, r+11, colorMacroGlowGold, true)
		}
		vector.FillCircle(screen, x, y, r, colorMacroNodeFill, true)

		ring := nodeRingColor(b.Event)
		rw := float32(2.0)
		switch {
		case isCurrent:
			ring, rw = colorMacroCurrent, 3.5
		case b.ID == selectedTo:
			ring, rw = colorMacroSelected, 3.0
		}
		vector.StrokeCircle(screen, x, y, r, rw, ring, true)

		drawBeaconLabel(screen, face, nodeTypeShort(b.Event), p[0], p[1]+float64(r)+4)
	}

	st.drawOverlay(screen, face, run, selectedTo, world.Config.Debug)
}

// drawOverlay は上部パネル（現在地・供給・寒波リード）と下部（選択中イベント・操作ヒント）を描く。
func (st *MacroMapState) drawOverlay(screen *ebiten.Image, face text.Face, run *gc.CaravanRun, selectedTo route.NodeID, debug bool) {
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
	drawText("停留点マップ ── どこへジャンプするか", 18, theme.TextPrimary)
	if cur := run.Beacons.BeaconByID(run.Current); cur != nil {
		drawText(fmt.Sprintf("現在地: %s　　糧食 %d ／ 燃料 %d ／ 積載 %d",
			nodeTypeJP(cur.Event), run.Supply.Food, run.Supply.Fuel, int(run.Supply.Cargo)), 42, theme.TextPrimary)
	}
	lead := fmt.Sprintf("❄ 寒波リード %d 列", run.FrontLead())
	if run.IsStarving() {
		lead += "　⚠飢餓：足が鈍り寒波が加速する"
	}
	drawText(lead, 66, colorMacroCold)

	// 選択中の次停留点のイベント
	if b := run.Beacons.BeaconByID(selectedTo); b != nil {
		drawText(fmt.Sprintf("→ %s：%s", nodeTypeJP(b.Event), eventHint(b.Event)), sh-48, colorMacroSelected)
	}
	hint := "↑↓: 選ぶ　　決定: ジャンプ　　キャンセル: 戻る"
	if debug {
		hint += "　　G: 目標へ(debug)"
	}
	drawText(hint, sh-26, colorMacroLabel)
}

// eventHint は停留点イベントの一言説明を返す。
func eventHint(ev route.NodeType) string {
	switch ev {
	case route.NodeRuin:
		return "潜行して戦利品（寒波が詰める）"
	case route.NodeMarket, route.NodeShop, route.NodeOutpost:
		return "交易・補給"
	case route.NodeCamp:
		return "野営で糧食回復（寒波が詰める）"
	case route.NodeMountain:
		return "吹雪の選択"
	case route.NodeGoal:
		return "遠征達成"
	default:
		return "穏やかな雪原"
	}
}

// beaconRadius は停留点の半径を返す。目標・母港は大きめ。
func beaconRadius(ev route.NodeType) float32 {
	switch ev {
	case route.NodeGoal:
		return 15
	case route.NodeHome:
		return 13
	default:
		return 11
	}
}

// drawBeaconLabel は停留点の表示名を (cx, topY) に中央寄せで描く（背景チップで辺から分離）。
func drawBeaconLabel(screen *ebiten.Image, face text.Face, label string, cx, topY float64) {
	if label == "" {
		return
	}
	const padX, padY = 4.0, 1.0
	lw, lh := text.Measure(label, face, 0)
	lx := cx - lw/2
	vector.FillRect(screen, float32(lx-padX), float32(topY-padY), float32(lw+2*padX), float32(lh+2*padY), colorMacroLabelBG, false)
	op := &text.DrawOptions{}
	op.GeoM.Translate(lx, topY)
	op.ColorScale.ScaleWithColor(colorMacroLabel)
	text.Draw(screen, label, face, op)
}

// マップ描画の色（モックの暗地＋発光リングに寄せる）
var (
	colorMacroBG        = color.RGBA{10, 16, 23, 255}
	colorMacroBGCenter  = color.RGBA{16, 24, 34, 130}
	colorMacroNodeFill  = color.RGBA{17, 24, 33, 255}
	colorMacroCurrent   = color.RGBA{245, 245, 245, 255}
	colorMacroSelected  = color.RGBA{229, 198, 117, 255}
	colorMacroLabel     = color.RGBA{220, 231, 240, 255}
	colorMacroLabelBG   = color.RGBA{10, 16, 24, 225}
	colorMacroCold      = color.RGBA{127, 214, 255, 255}
	colorMacroEdge      = color.RGBA{70, 84, 95, 200}
	colorMacroReachEdge = color.RGBA{150, 165, 180, 235}
	colorMacroGlowGold  = color.RGBA{255, 211, 92, 55}
	colorMacroGlowWhite = color.RGBA{240, 244, 250, 60}
)

// nodeRingColor は停留点イベントの明色リング色を返す。
func nodeRingColor(t route.NodeType) color.Color {
	switch t {
	case route.NodeHome:
		return color.RGBA{229, 198, 117, 255}
	case route.NodeMarket:
		return color.RGBA{95, 208, 255, 255}
	case route.NodeShop:
		return color.RGBA{201, 160, 255, 255}
	case route.NodeRuin:
		return color.RGBA{255, 138, 95, 255}
	case route.NodePlain:
		return color.RGBA{143, 209, 79, 255}
	case route.NodeMountain:
		return color.RGBA{143, 176, 214, 255}
	case route.NodeCamp:
		return color.RGBA{255, 157, 60, 255}
	case route.NodeOutpost:
		return color.RGBA{127, 214, 255, 255}
	case route.NodeGoal:
		return color.RGBA{255, 211, 92, 255}
	default:
		return color.RGBA{160, 160, 160, 255}
	}
}

// nodeTypeJP は停留点イベントの表示名を返す。
func nodeTypeJP(t route.NodeType) string {
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
		return "野営地"
	case route.NodePlain:
		return "雪原"
	case route.NodeMountain:
		return "吹雪の峠"
	case route.NodeOutpost:
		return "前哨"
	case route.NodeGoal:
		return "目標地点"
	default:
		return "地点"
	}
}

// nodeTypeShort は停留点ラベル用の短い表示名を返す。
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
	case route.NodePlain:
		return "雪原"
	case route.NodeMountain:
		return "吹雪"
	case route.NodeOutpost:
		return "前哨"
	case route.NodeGoal:
		return "目標"
	default:
		return ""
	}
}
