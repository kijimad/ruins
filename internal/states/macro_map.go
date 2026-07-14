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
	"github.com/kijimaD/ruins/internal/hooks"
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
const macroMapDebugSeed = 20260713

// MacroMapState はマクロ移動（ルート網のノード選択）画面のステート。
// ルート網を層状に描画し、進める辺を選んで踏破する。供給・寒波リードをオーバーレイ表示する。
type MacroMapState struct {
	es.BaseState[w.World]
	mount *hooks.Mount[macroMapProps]
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
	// 正式な遠征選択入口（Phase 5）ではそちらがプレイヤー・ランを用意するため、ここは通らない。
	if query.GetCaravanRun(world) == nil {
		if err := setupDebugRun(world); err != nil {
			return err
		}
	}
	st.mount = hooks.NewMount[macroMapProps]()
	return nil
}

// setupDebugRun はデバッグ入口用に最小の実ラン世界（プレイヤー・隊員・ラン）を用意する。
// Shop/Dungeon などプレイヤー依存のサブステートを MacroMap から起動できるようにするための暫定処理で、
// 正式な遠征選択入口（Phase 5）ができたら不要になる。DemoStart のセットアップに倣う。
func setupDebugRun(world w.World) error {
	if err := ensureDebugParty(world); err != nil {
		return err
	}
	query.SetCaravanRun(world, gc.NewCaravanRun(macroMapDebugSeed, route.ExpeditionDeepVault))
	return nil
}

// ensureDebugParty はプレイヤー・隊員が未生成なら生成する（デバッグ入口用。正式には母港で用意済み）。
// ラン終了で CaravanRun を除去してもプレイヤー/隊員は残るため、CaravanRun でなくプレイヤー不在で判定する
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

// Update はステートの更新処理
func (st *MacroMapState) Update(world w.World) (es.Transition[w.World], error) {
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
	hooks.UseTabMenu(st.mount.Store(), "macromap", hooks.TabMenuConfig{
		TabCount:   1,
		ItemCounts: []int{len(props.Items)},
	})
	st.mount.Update()

	return st.ConsumeTransition(), nil
}

// Draw はルート網を層状に描画する。背景は塗らず後ろの呼び出し元を見せる。
func (st *MacroMapState) Draw(world w.World, screen *ebiten.Image) error {
	run := query.GetCaravanRun(world)
	if run == nil || run.Graph == nil {
		return nil
	}
	st.drawMap(world, screen, run)
	return nil
}

// DoAction はActionを実行する
func (st *MacroMapState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionMenuSelect:
		return st.handleSelection(world)
	case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown,
		inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight,
		inputmapper.ActionMenuTabNext, inputmapper.ActionMenuTabPrev:
		return es.Transition[w.World]{Type: es.TransNone}, nil
	default:
		return es.Transition[w.World]{}, fmt.Errorf("macroMap: 未対応のアクション: %s", action)
	}
}

// ================
// Props
// ================

type macroMapProps struct {
	Items []macroMapItem
}

// macroMapItem は進める辺の1項目。表示要素は連結せず、役割ごとに別フィールドで持つ
// （インベントリメニューの inventoryItemData に倣う）。整形・色分け・列揃えは描画側の関心。
type macroMapItem struct {
	Dest     string     // 行き先ノードの表示名
	EdgeKind string     // 辺種別の表示名
	Faces    int        // 踏破に要する面数
	Edge     route.Edge // 選択時に踏破する辺の実体
	IsCancel bool       // 「戻る」項目
}

func (st *MacroMapState) fetchProps(world w.World) macroMapProps {
	run := query.GetCaravanRun(world)
	if run == nil || run.Graph == nil {
		return macroMapProps{Items: []macroMapItem{{IsCancel: true}}}
	}

	outgoing := run.Graph.Outgoing(run.Current)
	items := make([]macroMapItem, 0, len(outgoing)+1)
	for _, e := range outgoing {
		to := run.Graph.NodeByID(e.To)
		if to == nil {
			continue // 生成不整合等でエッジ先を引けない辺はスキップ
		}
		items = append(items, macroMapItem{
			Dest:     nodeTypeJP(to.Type),
			EdgeKind: edgeTypeJP(e.Type),
			Faces:    e.Faces,
			Edge:     e,
		})
	}
	items = append(items, macroMapItem{IsCancel: true})

	return macroMapProps{Items: items}
}

func (st *MacroMapState) handleSelection(world w.World) (es.Transition[w.World], error) {
	menuState, ok := hooks.GetState[hooks.TabMenuState](st.mount, "macromap")
	if !ok {
		return es.Transition[w.World]{}, fmt.Errorf("macromapの取得に失敗")
	}
	item := st.mount.GetProps().Items[menuState.ItemIndex]
	if item.IsCancel {
		return es.Transition[w.World]{Type: es.TransPop}, nil
	}

	run := query.GetCaravanRun(world)
	if run == nil {
		return es.Transition[w.World]{Type: es.TransPop}, nil
	}
	to := run.Graph.NodeByID(item.Edge.To)
	if to == nil {
		return es.Transition[w.World]{Type: es.TransPop}, nil
	}
	res := run.AdvanceAlong(item.Edge)
	gamelog.New(query.GetGameLog(world)).System(fmt.Sprintf(
		"%sへ移動した。糧食-%d 燃料-%d、寒波が接近する。",
		nodeTypeJP(to.Type), res.Cost.Food, res.Cost.Fuel)).Log()
	if run.IsStarving() {
		gamelog.New(query.GetGameLog(world)).System("糧食が尽きた。飢えで足が鈍り、寒波が余分に詰める。").Log()
	}
	if run.Swallowed() {
		return st.failSwallowed(world)
	}

	return st.dispatchNode(world, run, to)
}

// dispatchNode は到達ノードの型に応じたサブ挙動へ振り分ける（Phase 4）。
func (st *MacroMapState) dispatchNode(world w.World, run *gc.CaravanRun, node *route.Node) (es.Transition[w.World], error) {
	switch node.Type {
	case route.NodeGoal:
		gamelog.New(query.GetGameLog(world)).System("目標地点に到達した。目標物を納品して遠征達成。").Log()
		summary := goalSummary(world)
		query.SetCaravanRun(world, nil) // ランを終了（再入時は新規生成）
		return es.Transition[w.World]{
			Type:          es.TransSwitch,
			NewStateFuncs: []es.StateFactory[w.World]{NewGoalResultState(summary)},
		}, nil
	case route.NodeCamp:
		// 野営で糧食を回復するが、休息の間に寒波前線が詰める（道草の代償）
		const campFoodRestore = 15
		run.Supply.Food += campFoodRestore
		run.Dawdle(gc.CampFrontCost)
		gamelog.New(query.GetGameLog(world)).System(fmt.Sprintf(
			"野営した。糧食を %d 回復したが、寒波が %d 面詰めた。", campFoodRestore, gc.CampFrontCost)).Log()
		if run.Swallowed() {
			return st.failSwallowed(world)
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case route.NodeMarket, route.NodeShop, route.NodeJunction, route.NodeOutpost:
		// 集落マップに入る。商人に話しかけて交易し、帰還ゲートで MacroMap へ戻る（ShopMenu は即出さない）。
		// TODO(Phase後段): 専門店(改造)・合流点(全ルート合流演出)・前哨(最終補給/売却点)を型ごとに差別化する（現状は同じ集落マップ）
		gamelog.New(query.GetGameLog(world)).System(fmt.Sprintf(
			"%sに到着した。集落に入る。", nodeTypeJP(node.Type))).Log()
		return es.Transition[w.World]{
			Type: es.TransPush,
			NewStateFuncs: []es.StateFactory[w.World]{
				NewMarketState(WithEscapePop()),
			},
		}, nil
	case route.NodeRuin:
		// 潜行する間に寒波前線が詰める（引き際の核）。呑まれたら潜らずラン失敗。
		// 脱出時は自動精算を通さず、荷を持ったまま MacroMap へ Pop で戻す（WithEscapePop）
		run.Dawdle(gc.RuinFrontCost)
		if run.Swallowed() {
			return st.failSwallowed(world)
		}
		gamelog.New(query.GetGameLog(world)).System(fmt.Sprintf(
			"遺跡に到着した。潜行する（寒波が %d 面詰める）。", gc.RuinFrontCost)).Log()
		return es.Transition[w.World]{
			Type: es.TransPush,
			NewStateFuncs: []es.StateFactory[w.World]{
				NewRuinState(WithEscapePop()),
			},
		}, nil
	case route.NodeHome:
		return es.Transition[w.World]{Type: es.TransNone}, nil
	}
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

// failSwallowed は寒波前線に呑まれてラン失敗した際の処理。ランを終了して道中を閉じる。
func (st *MacroMapState) failSwallowed(world w.World) (es.Transition[w.World], error) {
	gamelog.New(query.GetGameLog(world)).System("寒波前線に呑まれた。ラン失敗。").Log()
	query.SetCaravanRun(world, nil) // ランを終了（再入時は新規生成）
	return es.Transition[w.World]{Type: es.TransPop}, nil
}

// goalSummary は遠征達成時のスコア要約を作る（富＝所持通貨＋生存者。希少財の加点は後段）。
func goalSummary(world w.World) string {
	currency := 0
	if pe, err := query.GetPlayerEntity(world); err == nil && world.Components.Wallet.Has(pe) {
		currency = query.GetCurrency(world, pe)
	}
	survivors := query.SquadMemberCount(world) + 1 // ＋リーダー
	const survivorBonus = 50
	score := currency + survivors*survivorBonus
	return fmt.Sprintf("遠征達成！\n\n富（通貨）: %d\n生存者: %d 人\nスコア: %d\n\n（希少財の加点は後段）",
		currency, survivors, score)
}

// ================
// 描画（ルート網）
// ================

func (st *MacroMapState) drawMap(world w.World, screen *ebiten.Image, run *gc.CaravanRun) {
	g := run.Graph
	face := world.Resources.UIResources.Text.BodyFace
	sw := float64(screen.Bounds().Dx())
	sh := float64(screen.Bounds().Dy())

	const (
		marginX      = 80.0
		marginTop    = 120.0
		marginBottom = 80.0
	)
	mapW := sw - 2*marginX
	mapH := sh - marginTop - marginBottom

	// 層ごとにノードを集め、位置を計算する（層＝列、層内は縦に等間隔）
	maxLayer := 0
	layerNodes := map[int][]route.NodeID{}
	for _, n := range g.Nodes {
		if n.Layer > maxLayer {
			maxLayer = n.Layer
		}
		layerNodes[n.Layer] = append(layerNodes[n.Layer], n.ID)
	}
	if maxLayer == 0 {
		maxLayer = 1
	}
	pos := make(map[route.NodeID][2]float64, len(g.Nodes))
	for layer, ids := range layerNodes {
		x := marginX + float64(layer)/float64(maxLayer)*mapW
		for i, id := range ids {
			y := marginTop + float64(i+1)/float64(len(ids)+1)*mapH
			pos[id] = [2]float64{x, y}
		}
	}

	// 選択中の辺・進める先
	selectedTo := route.NodeID(-1)
	if ms, ok := hooks.GetState[hooks.TabMenuState](st.mount, "macromap"); ok {
		items := st.mount.GetProps().Items
		if ms.ItemIndex >= 0 && ms.ItemIndex < len(items) && !items[ms.ItemIndex].IsCancel {
			selectedTo = items[ms.ItemIndex].Edge.To
		}
	}
	reachable := map[route.NodeID]bool{}
	for _, e := range g.Outgoing(run.Current) {
		reachable[e.To] = true
	}

	// 辺を描画（現在ノードから進める辺・選択中の辺を強調）
	for _, e := range g.Edges {
		p1, ok1 := pos[e.From]
		p2, ok2 := pos[e.To]
		if !ok1 || !ok2 {
			continue
		}
		width := float32(1.5)
		if e.From == run.Current {
			width = 2.5
			if e.To == selectedTo {
				width = 4.5
			}
		}
		vector.StrokeLine(screen, float32(p1[0]), float32(p1[1]), float32(p2[0]), float32(p2[1]), width, edgeColor(e.Type), true)
	}

	// ノードを描画
	for _, n := range g.Nodes {
		p := pos[n.ID]
		x, y := float32(p[0]), float32(p[1])
		vector.FillCircle(screen, x, y, 11, nodeColor(n.Type), true)
		switch {
		case n.ID == run.Current:
			vector.StrokeCircle(screen, x, y, 15, 3, colorMacroCurrent, true)
		case n.ID == selectedTo:
			vector.StrokeCircle(screen, x, y, 14, 2.5, colorMacroSelected, true)
		case reachable[n.ID]:
			vector.StrokeCircle(screen, x, y, 13, 1.5, colorMacroReachable, true)
		}
		drawNodeLabel(screen, face, nodeTypeShort(n.Type), p[0], p[1])
	}

	st.drawOverlay(screen, face, run)
}

// drawOverlay は上部に現在地・供給・寒波リード、下部に選択中の辺と操作ヒントを描く。
func (st *MacroMapState) drawOverlay(screen *ebiten.Image, face text.Face, run *gc.CaravanRun) {
	sw := screen.Bounds().Dx()
	sh := screen.Bounds().Dy()

	const textX = 22.0
	drawText := func(s string, y int, c color.Color) {
		op := &text.DrawOptions{}
		op.GeoM.Translate(textX, float64(y))
		op.ColorScale.ScaleWithColor(c)
		text.Draw(screen, s, face, op)
	}

	// 上部パネル
	styled.DrawFramedBackground(screen, 10, 10, sw-20, 92, styled.PanelStyle())
	drawText("ルート網 ── どこを経由するか", 18, theme.TextPrimary)
	if cur := run.Graph.NodeByID(run.Current); cur != nil {
		drawText(fmt.Sprintf("現在地: %s　　糧食 %d ／ 燃料 %d ／ 積載 %d",
			nodeTypeJP(cur.Type), run.Supply.Food, run.Supply.Fuel, int(run.Supply.Cargo)), 42, theme.TextPrimary)
	}
	lead := fmt.Sprintf("❄ 寒波リード %d 面", run.FrontLead())
	if run.IsStarving() {
		lead += "　⚠飢餓：足が鈍り寒波が加速する"
	}
	drawText(lead, 66, colorMacroCold)

	// 下部: 選択中の辺（要素ごとに列で並べる。文字列連結しない）と操作ヒント
	if ms, ok := hooks.GetState[hooks.TabMenuState](st.mount, "macromap"); ok {
		items := st.mount.GetProps().Items
		if ms.ItemIndex >= 0 && ms.ItemIndex < len(items) {
			drawItemRow(screen, face, textX, float64(sh-52), items[ms.ItemIndex])
		}
	}
	drawText("↑↓/←→: 選ぶ　　決定: 進む　　キャンセル: 戻る", sh-28, colorMacroLabel)
}

// drawNodeLabel はノードの表示名を円の直下に中央寄せで描く。
// 辺と重ならないよう背景チップを敷き、リング半径ぶん下げて分離する。
func drawNodeLabel(screen *ebiten.Image, face text.Face, label string, cx, cy float64) {
	const (
		gapBelow = 18.0 // 現在ノードのリング半径(15)より下げてラベルを分離する
		padX     = 4.0
		padY     = 1.0
	)
	lw, lh := text.Measure(label, face, 0)
	lx := cx - lw/2
	ly := cy + gapBelow
	vector.FillRect(screen,
		float32(lx-padX), float32(ly-padY), float32(lw+2*padX), float32(lh+2*padY),
		colorMacroLabelBG, false)
	op := &text.DrawOptions{}
	op.GeoM.Translate(lx, ly)
	op.ColorScale.ScaleWithColor(colorMacroLabel)
	text.Draw(screen, label, face, op)
}

// textSegment は横並びで描く1区画のテキストと色。
type textSegment struct {
	text  string
	color color.Color
}

// drawTextRow はセグメントを左から順に、実測幅ぶん送りながら描く（列レイアウト）。
func drawTextRow(screen *ebiten.Image, face text.Face, x, y, gap float64, segs []textSegment) {
	for _, s := range segs {
		if s.text == "" {
			continue
		}
		op := &text.DrawOptions{}
		op.GeoM.Translate(x, y)
		op.ColorScale.ScaleWithColor(s.color)
		text.Draw(screen, s.text, face, op)
		wSeg, _ := text.Measure(s.text, face, 0)
		x += wSeg + gap
	}
}

// drawItemRow は選択中の辺を「行き先／辺種別／面数」の列に分けて描く。
// 表示要素は連結せず、フィールドごとに別セグメントで色分けする（インベントリ流）。
func drawItemRow(screen *ebiten.Image, face text.Face, x, y float64, item macroMapItem) {
	if item.IsCancel {
		drawTextRow(screen, face, x, y, 0, []textSegment{{"戻る", colorMacroSelected}})
		return
	}
	drawTextRow(screen, face, x, y, 14, []textSegment{
		{"→ " + item.Dest, colorMacroSelected},
		{item.EdgeKind, colorMacroLabel},
		{fmt.Sprintf("面 %d", item.Faces), colorMacroLabel},
	})
}

// マップ描画の色
var (
	colorMacroCurrent   = color.RGBA{245, 245, 245, 255}
	colorMacroSelected  = color.RGBA{224, 190, 110, 255}
	colorMacroReachable = color.RGBA{150, 165, 180, 255}
	colorMacroLabel     = color.RGBA{190, 205, 220, 255}
	colorMacroLabelBG   = color.RGBA{18, 24, 32, 215} // ラベルを辺から分離する背景チップ
	colorMacroCold      = color.RGBA{120, 190, 230, 255}
)

func nodeColor(t route.NodeType) color.Color {
	switch t {
	case route.NodeHome:
		return color.RGBA{230, 230, 230, 255}
	case route.NodeMarket:
		return color.RGBA{212, 175, 95, 255}
	case route.NodeShop:
		return color.RGBA{110, 160, 210, 255}
	case route.NodeRuin:
		return color.RGBA{200, 90, 90, 255}
	case route.NodeCamp:
		return color.RGBA{110, 180, 120, 255}
	case route.NodeOutpost:
		return color.RGBA{110, 200, 200, 255}
	case route.NodeJunction:
		return color.RGBA{220, 150, 80, 255}
	case route.NodeGoal:
		return color.RGBA{245, 220, 120, 255}
	default:
		return color.RGBA{160, 160, 160, 255}
	}
}

func edgeColor(t route.EdgeType) color.Color {
	switch t {
	case route.EdgeShortcut:
		return color.RGBA{100, 170, 220, 220} // 凍える近道
	case route.EdgeDetour:
		return color.RGBA{200, 165, 90, 220} // 暖かい迂回
	case route.EdgeDanger:
		return color.RGBA{200, 90, 90, 220} // 危険路
	default:
		return color.RGBA{90, 105, 120, 220} // 本道
	}
}

// nodeTypeJP はノード種別の表示名を返す（表示層の関心。route はモデルを英字で持つ）。
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
	case route.NodeOutpost:
		return "前哨"
	case route.NodeJunction:
		return "隊商宿（合流）"
	case route.NodeGoal:
		return "目標地点"
	default:
		return "地点"
	}
}

// nodeTypeShort はマップ上のノードラベル用の短い表示名を返す。
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
	case route.NodeJunction:
		return "隊商宿"
	case route.NodeGoal:
		return "目標"
	default:
		return "地点"
	}
}

// edgeTypeJP は辺種別の表示名を返す。
func edgeTypeJP(t route.EdgeType) string {
	switch t {
	case route.EdgeShortcut:
		return "凍える近道"
	case route.EdgeDetour:
		return "暖かい迂回"
	case route.EdgeDanger:
		return "危険路"
	default:
		return "本道"
	}
}
