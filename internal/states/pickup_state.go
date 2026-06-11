package states

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	gs "github.com/kijimaD/ruins/internal/systems"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// PickupState は拾うモードのモーダルステート。
// WASD/矢印キーでカーソルを隣接8タイルに移動し、Enterで拾得を実行する
type PickupState struct {
	es.BaseState[w.World]
	cursor        consts.Coord[consts.Tile] // カーソル位置（絶対座標）
	playerPos     consts.Coord[consts.Tile] // プレイヤー位置（移動制限用）
	itemsAtTarget []ecs.Entity              // 対象タイル上の拾得可能エンティティ
	blinkCounter  int                       // カーソル点滅用
}

// StateConfig はこのステートの設定を返す
func (st *PickupState) StateConfig() StateConfig {
	return StateConfig{BlurBackground: false}
}

func (st PickupState) String() string {
	return "Pickup"
}

var _ es.State[w.World] = &PickupState{}
var _ Configurable = &PickupState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *PickupState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *PickupState) OnResume(_ w.World) error { return nil }

// OnStart はステートが開始される際に呼ばれる
func (st *PickupState) OnStart(world w.World) error {
	playerEntity, err := worldhelper.GetPlayerEntity(world)
	if err != nil {
		return err
	}
	playerGrid := world.Components.GridElement.Get(playerEntity).(*gc.GridElement)
	st.playerPos = consts.Coord[consts.Tile]{X: playerGrid.X, Y: playerGrid.Y}
	st.cursor = st.playerPos
	st.refreshItems(world)
	return nil
}

// OnStop はステートが終了する際に呼ばれる
func (st *PickupState) OnStop(_ w.World) error { return nil }

// Update はステートの更新処理
func (st *PickupState) Update(world w.World) (es.Transition[w.World], error) {
	st.blinkCounter++

	if action, ok := st.handleInput(); ok {
		return st.doAction(world, action)
	}

	return st.ConsumeTransition(), nil
}

func (st *PickupState) handleInput() (inputmapper.ActionID, bool) {
	keyboardInput := input.GetSharedKeyboardInput()

	if keyboardInput.IsKeyJustPressed(ebiten.KeyEscape) {
		return inputmapper.ActionCloseMenu, true
	}
	if keyboardInput.IsEnterJustPressedOnce() {
		return inputmapper.ActionPickup, true
	}

	// 方向キー
	if keyboardInput.IsKeyJustPressed(ebiten.KeyW) || keyboardInput.IsKeyJustPressed(ebiten.KeyUp) {
		return inputmapper.ActionMoveNorth, true
	}
	if keyboardInput.IsKeyJustPressed(ebiten.KeyS) || keyboardInput.IsKeyJustPressed(ebiten.KeyDown) {
		return inputmapper.ActionMoveSouth, true
	}
	if keyboardInput.IsKeyJustPressed(ebiten.KeyA) || keyboardInput.IsKeyJustPressed(ebiten.KeyLeft) {
		return inputmapper.ActionMoveWest, true
	}
	if keyboardInput.IsKeyJustPressed(ebiten.KeyD) || keyboardInput.IsKeyJustPressed(ebiten.KeyRight) {
		return inputmapper.ActionMoveEast, true
	}

	return "", false
}

func (st *PickupState) doAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil

	case inputmapper.ActionMoveNorth:
		st.moveCursor(0, -1)
		st.refreshItems(world)
	case inputmapper.ActionMoveSouth:
		st.moveCursor(0, 1)
		st.refreshItems(world)
	case inputmapper.ActionMoveWest:
		st.moveCursor(-1, 0)
		st.refreshItems(world)
	case inputmapper.ActionMoveEast:
		st.moveCursor(1, 0)
		st.refreshItems(world)

	case inputmapper.ActionPickup:
		if len(st.itemsAtTarget) > 0 {
			if err := st.executePickup(world); err != nil {
				return es.Transition[w.World]{}, err
			}
			return es.Transition[w.World]{Type: es.TransPop}, nil
		}

	default:
		// このステートでは処理しないアクションは無視する
	}

	return st.ConsumeTransition(), nil
}

// moveCursor はカーソルを移動する。プレイヤーからチェビシェフ距離1以内に制限する
func (st *PickupState) moveCursor(dx, dy int) {
	newX := int(st.cursor.X) + dx
	newY := int(st.cursor.Y) + dy

	distX := newX - int(st.playerPos.X)
	distY := newY - int(st.playerPos.Y)
	if distX < 0 {
		distX = -distX
	}
	if distY < 0 {
		distY = -distY
	}

	if distX <= 1 && distY <= 1 {
		st.cursor.X = consts.Tile(newX)
		st.cursor.Y = consts.Tile(newY)
	}
}

// refreshItems はカーソル位置のタイル上の拾得可能エンティティを更新する
func (st *PickupState) refreshItems(world w.World) {
	st.itemsAtTarget = nil

	targetX := int(st.cursor.X)
	targetY := int(st.cursor.Y)

	world.Manager.Join(
		world.Components.GridElement,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		grid := world.Components.GridElement.Get(entity).(*gc.GridElement)
		if int(grid.X) != targetX || int(grid.Y) != targetY {
			return
		}
		if worldhelper.IsPickable(entity, world) {
			st.itemsAtTarget = append(st.itemsAtTarget, entity)
		}
	}))
}

// executePickup は拾得アクションを実行する
func (st *PickupState) executePickup(world w.World) error {
	playerEntity, err := worldhelper.GetPlayerEntity(world)
	if err != nil {
		return err
	}

	destination := gc.GridElement{X: st.cursor.X, Y: st.cursor.Y}
	params := activity.ActionParams{
		Actor:       playerEntity,
		Destination: &destination,
	}
	_, err = activity.Execute(&activity.PickupActivity{}, params, world)
	return err
}

// Draw はステートの描画処理
func (st *PickupState) Draw(world w.World, screen *ebiten.Image) error {
	st.drawTargetCursor(world, screen)
	return st.drawPickupPanel(world, screen)
}

// pickupCursorCache はカーソル画像のキャッシュ
var pickupCursorCache *ebiten.Image

// pickupPanelCache は情報パネル画像のキャッシュ
var pickupPanelCache *ebiten.Image

func (st *PickupState) drawTargetCursor(world w.World, screen *ebiten.Image) {
	tileSize := int(consts.TileSize)
	cursorPixelX := float64(int(st.cursor.X) * tileSize)
	cursorPixelY := float64(int(st.cursor.Y) * tileSize)

	if pickupCursorCache == nil {
		pickupCursorCache = ebiten.NewImage(tileSize, tileSize)
		cursorColor := color.RGBA{R: 50, G: 200, B: 255, A: 255} // 青
		for i := 0; i < 3; i++ {
			for x := 0; x < tileSize; x++ {
				pickupCursorCache.Set(x, i, cursorColor)
				pickupCursorCache.Set(x, tileSize-1-i, cursorColor)
			}
			for y := 0; y < tileSize; y++ {
				pickupCursorCache.Set(i, y, cursorColor)
				pickupCursorCache.Set(tileSize-1-i, y, cursorColor)
			}
		}
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(cursorPixelX, cursorPixelY)
	gs.SetTranslate(world, op)

	alpha := 0.6 + 0.4*math.Sin(float64(st.blinkCounter)*0.15)
	op.ColorScale.ScaleAlpha(float32(alpha))

	screen.DrawImage(pickupCursorCache, op)
}

func (st *PickupState) drawPickupPanel(world w.World, screen *ebiten.Image) error {
	face := world.Resources.UIResources.Text.BodyFace

	const (
		panelWidth  = 300
		panelHeight = 200
		marginX     = 10
		marginY     = 10
		lineHeight  = 20
	)

	if pickupPanelCache == nil {
		pickupPanelCache = ebiten.NewImage(panelWidth, panelHeight)
		pickupPanelCache.Fill(color.RGBA{R: 0, G: 0, B: 0, A: 200})
	}

	panelX := screen.Bounds().Dx() - panelWidth - marginX
	panelY := marginY
	panelOp := &ebiten.DrawImageOptions{}
	panelOp.GeoM.Translate(float64(panelX), float64(panelY))
	screen.DrawImage(pickupPanelCache, panelOp)

	textX := float64(panelX + 10)
	y := panelY + 10

	drawText := func(str string) {
		op := &text.DrawOptions{}
		op.GeoM.Translate(textX, float64(y))
		text.Draw(screen, str, face, op)
		y += lineHeight
	}

	drawText("== 拾うモード ==")
	y += 5

	dirLabel := offsetToLabel(int(st.cursor.X)-int(st.playerPos.X), int(st.cursor.Y)-int(st.playerPos.Y))
	drawText(fmt.Sprintf("方向: %s", dirLabel))
	y += 5

	if len(st.itemsAtTarget) == 0 {
		drawText("拾えるものがありません")
	} else {
		drawText(fmt.Sprintf("アイテム: %d個", len(st.itemsAtTarget)))
		for i, entity := range st.itemsAtTarget {
			if i >= 5 {
				drawText("...")
				break
			}
			name := worldhelper.GetEntityName(entity, world)
			drawText(fmt.Sprintf("  - %s", name))
		}
	}

	y = panelY + panelHeight - 30
	drawText("WASD/矢印:移動 Enter:拾う Esc:戻る")

	return nil
}

// offsetToLabel はプレイヤーからのオフセットを日本語ラベルに変換する
func offsetToLabel(dx, dy int) string {
	switch {
	case dx == 0 && dy == -1:
		return "上"
	case dx == 0 && dy == 1:
		return "下"
	case dx == -1 && dy == 0:
		return "左"
	case dx == 1 && dy == 0:
		return "右"
	case dx == -1 && dy == -1:
		return "左上"
	case dx == 1 && dy == -1:
		return "右上"
	case dx == -1 && dy == 1:
		return "左下"
	case dx == 1 && dy == 1:
		return "右下"
	default:
		return "足元"
	}
}
