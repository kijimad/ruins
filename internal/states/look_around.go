package states

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	gs "github.com/kijimaD/ruins/internal/systems"
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

// StateConfig はこのステートの設定を返す
func (st *LookAroundState) StateConfig() StateConfig {
	return StateConfig{BlurBackground: false}
}

func (st LookAroundState) String() string {
	return "LookAround"
}

var _ es.State[w.World] = &LookAroundState{}
var _ Configurable = &LookAroundState{}

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
		return fmt.Errorf("プレイヤーがGridElementを持っていません")
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

	if action, ok := st.handleInput(); ok {
		return st.doAction(world, action)
	}

	return st.ConsumeTransition(), nil
}

// handleInput はキー入力をActionIDに変換する
func (st *LookAroundState) handleInput() (inputmapper.ActionID, bool) {
	keyboardInput := input.GetSharedKeyboardInput()

	if keyboardInput.IsKeyJustPressed(ebiten.KeyEscape) || keyboardInput.IsKeyJustPressed(ebiten.KeyX) {
		return inputmapper.ActionCloseMenu, true
	}
	if keyboardInput.IsKeyJustPressed(ebiten.KeyUp) || keyboardInput.IsKeyJustPressed(ebiten.KeyW) {
		return inputmapper.ActionMoveNorth, true
	}
	if keyboardInput.IsKeyJustPressed(ebiten.KeyDown) || keyboardInput.IsKeyJustPressed(ebiten.KeyS) {
		return inputmapper.ActionMoveSouth, true
	}
	if keyboardInput.IsKeyJustPressed(ebiten.KeyLeft) || keyboardInput.IsKeyJustPressed(ebiten.KeyA) {
		return inputmapper.ActionMoveWest, true
	}
	if keyboardInput.IsKeyJustPressed(ebiten.KeyRight) || keyboardInput.IsKeyJustPressed(ebiten.KeyD) {
		return inputmapper.ActionMoveEast, true
	}

	return "", false
}

// doAction はActionIDを実行する
func (st *LookAroundState) doAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionMoveNorth:
		st.moveCursor(world, 0, -1)
	case inputmapper.ActionMoveSouth:
		st.moveCursor(world, 0, 1)
	case inputmapper.ActionMoveWest:
		st.moveCursor(world, -1, 0)
	case inputmapper.ActionMoveEast:
		st.moveCursor(world, 1, 0)
	default:
		return es.Transition[w.World]{}, fmt.Errorf("未対応のアクション: %s", action)
	}

	return st.ConsumeTransition(), nil
}

// moveCursor はカーソルを移動する
func (st *LookAroundState) moveCursor(world w.World, dx, dy int) {
	newX := int(st.cursor.X) + dx
	newY := int(st.cursor.Y) + dy

	level := query.GetDungeon(world).Level
	if newX >= 0 && newX < int(level.TileWidth) && newY >= 0 && newY < int(level.TileHeight) {
		st.cursor.X = consts.Tile(newX)
		st.cursor.Y = consts.Tile(newY)
	}
}

// Draw はステートの描画処理
func (st *LookAroundState) Draw(world w.World, screen *ebiten.Image) error {
	// カーソルを描画
	st.drawCursor(world, screen)

	// タイル情報パネルを描画
	return st.drawInfoPanel(world, screen)
}

// 画像キャッシュ
var (
	cursorImageCache *ebiten.Image
	panelImageCache  *ebiten.Image
)

// drawCursor はカーソルを描画する
func (st *LookAroundState) drawCursor(world w.World, screen *ebiten.Image) {
	tileSize := int(consts.TileSize)
	cursorPixelX := float64(int(st.cursor.X) * tileSize)
	cursorPixelY := float64(int(st.cursor.Y) * tileSize)

	// カーソル画像をキャッシュから取得または作成
	if cursorImageCache == nil {
		cursorImageCache = ebiten.NewImage(tileSize, tileSize)
		// 枠線を描画（太さ3px、白色で目立つように）
		cursorColor := theme.CursorLook
		for i := range 3 {
			// 上辺
			for x := range tileSize {
				cursorImageCache.Set(x, i, cursorColor)
			}
			// 下辺
			for x := range tileSize {
				cursorImageCache.Set(x, tileSize-1-i, cursorColor)
			}
			// 左辺
			for y := range tileSize {
				cursorImageCache.Set(i, y, cursorColor)
			}
			// 右辺
			for y := range tileSize {
				cursorImageCache.Set(tileSize-1-i, y, cursorColor)
			}
		}
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(cursorPixelX, cursorPixelY)
	gs.SetTranslate(world, op)

	// 点滅エフェクト: アルファ値を変化させる。アニメーション無効時は固定値
	if !world.Config.DisableAnimation {
		alpha := 0.6 + 0.4*math.Sin(float64(st.blinkCounter)*0.15)
		op.ColorScale.ScaleAlpha(float32(alpha))
	}

	screen.DrawImage(cursorImageCache, op)
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

	// パネル背景をキャッシュから取得または生成
	if panelImageCache == nil {
		panelImageCache = ebiten.NewImage(panelWidth, panelHeight)
		panelImageCache.Fill(theme.Overlay)
	}

	panelX := screen.Bounds().Dx() - panelWidth - marginX
	panelY := marginY
	panelOp := &ebiten.DrawImageOptions{}
	panelOp.GeoM.Translate(float64(panelX), float64(panelY))
	screen.DrawImage(panelImageCache, panelOp)

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
	drawText(fmt.Sprintf("座標: (%d, %d)", st.cursor.X, st.cursor.Y))
	y += 5

	// 視界内かどうかをチェック
	playerEntity, err := query.GetPlayerEntity(world)
	if err != nil {
		return err
	}
	playerGrid := world.Components.GridElement.Get(playerEntity)
	inVision := query.IsInVision(world, int(playerGrid.X), int(playerGrid.Y), int(st.cursor.X), int(st.cursor.Y))

	if !inVision {
		drawText("暗闇")
		return nil
	}

	// タイル上のエンティティを取得
	entities := query.GetEntitiesAt(world, st.cursor.X, st.cursor.Y)

	if len(entities) == 0 {
		drawText("何もありません")
	} else {
		for _, entity := range entities {
			st.drawEntityInfo(world, entity, drawText)
		}
	}

	// 移動コストを表示
	st.drawPassCost(world, entities, &y, drawText)

	// タイル温度を表示（TileTemperatureコンポーネントを持つエンティティ）
	st.drawTileTemperature(world, entities, &y, drawText)

	// 操作説明
	y = panelY + panelHeight - 30
	drawText("WASD/矢印: 移動  X/Esc: 閉じる")

	return nil
}

// drawEntityInfo はエンティティ情報を描画する
func (st *LookAroundState) drawEntityInfo(world w.World, entity ecs.Entity, drawText func(string)) {
	name := query.GetEntityName(entity, world)

	cat, ok := world.Components.CategoryOf(gc.FieldLookCategoryKey, entity)
	if !ok {
		// 壁などは名前だけ表示する
		if name != "" {
			drawText(name)
		}
		return
	}

	typeStr := fmt.Sprintf("[%s]", cat)
	if name != "" {
		drawText(fmt.Sprintf("%s %s", typeStr, name))
	} else {
		drawText(typeStr)
	}

	// HPを持つエンティティはHP表示
	if world.Components.HP.Has(entity) {
		hp := world.Components.HP.Get(entity)
		label := "HP"
		if world.Components.Prop.Has(entity) {
			label = "耐久"
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
		drawText("移動コスト: 不可")
	} else {
		cost := consts.StandardActionCost + totalAdd
		drawText(fmt.Sprintf("移動コスト: %d", cost))
	}
}

// drawTileTemperature はタイル温度修正値を描画する
func (st *LookAroundState) drawTileTemperature(world w.World, entities []ecs.Entity, y *int, drawText func(string)) {
	for _, entity := range entities {
		if world.Components.TileTemperature.Has(entity) {
			temp := world.Components.TileTemperature.Get(entity)
			*y += 5
			drawText(fmt.Sprintf("気温修正: %+d", temp.Total()))
			if temp.Shelter != 0 {
				drawText(fmt.Sprintf("  屋内: %+d", temp.Shelter))
			}
			if temp.Water != 0 {
				drawText(fmt.Sprintf("  水辺: %+d", temp.Water))
			}
			if temp.Foliage != 0 {
				drawText(fmt.Sprintf("  植生: %+d", temp.Foliage))
			}
			return
		}
	}
}
