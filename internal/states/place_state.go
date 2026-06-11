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

// placePhase は置くモードのフェーズを表す
type placePhase int

const (
	// placePhaseSelectItem はアイテム選択フェーズを表す
	placePhaseSelectItem placePhase = iota
	// placePhaseSelectTile は配置先選択フェーズを表す
	placePhaseSelectTile
)

// PlaceState は置くモードのモーダルステート。
// フェーズ1: 上下キーでバックパック内アイテムを選択しEnterで確定する。
// フェーズ2: WASD/矢印キーで隣接8タイルにカーソルを移動しEnterで置く
type PlaceState struct {
	es.BaseState[w.World]
	phase         placePhase                // 現在のフェーズ
	cursor        consts.Coord[consts.Tile] // カーソル位置（絶対座標）
	playerPos     consts.Coord[consts.Tile] // プレイヤー位置（移動制限用）
	backpackItems []ecs.Entity              // バックパック内のアイテム一覧
	selectedIndex int                       // 選択中のアイテムインデックス
	blinkCounter  int                       // カーソル点滅用
}

// StateConfig はこのステートの設定を返す
func (st *PlaceState) StateConfig() StateConfig {
	return StateConfig{BlurBackground: false}
}

func (st PlaceState) String() string {
	return "Place"
}

var _ es.State[w.World] = &PlaceState{}
var _ Configurable = &PlaceState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *PlaceState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *PlaceState) OnResume(_ w.World) error { return nil }

// OnStart はステートが開始される際に呼ばれる
func (st *PlaceState) OnStart(world w.World) error {
	playerEntity, err := worldhelper.GetPlayerEntity(world)
	if err != nil {
		return err
	}
	playerGrid := world.Components.GridElement.Get(playerEntity).(*gc.GridElement)
	st.playerPos = consts.Coord[consts.Tile]{X: playerGrid.X, Y: playerGrid.Y}
	st.phase = placePhaseSelectItem
	st.refreshBackpackItems(world)
	return nil
}

// OnStop はステートが終了する際に呼ばれる
func (st *PlaceState) OnStop(_ w.World) error { return nil }

// Update はステートの更新処理
func (st *PlaceState) Update(world w.World) (es.Transition[w.World], error) {
	st.blinkCounter++

	if action, ok := st.handleInput(); ok {
		return st.doAction(world, action)
	}

	return st.ConsumeTransition(), nil
}

func (st *PlaceState) handleInput() (inputmapper.ActionID, bool) {
	keyboardInput := input.GetSharedKeyboardInput()

	if keyboardInput.IsKeyJustPressed(ebiten.KeyEscape) {
		return inputmapper.ActionCloseMenu, true
	}
	if keyboardInput.IsEnterJustPressedOnce() {
		return inputmapper.ActionPlace, true
	}

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

func (st *PlaceState) doAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch st.phase {
	case placePhaseSelectItem:
		return st.doActionSelectItem(world, action)
	case placePhaseSelectTile:
		return st.doActionSelectTile(world, action)
	}
	return st.ConsumeTransition(), nil
}

// doActionSelectItem はアイテム選択フェーズのアクションを処理する
func (st *PlaceState) doActionSelectItem(_ w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil

	case inputmapper.ActionMoveNorth:
		if len(st.backpackItems) > 0 {
			st.selectedIndex = (st.selectedIndex - 1 + len(st.backpackItems)) % len(st.backpackItems)
		}
	case inputmapper.ActionMoveSouth:
		if len(st.backpackItems) > 0 {
			st.selectedIndex = (st.selectedIndex + 1) % len(st.backpackItems)
		}

	case inputmapper.ActionPlace:
		if len(st.backpackItems) > 0 {
			st.phase = placePhaseSelectTile
			st.cursor = consts.Coord[consts.Tile]{X: st.playerPos.X, Y: st.playerPos.Y - 1}
		}

	default:
	}

	return st.ConsumeTransition(), nil
}

// doActionSelectTile は配置先選択フェーズのアクションを処理する
func (st *PlaceState) doActionSelectTile(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionCloseMenu:
		st.phase = placePhaseSelectItem
		return st.ConsumeTransition(), nil

	case inputmapper.ActionMoveNorth:
		st.moveCursor(0, -1)
	case inputmapper.ActionMoveSouth:
		st.moveCursor(0, 1)
	case inputmapper.ActionMoveWest:
		st.moveCursor(-1, 0)
	case inputmapper.ActionMoveEast:
		st.moveCursor(1, 0)

	case inputmapper.ActionPlace:
		if err := st.executeDrop(world); err != nil {
			return es.Transition[w.World]{}, err
		}
		return es.Transition[w.World]{Type: es.TransPop}, nil

	default:
	}

	return st.ConsumeTransition(), nil
}

// moveCursor はカーソルを移動する。プレイヤーからチェビシェフ距離1以内に制限する
func (st *PlaceState) moveCursor(dx, dy int) {
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

// refreshBackpackItems はバックパック内の全アイテムを取得する
func (st *PlaceState) refreshBackpackItems(world w.World) {
	st.backpackItems = nil

	world.Manager.Join(
		world.Components.Item,
		world.Components.ItemLocationInPlayerBackpack,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		st.backpackItems = append(st.backpackItems, entity)
	}))

	if st.selectedIndex >= len(st.backpackItems) {
		st.selectedIndex = 0
	}
}

// executeDrop は置くアクションを実行する
func (st *PlaceState) executeDrop(world w.World) error {
	playerEntity, err := worldhelper.GetPlayerEntity(world)
	if err != nil {
		return err
	}

	item := st.backpackItems[st.selectedIndex]
	destination := gc.GridElement{X: st.cursor.X, Y: st.cursor.Y}
	params := activity.ActionParams{
		Actor:       playerEntity,
		Target:      &item,
		Destination: &destination,
	}
	_, err = activity.Execute(&activity.DropActivity{}, params, world)
	return err
}

// Draw はステートの描画処理
func (st *PlaceState) Draw(world w.World, screen *ebiten.Image) error {
	if st.phase == placePhaseSelectTile {
		st.drawTargetCursor(world, screen)
	}
	return st.drawPlacePanel(world, screen)
}

// placeCursorCache はカーソル画像のキャッシュ
var placeCursorCache *ebiten.Image

func (st *PlaceState) drawTargetCursor(world w.World, screen *ebiten.Image) {
	tileSize := int(consts.TileSize)
	cursorPixelX := float64(int(st.cursor.X) * tileSize)
	cursorPixelY := float64(int(st.cursor.Y) * tileSize)

	if placeCursorCache == nil {
		placeCursorCache = ebiten.NewImage(tileSize, tileSize)
		cursorColor := color.RGBA{R: 50, G: 255, B: 100, A: 255} // 緑
		for i := 0; i < 3; i++ {
			for x := 0; x < tileSize; x++ {
				placeCursorCache.Set(x, i, cursorColor)
				placeCursorCache.Set(x, tileSize-1-i, cursorColor)
			}
			for y := 0; y < tileSize; y++ {
				placeCursorCache.Set(i, y, cursorColor)
				placeCursorCache.Set(tileSize-1-i, y, cursorColor)
			}
		}
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(cursorPixelX, cursorPixelY)
	gs.SetTranslate(world, op)

	alpha := 0.6 + 0.4*math.Sin(float64(st.blinkCounter)*0.15)
	op.ColorScale.ScaleAlpha(float32(alpha))

	screen.DrawImage(placeCursorCache, op)
}

func (st *PlaceState) drawPlacePanel(world w.World, screen *ebiten.Image) error {
	face := world.Resources.UIResources.Text.BodyFace

	const (
		panelWidth  = 300
		panelHeight = 200
		marginX     = 10
		marginY     = 10
		lineHeight  = 20
	)

	panelImg := ebiten.NewImage(panelWidth, panelHeight)
	panelImg.Fill(color.RGBA{R: 0, G: 0, B: 0, A: 200})

	panelX := screen.Bounds().Dx() - panelWidth - marginX
	panelY := marginY
	panelOp := &ebiten.DrawImageOptions{}
	panelOp.GeoM.Translate(float64(panelX), float64(panelY))
	screen.DrawImage(panelImg, panelOp)

	textX := float64(panelX + 10)
	y := panelY + 10

	drawText := func(str string) {
		op := &text.DrawOptions{}
		op.GeoM.Translate(textX, float64(y))
		text.Draw(screen, str, face, op)
		y += lineHeight
	}

	switch st.phase {
	case placePhaseSelectItem:
		drawText("== 置くモード: アイテム選択 ==")
		y += 5

		if len(st.backpackItems) == 0 {
			drawText("置けるアイテムがありません")
		} else {
			for i, entity := range st.backpackItems {
				if i >= 7 {
					drawText("...")
					break
				}
				name := worldhelper.GetEntityName(entity, world)
				prefix := "  "
				if i == st.selectedIndex {
					prefix = "> "
				}
				drawText(fmt.Sprintf("%s%s", prefix, name))
			}
		}

		y = panelY + panelHeight - 30
		drawText("↑↓:選択 Enter:決定 Esc:戻る")

	case placePhaseSelectTile:
		drawText("== 置くモード: 配置先選択 ==")
		y += 5

		item := st.backpackItems[st.selectedIndex]
		name := worldhelper.GetEntityName(item, world)
		drawText(fmt.Sprintf("アイテム: %s", name))

		dirLabel := offsetToLabel(int(st.cursor.X)-int(st.playerPos.X), int(st.cursor.Y)-int(st.playerPos.Y))
		drawText(fmt.Sprintf("方向: %s", dirLabel))

		y = panelY + panelHeight - 30
		drawText("WASD/矢印:移動 Enter:置く Esc:戻る")
	}

	return nil
}
