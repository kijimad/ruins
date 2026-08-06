package states

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/kijimaD/ruins/internal/activity"
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

var _ es.State[w.World] = &ShootingState{}

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
		if len(st.enemies) > 0 {
			playerEntity, err := query.GetPlayerEntity(world)
			if err != nil {
				return es.Transition[w.World]{}, err
			}
			target := st.enemies[st.targetIndex]
			if err := activity.ExecuteShootAction(playerEntity, target, world); err != nil {
				return es.Transition[w.World]{}, err
			}
			return es.Transition[w.World]{Type: es.TransPop}, nil
		}

	case inputmapper.ActionReload:
		playerEntity, err := query.GetPlayerEntity(world)
		if err != nil {
			return es.Transition[w.World]{}, err
		}
		if err := activity.ExecuteReloadAction(playerEntity, world); err != nil {
			return es.Transition[w.World]{}, err
		}
		return es.Transition[w.World]{Type: es.TransPop}, nil

	default:
		return es.Transition[w.World]{}, fmt.Errorf("unsupported action: %s", action)
	}

	return st.ConsumeTransition(), nil
}

// checkFireWeaponStatus は選択中の武器の射撃可否をチェックし、不可の場合は理由メッセージを返す。
// 射撃可能な場合は空文字を返す
func (st *ShootingState) checkFireWeaponStatus(world w.World) string {
	playerEntity, err := query.GetPlayerEntity(world)
	if err != nil {
		return ""
	}
	selectedSlot := query.GetWeaponSelection(world).Slot
	weapons := query.GetWeapons(world, playerEntity)
	weaponIndex := selectedSlot - 1
	if weaponIndex < 0 || weaponIndex >= len(weapons) || weapons[weaponIndex] == nil {
		return query.T(world, "No ranged weapon equipped")
	}
	fire := world.Components.Fire.Get(*weapons[weaponIndex])
	if fire == nil {
		return query.T(world, "No ranged weapon equipped")
	}
	if fire.Magazine <= 0 {
		return query.T(world, "Not loaded")
	}
	return ""
}

// refreshEnemies は射撃可能な敵一覧を距離順で更新する。
// 視界内の敵から死亡済み・射程外・射線遮断の敵を除外する
func (st *ShootingState) refreshEnemies(world w.World) error {
	enemies, err := query.GetVisibleEnemies(world)
	if err != nil {
		return err
	}

	playerEntity, playerErr := query.GetPlayerEntity(world)
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
	slices.SortFunc(shootable, func(a, b ecs.Entity) int {
		da := activity.EntityDistance(playerEntity, a, world)
		db := activity.EntityDistance(playerEntity, b, world)
		return cmp.Compare(da, db)
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
	playerEntity, err := query.GetPlayerEntity(world)
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

// shootingCursorCache はターゲットカーソル画像のキャッシュ。sync.Once で一度だけ初期化する
var (
	shootingCursorCache     *ebiten.Image
	shootingCursorCacheOnce sync.Once
)

// drawTargetCursor は選択中の敵にカーソルを描画する
func (st *ShootingState) drawTargetCursor(world w.World, screen *ebiten.Image) {
	target := st.enemies[st.targetIndex]
	if !world.Components.GridElement.Has(target) {
		return
	}
	targetGrid := world.Components.GridElement.Get(target)

	tileSize := int(consts.TileSize)
	cursorPixelX := float64(int(targetGrid.X) * tileSize)
	cursorPixelY := float64(int(targetGrid.Y) * tileSize)

	shootingCursorCacheOnce.Do(func() {
		shootingCursorCache = ebiten.NewImage(tileSize, tileSize)
		cursorColor := theme.CursorShoot
		for i := range 3 {
			for x := range tileSize {
				shootingCursorCache.Set(x, i, cursorColor)
				shootingCursorCache.Set(x, tileSize-1-i, cursorColor)
			}
			for y := range tileSize {
				shootingCursorCache.Set(i, y, cursorColor)
				shootingCursorCache.Set(tileSize-1-i, y, cursorColor)
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

	screen.DrawImage(shootingCursorCache, op)
}

// shootingPanelCache は情報パネル画像のキャッシュ。sync.Once で一度だけ初期化する
var (
	shootingPanelCache     *ebiten.Image
	shootingPanelCacheOnce sync.Once
)

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

	shootingPanelCacheOnce.Do(func() {
		shootingPanelCache = ebiten.NewImage(panelWidth, panelHeight)
		shootingPanelCache.Fill(theme.Overlay)
	})

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

	drawText(query.T(world, "== Shooting Mode =="))
	y += 5

	// 武器・残弾情報
	playerEntity, err := query.GetPlayerEntity(world)
	if err != nil {
		drawText(query.T(world, "Error: player not found"))
		return err
	}

	st.drawWeaponInfo(world, playerEntity, drawText)
	y += 5

	// ターゲット情報
	if msg := st.checkFireWeaponStatus(world); msg != "" {
		drawText(msg)
	} else if len(st.enemies) == 0 {
		drawText(query.T(world, "No shooting target"))
	} else {
		target := st.enemies[st.targetIndex]
		st.drawTargetInfo(world, target, drawText)
	}

	// 操作説明
	y = panelY + panelHeight - 30
	drawText(query.T(world, "Tab: Switch  Enter: Fire  R: Reload  Esc: Back"))

	return nil
}

// drawWeaponInfo は武器情報を描画する
func (st *ShootingState) drawWeaponInfo(world w.World, playerEntity ecs.Entity, drawText func(string)) {
	selectedSlot := query.GetWeaponSelection(world).Slot
	weapons := query.GetWeapons(world, playerEntity)
	weaponIndex := selectedSlot - 1
	if weaponIndex < 0 || weaponIndex >= len(weapons) {
		drawText(query.T(world, "Weapon slot: invalid"))
		return
	}

	weaponEntity := weapons[weaponIndex]
	if weaponEntity == nil {
		drawText(query.T(world, "Weapon: none"))
		return
	}

	// 武器名
	weaponName := query.GetEntityName(*weaponEntity, world)
	drawText(fmt.Sprintf("%s: %s", query.T(world, "Weapon"), weaponName))

	// 残弾表示
	fireComp := world.Components.Fire.Get(*weaponEntity)
	if fireComp != nil {
		drawText(fmt.Sprintf("%s: %d/%d", query.T(world, "Ammo"), fireComp.Magazine, fireComp.MagazineSize))
	} else {
		drawText(query.T(world, "Melee weapon"))
	}
}

// drawTargetInfo はターゲット情報を描画する。キャッシュ済みの値を使用する
func (st *ShootingState) drawTargetInfo(world w.World, target ecs.Entity, drawText func(string)) {
	drawText(fmt.Sprintf("%s: %s (%d/%d)", query.T(world, "Target"),
		query.GetEntityName(target, world),
		st.targetIndex+1, len(st.enemies)))

	// HP
	if world.Components.HP.Has(target) {
		hp := world.Components.HP.Get(target)
		drawText(fmt.Sprintf("HP: %d/%d", hp.Current, hp.Max))
	}

	drawText(fmt.Sprintf("%s: %d%%", query.T(world, "Hit rate"), st.cachedHitRate))
	drawText(fmt.Sprintf("%s: %.1f", query.T(world, "Distance"), st.cachedDistance))
}
