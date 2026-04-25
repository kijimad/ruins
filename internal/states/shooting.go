package states

import (
	"fmt"
	"image/color"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	gs "github.com/kijimaD/ruins/internal/systems"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// ShootingState は射撃ターゲット選択モードのステート
// 視界内の敵をTabで巡回し、Enterで射撃、Rでリロード、Escapeでキャンセルする
type ShootingState struct {
	es.BaseState[w.World]
	enemies        []ecs.Entity // 視界内の敵一覧
	targetIndex    int          // 現在選択中の敵インデックス
	blinkCounter   int          // カーソル点滅用カウンタ
	cachedHitRate  int          // キャッシュ済み命中率
	cachedDistance float64      // キャッシュ済み距離
}

// StateConfig はこのステートの設定を返す
func (st *ShootingState) StateConfig() StateConfig {
	return StateConfig{BlurBackground: false}
}

func (st ShootingState) String() string {
	return "Shooting"
}

var _ es.State[w.World] = &ShootingState{}
var _ Configurable = &ShootingState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *ShootingState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *ShootingState) OnResume(_ w.World) error { return nil }

// OnStart はステートが開始される際に呼ばれる
func (st *ShootingState) OnStart(world w.World) error {
	if err := st.refreshEnemies(world); err != nil {
		return err
	}
	st.updateTargetCache(world)
	return nil
}

// OnStop はステートが終了する際に呼ばれる
func (st *ShootingState) OnStop(_ w.World) error { return nil }

// Update はステートの更新処理
func (st *ShootingState) Update(world w.World) (es.Transition[w.World], error) {
	st.blinkCounter++

	if action, ok := st.handleInput(); ok {
		return st.doAction(world, action)
	}

	return st.ConsumeTransition(), nil
}

// handleInput はキー入力をActionIDに変換する
func (st *ShootingState) handleInput() (inputmapper.ActionID, bool) {
	keyboardInput := input.GetSharedKeyboardInput()

	if keyboardInput.IsKeyJustPressed(ebiten.KeyEscape) {
		return inputmapper.ActionCloseMenu, true
	}
	if keyboardInput.IsKeyJustPressed(ebiten.KeyTab) {
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			return inputmapper.ActionMenuTabPrev, true
		}
		return inputmapper.ActionMenuTabNext, true
	}
	if keyboardInput.IsEnterJustPressedOnce() {
		return inputmapper.ActionShoot, true
	}
	if keyboardInput.IsKeyJustPressed(ebiten.KeyR) {
		return inputmapper.ActionReload, true
	}

	return "", false
}

// doAction はActionIDを実行する
func (st *ShootingState) doAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil

	case inputmapper.ActionMenuTabNext:
		if len(st.enemies) > 0 {
			st.targetIndex = (st.targetIndex + 1) % len(st.enemies)
			st.updateTargetCache(world)
		}

	case inputmapper.ActionMenuTabPrev:
		if len(st.enemies) > 0 {
			st.targetIndex = (st.targetIndex - 1 + len(st.enemies)) % len(st.enemies)
			st.updateTargetCache(world)
		}

	case inputmapper.ActionShoot:
		if len(st.enemies) == 0 {
			gamelog.New(gamelog.FieldLog).Append("射撃対象がいません").Log()
			return es.Transition[w.World]{Type: es.TransPop}, nil
		}
		playerEntity, err := worldhelper.GetPlayerEntity(world)
		if err != nil {
			return es.Transition[w.World]{}, err
		}
		target := st.enemies[st.targetIndex]
		if err := activity.ExecuteShootAction(playerEntity, target, world); err != nil {
			return es.Transition[w.World]{}, err
		}
		return es.Transition[w.World]{Type: es.TransPop}, nil

	case inputmapper.ActionReload:
		playerEntity, err := worldhelper.GetPlayerEntity(world)
		if err != nil {
			return es.Transition[w.World]{}, err
		}
		if err := activity.ExecuteReloadAction(playerEntity, world); err != nil {
			return es.Transition[w.World]{}, err
		}
		return es.Transition[w.World]{Type: es.TransPop}, nil

	default:
		return es.Transition[w.World]{}, fmt.Errorf("未対応のアクション: %s", action)
	}

	return st.ConsumeTransition(), nil
}

// refreshEnemies は射撃可能な敵一覧を距離順で更新する。
// 視界内の敵から死亡済み・射程外・射線遮断の敵を除外する
func (st *ShootingState) refreshEnemies(world w.World) error {
	enemies, err := worldhelper.GetVisibleEnemies(world)
	if err != nil {
		return err
	}

	playerEntity, playerErr := worldhelper.GetPlayerEntity(world)
	if playerErr != nil {
		return playerErr
	}

	// 射撃可能な敵のみ残す
	var shootable []ecs.Entity
	for _, e := range enemies {
		if activity.CanShootTarget(playerEntity, e, world) {
			shootable = append(shootable, e)
		}
	}

	// プレイヤーからの距離順にソート
	sort.Slice(shootable, func(i, j int) bool {
		di := activity.EntityDistance(playerEntity, shootable[i], world)
		dj := activity.EntityDistance(playerEntity, shootable[j], world)
		return di < dj
	})

	st.enemies = shootable
	if st.targetIndex >= len(st.enemies) {
		st.targetIndex = 0
	}
	return nil
}

// updateTargetCache はターゲット変更時に命中率と距離をキャッシュする
func (st *ShootingState) updateTargetCache(world w.World) {
	if len(st.enemies) == 0 {
		return
	}
	playerEntity, err := worldhelper.GetPlayerEntity(world)
	if err != nil {
		return
	}
	target := st.enemies[st.targetIndex]
	st.cachedHitRate = activity.CalculateShootHitRate(playerEntity, target, world)
	st.cachedDistance = activity.EntityDistance(playerEntity, target, world)
}

// Draw はステートの描画処理
func (st *ShootingState) Draw(world w.World, screen *ebiten.Image) error {
	if len(st.enemies) > 0 {
		st.drawTargetCursor(world, screen)
	}
	return st.drawShootingPanel(world, screen)
}

// shootingCursorCache はターゲットカーソル画像のキャッシュ
var shootingCursorCache *ebiten.Image

// drawTargetCursor は選択中の敵にカーソルを描画する
func (st *ShootingState) drawTargetCursor(world w.World, screen *ebiten.Image) {
	target := st.enemies[st.targetIndex]
	if !target.HasComponent(world.Components.GridElement) {
		return
	}
	targetGrid := world.Components.GridElement.Get(target).(*gc.GridElement)

	tileSize := int(consts.TileSize)
	cursorPixelX := float64(int(targetGrid.X) * tileSize)
	cursorPixelY := float64(int(targetGrid.Y) * tileSize)

	if shootingCursorCache == nil {
		shootingCursorCache = ebiten.NewImage(tileSize, tileSize)
		cursorColor := color.RGBA{R: 255, G: 50, B: 50, A: 255} // 赤
		for i := 0; i < 3; i++ {
			for x := 0; x < tileSize; x++ {
				shootingCursorCache.Set(x, i, cursorColor)
				shootingCursorCache.Set(x, tileSize-1-i, cursorColor)
			}
			for y := 0; y < tileSize; y++ {
				shootingCursorCache.Set(i, y, cursorColor)
				shootingCursorCache.Set(tileSize-1-i, y, cursorColor)
			}
		}
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(cursorPixelX, cursorPixelY)
	gs.SetTranslate(world, op)

	// 点滅エフェクト
	alpha := 0.6 + 0.4*math.Sin(float64(st.blinkCounter)*0.15)
	op.ColorScale.ScaleAlpha(float32(alpha))

	screen.DrawImage(shootingCursorCache, op)
}

// shootingPanelCache は情報パネル画像のキャッシュ
var shootingPanelCache *ebiten.Image

// drawShootingPanel は射撃情報パネルを描画する
func (st *ShootingState) drawShootingPanel(world w.World, screen *ebiten.Image) error {
	face := world.Resources.UIResources.Text.BodyFace

	const (
		panelWidth  = 300
		panelHeight = 250
		marginX     = 10
		marginY     = 10
		lineHeight  = 20
	)

	if shootingPanelCache == nil {
		shootingPanelCache = ebiten.NewImage(panelWidth, panelHeight)
		shootingPanelCache.Fill(color.RGBA{R: 0, G: 0, B: 0, A: 200})
	}

	panelX := screen.Bounds().Dx() - panelWidth - marginX
	panelY := marginY
	panelOp := &ebiten.DrawImageOptions{}
	panelOp.GeoM.Translate(float64(panelX), float64(panelY))
	screen.DrawImage(shootingPanelCache, panelOp)

	textX := float64(panelX + 10)
	y := panelY + 10

	drawText := func(str string) {
		op := &text.DrawOptions{}
		op.GeoM.Translate(textX, float64(y))
		text.Draw(screen, str, face, op)
		y += lineHeight
	}

	drawText("== 射撃モード ==")
	y += 5

	// 武器・残弾情報
	playerEntity, err := worldhelper.GetPlayerEntity(world)
	if err != nil {
		drawText("エラー: プレイヤーが見つかりません")
		return err
	}

	st.drawWeaponInfo(world, playerEntity, drawText)
	y += 5

	// ターゲット情報
	if len(st.enemies) == 0 {
		drawText("射撃対象がいません")
	} else {
		target := st.enemies[st.targetIndex]
		st.drawTargetInfo(world, target, drawText)
	}

	// 操作説明
	y = panelY + panelHeight - 30
	drawText("Tab:切替 Enter:射撃 R:装填 Esc:戻る")

	return nil
}

// drawWeaponInfo は武器情報を描画する
func (st *ShootingState) drawWeaponInfo(world w.World, playerEntity ecs.Entity, drawText func(string)) {
	selectedSlot := world.Resources.Dungeon.SelectedWeaponSlot
	weapons := worldhelper.GetWeapons(world, playerEntity)
	weaponIndex := selectedSlot - 1
	if weaponIndex < 0 || weaponIndex >= len(weapons) {
		drawText("武器スロット: 無効")
		return
	}

	weaponEntity := weapons[weaponIndex]
	if weaponEntity == nil {
		drawText("武器: なし")
		return
	}

	// 武器名
	weaponName := worldhelper.GetEntityName(*weaponEntity, world)
	drawText(fmt.Sprintf("武器: %s", weaponName))

	// 残弾表示
	weaponComp := world.Components.Weapon.Get(*weaponEntity)
	if weaponComp != nil {
		weapon := weaponComp.(*gc.Weapon)
		if weapon.MagazineSize > 0 {
			drawText(fmt.Sprintf("残弾: %d/%d", weapon.Magazine, weapon.MagazineSize))
		} else {
			drawText("近接武器")
		}
	}
}

// drawTargetInfo はターゲット情報を描画する。キャッシュ済みの値を使用する
func (st *ShootingState) drawTargetInfo(world w.World, target ecs.Entity, drawText func(string)) {
	drawText(fmt.Sprintf("対象: %s (%d/%d)",
		worldhelper.GetEntityName(target, world),
		st.targetIndex+1, len(st.enemies)))

	// HP
	if target.HasComponent(world.Components.Pools) {
		pools := world.Components.Pools.Get(target).(*gc.Pools)
		drawText(fmt.Sprintf("HP: %d/%d", pools.HP.Current, pools.HP.Max))
	}

	drawText(fmt.Sprintf("命中率: %d%%", st.cachedHitRate))
	drawText(fmt.Sprintf("距離: %.1f", st.cachedDistance))
}
