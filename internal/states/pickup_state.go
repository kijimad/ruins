package states

import (
	"fmt"
	"image/color"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/geometry"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	gs "github.com/kijimaD/ruins/internal/systems"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
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
	playerEntity, err := query.GetPlayerEntity(world)
	if err != nil {
		return err
	}
	playerGrid := world.Components.GridElement.Get(playerEntity)
	st.playerPos = playerGrid.Coord
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
		moveCursorAdjacent(&st.cursor, st.playerPos, 0, -1)
		st.refreshItems(world)
	case inputmapper.ActionMoveSouth:
		moveCursorAdjacent(&st.cursor, st.playerPos, 0, 1)
		st.refreshItems(world)
	case inputmapper.ActionMoveWest:
		moveCursorAdjacent(&st.cursor, st.playerPos, -1, 0)
		st.refreshItems(world)
	case inputmapper.ActionMoveEast:
		moveCursorAdjacent(&st.cursor, st.playerPos, 1, 0)
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

// refreshItems はカーソル位置のタイル上の拾得可能エンティティを更新する
func (st *PickupState) refreshItems(world w.World) {
	st.itemsAtTarget = nil
	for _, entity := range query.GetEntitiesAt(world, st.cursor.X, st.cursor.Y) {
		if query.IsPickable(entity, world) {
			st.itemsAtTarget = append(st.itemsAtTarget, entity)
		}
	}
}

// executePickup は拾得アクションを実行する
func (st *PickupState) executePickup(world w.World) error {
	playerEntity, err := query.GetPlayerEntity(world)
	if err != nil {
		return err
	}

	destination := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: st.cursor.X, Y: st.cursor.Y}}
	_, err = activity.Execute(&activity.PickupActivity{Destination: &destination}, playerEntity, world)
	return err
}

// Draw はステートの描画処理
func (st *PickupState) Draw(world w.World, screen *ebiten.Image) error {
	st.drawTargetCursor(world, screen)
	return st.drawPickupPanel(world, screen)
}

// pickupCursorCache はカーソル画像のキャッシュ。sync.Once で一度だけ初期化する
var (
	pickupCursorCache     *ebiten.Image
	pickupCursorCacheOnce sync.Once
)

func (st *PickupState) drawTargetCursor(world w.World, screen *ebiten.Image) {
	tileSize := int(consts.TileSize)
	cursorPixelX := float64(int(st.cursor.X) * tileSize)
	cursorPixelY := float64(int(st.cursor.Y) * tileSize)

	pickupCursorCacheOnce.Do(func() {
		pickupCursorCache = ebiten.NewImage(tileSize, tileSize)
		cursorColor := theme.CursorPickup
		for i := range 3 {
			for x := range tileSize {
				pickupCursorCache.Set(x, i, cursorColor)
				pickupCursorCache.Set(x, tileSize-1-i, cursorColor)
			}
			for y := range tileSize {
				pickupCursorCache.Set(i, y, cursorColor)
				pickupCursorCache.Set(tileSize-1-i, y, cursorColor)
			}
		}
	})

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(cursorPixelX, cursorPixelY)
	gs.SetTranslate(world, op)

	if !world.Config.DisableAnimation {
		alpha := 0.6 + 0.4*math.Sin(float64(st.blinkCounter)*0.15)
		op.ColorScale.ScaleAlpha(float32(alpha))
	}

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

	panelX := screen.Bounds().Dx() - panelWidth - marginX
	panelY := marginY
	styled.DrawFramedBackground(screen, panelX, panelY, panelWidth, panelHeight, styled.PanelStyle())

	textX := float64(panelX + 12)
	y := panelY + 12

	drawColorText := func(str string, c color.RGBA) {
		op := &text.DrawOptions{}
		op.GeoM.Translate(textX, float64(y))
		op.ColorScale.ScaleWithColor(c)
		text.Draw(screen, str, face, op)
		y += lineHeight
	}
	drawText := func(str string) { drawColorText(str, theme.TextPrimary) }

	drawColorText("拾うモード", theme.TextPrimary)
	y += 5

	dirLabel := offsetToLabel(int(st.cursor.X)-int(st.playerPos.X), int(st.cursor.Y)-int(st.playerPos.Y))
	drawText(fmt.Sprintf("方向: %s", dirLabel))
	y += 5

	if len(st.itemsAtTarget) == 0 {
		drawColorText("拾えるものがありません", theme.TextSecondary)
	} else {
		drawText(fmt.Sprintf("アイテム: %d個", len(st.itemsAtTarget)))
		for i, entity := range st.itemsAtTarget {
			if i >= 5 {
				drawColorText("...", theme.TextSecondary)
				break
			}
			name := query.GetEntityName(entity, world)
			drawText(fmt.Sprintf("  - %s", name))
		}
	}

	y = panelY + panelHeight - 30
	drawColorText("WASD/矢印:移動 Enter:拾う Esc:戻る", theme.TextSecondary)

	return nil
}

// moveCursorAdjacent はカーソルを移動する。基準点からチェビシェフ距離1以内に制限する
func moveCursorAdjacent(cursor *consts.Coord[consts.Tile], origin consts.Coord[consts.Tile], dx, dy int) {
	next := cursor.Add(consts.Coord[consts.Tile]{X: consts.Tile(dx), Y: consts.Tile(dy)})
	if geometry.ChebyshevDistance(int(next.X), int(next.Y), int(origin.X), int(origin.Y)) <= 1 {
		*cursor = next
	}
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
