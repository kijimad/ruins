package states

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
	"github.com/kijimaD/ruins/internal/render3d"
	"github.com/kijimaD/ruins/internal/widgets/framedbg"
	"github.com/kijimaD/ruins/internal/widgets/hud"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// LookAroundState はタイル情報確認モードのステート
// カーソルをマップ上で動かしてタイル・エンティティ情報を確認できる
type LookAroundState struct {
	es.BaseState[w.World]
	cursor       consts.Coord[consts.Tile]
	blinkCounter int
}

var _ es.State[w.World] = &LookAroundState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *LookAroundState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *LookAroundState) OnResume(_ w.World) error { return nil }

// OnStart はステートが開始される際に呼ばれる
func (st *LookAroundState) OnStart(world w.World) error {
	// プレイヤー位置からカーソルを開始
	playerEntity, err := query.GetPlayerEntity(world)
	if err != nil {
		return err
	}

	if !world.Components.GridElement.Has(playerEntity) {
		return fmt.Errorf("player does not have GridElement")
	}

	playerGrid := world.Components.GridElement.Get(playerEntity)
	st.cursor.X = playerGrid.X
	st.cursor.Y = playerGrid.Y

	return nil
}

// OnStop はステートが終了する際に呼ばれる
func (st *LookAroundState) OnStop(_ w.World) error { return nil }

// Update はステートの更新処理
func (st *LookAroundState) Update(world w.World) (es.Transition[w.World], error) {
	st.blinkCounter++

	if action, ok := keybind.ReadInput(world, lookAroundBindings); ok {
		return st.doAction(world, action)
	}

	return st.ConsumeTransition(), nil
}

// lookAroundBindings は見回しモードの束縛表。矢印でカーソルを動かし、Esc で閉じる
var lookAroundBindings = []keybind.Binding{
	{Key: ebiten.KeyEscape, Action: inputmapper.ActionCloseMenu},
	{Key: ebiten.KeyUp, Action: inputmapper.ActionMoveNorth},
	{Key: ebiten.KeyDown, Action: inputmapper.ActionMoveSouth},
	{Key: ebiten.KeyLeft, Action: inputmapper.ActionMoveWest},
	{Key: ebiten.KeyRight, Action: inputmapper.ActionMoveEast},
}

// doAction はActionIDを実行する
func (st *LookAroundState) doAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionMoveNorth:
		st.moveCursor(world, gc.DirectionUp)
	case inputmapper.ActionMoveSouth:
		st.moveCursor(world, gc.DirectionDown)
	case inputmapper.ActionMoveWest:
		st.moveCursor(world, gc.DirectionLeft)
	case inputmapper.ActionMoveEast:
		st.moveCursor(world, gc.DirectionRight)
	default:
		return es.Transition[w.World]{}, fmt.Errorf("unsupported action: %s", action)
	}

	return st.ConsumeTransition(), nil
}

// moveCursor はカーソルを移動する。押した向きはカメラの水平角で回し、画面の上下左右に合わせる。
// カメラが回転していても、右キーで画面の右にあるタイルへ動く。
func (st *LookAroundState) moveCursor(world w.World, base gc.Direction) {
	var yaw float64
	if camera := query.GetPlayerCamera(world); camera != nil {
		yaw = camera.Yaw()
	}
	next := st.cursor.Add(gc.RotateScreenDir(base, yaw).GetDelta())

	field := query.GetCurrentStageField(world)
	if field == nil {
		return
	}
	level := field.Level
	if next.X >= 0 && next.X < level.TileWidth && next.Y >= 0 && next.Y < level.TileHeight {
		st.cursor = next
	}
}

// Draw はステートの描画処理
func (st *LookAroundState) Draw(world w.World, screen *ebiten.Image) error {
	// カーソルを描画
	if err := st.drawCursor(world, screen); err != nil {
		return err
	}

	// タイル情報パネルを描画
	return st.drawInfoPanel(world, screen)
}

// cursorFrameWidth はカーソル枠の線の太さ
const cursorFrameWidth = 3

// drawCursor はカーソルを描画する。
// タイルは透視投影で台形になるので、投影した四隅を線で結んで実際の輪郭に合わせる。
func (st *LookAroundState) drawCursor(world w.World, screen *ebiten.Image) error {
	projector, err := render3d.WorldProjector(world)
	if err != nil {
		return err
	}
	corners, ok := projector.TileCorners(st.cursor, render3d.TileTopHeight(world, st.cursor))
	if !ok {
		return nil
	}

	cursorColor := theme.CursorLook
	// 点滅エフェクト: アルファ値を変化させる。アニメーション無効時は固定値
	if !world.Resources.Config.DisableAnimation {
		cursorColor = hud.ScaleAlpha(cursorColor, 0.6+0.4*math.Sin(float64(st.blinkCounter)*0.15))
	}
	hud.TileFrame(screen, corners, cursorFrameWidth, cursorColor)
	return nil
}

// drawInfoPanel はタイル情報パネルを描画する
func (st *LookAroundState) drawInfoPanel(world w.World, screen *ebiten.Image) error {
	face := world.Resources.UIResources.Text.BodyFace

	const (
		panelWidth  = 300
		panelHeight = 200
		marginX     = 10
		marginY     = 10
		lineHeight  = 20
	)

	panelX := screen.Bounds().Dx() - panelWidth - marginX
	panelY := marginY
	// パネル背景をメニュー枠と同じ共通 chrome に揃える
	framedbg.Draw(screen, panelX, panelY, panelWidth, panelHeight, framedbg.PanelStyle())

	// テキスト描画ヘルパー
	textX := float64(panelX + 10)
	y := panelY + 10

	drawText := func(str string) {
		op := &text.DrawOptions{}
		op.GeoM.Translate(textX, float64(y))
		text.Draw(screen, str, face, op)
		y += lineHeight
	}

	// 座標表示
	drawText(fmt.Sprintf("%s: %s", query.T(world, "Coord"), st.cursor))
	y += 5

	// 視界内かどうかをチェック
	playerEntity, err := query.GetPlayerEntity(world)
	if err != nil {
		return err
	}
	playerGrid := world.Components.GridElement.Get(playerEntity)
	inVision := query.IsInVision(world, playerGrid.Coord, st.cursor)

	if !inVision {
		drawText(query.T(world, "Darkness"))
		return nil
	}

	// タイル上のエンティティを取得
	entities := query.GetEntitiesAt(world, st.cursor.X, st.cursor.Y)

	if len(entities) == 0 {
		drawText(query.T(world, "Nothing here"))
	} else {
		// 床の同一スタックは1行に束ね、拾得メニューと見え方を揃える
		for _, stack := range query.GroupStacks(world, entities) {
			st.drawEntityInfo(world, stack.Rep, stack.Count, drawText)
		}
	}

	// 移動コストを表示
	st.drawPassCost(world, entities, &y, drawText)

	// タイル温度を表示（TileTemperatureコンポーネントを持つエンティティ）
	st.drawTileTemperature(world, entities, &y, drawText)

	// 操作説明
	y = panelY + panelHeight - 30
	drawText(query.T(world, "Arrows: Move  X/Esc: Close"))

	return nil
}

// drawEntityInfo はエンティティ情報を描画する。個数は束ねた結果を呼び出し側が渡す
func (st *LookAroundState) drawEntityInfo(world w.World, entity ecs.Entity, count int, drawText func(string)) {
	name := query.FormatNameCount(query.GetEntityName(entity, world), count)

	cat, ok := world.Components.CategoryOf(gc.FieldLookCategoryKey, entity)
	if !ok {
		// 壁などは名前だけ表示する
		if name != "" {
			drawText(name)
		}
		return
	}

	typeStr := fmt.Sprintf("[%s]", query.T(world, cat))
	if name != "" {
		drawText(fmt.Sprintf("%s %s", typeStr, name))
	} else {
		drawText(typeStr)
	}

	// HPを持つエンティティはHP表示
	if world.Components.HP.Has(entity) {
		hp := world.Components.HP.Get(entity)
		label := "HP"
		if world.Components.Fixed.Has(entity) {
			label = query.T(world, "Durability")
		}
		drawText(fmt.Sprintf("  %s: %d/%d", label, hp.Current, hp.Max))
	}
}

// drawPassCost は移動コストを描画する
func (st *LookAroundState) drawPassCost(world w.World, entities []ecs.Entity, y *int, drawText func(string)) {
	blocked := false
	totalAdd := 0
	for _, entity := range entities {
		if world.Components.BlockPass.Has(entity) {
			blocked = true
		}
		if world.Components.PassCost.Has(entity) {
			mc := world.Components.PassCost.Get(entity)
			totalAdd += mc.Value
		}
	}
	*y += 5
	if blocked {
		drawText(query.T(world, "Move cost") + ": " + query.T(world, "Impassable"))
	} else {
		cost := consts.StandardActionCost + totalAdd
		drawText(fmt.Sprintf("%s: %d", query.T(world, "Move cost"), cost))
	}
}

// drawTileTemperature はタイル温度修正値を描画する
func (st *LookAroundState) drawTileTemperature(world w.World, entities []ecs.Entity, y *int, drawText func(string)) {
	for _, entity := range entities {
		if world.Components.TileTemperature.Has(entity) {
			temp := world.Components.TileTemperature.Get(entity)
			*y += 5
			drawText(fmt.Sprintf("%s: %+d", query.T(world, "Temperature modifier"), temp.Total()))
			if temp.Shelter != 0 {
				drawText(fmt.Sprintf("  %s: %+d", query.T(world, "Indoor"), temp.Shelter))
			}
			if temp.Water != 0 {
				drawText(fmt.Sprintf("  %s: %+d", query.T(world, "Waterside"), temp.Water))
			}
			if temp.Foliage != 0 {
				drawText(fmt.Sprintf("  %s: %+d", query.T(world, "Foliage"), temp.Foliage))
			}
			return
		}
	}
}
