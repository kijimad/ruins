package states

import (
	"cmp"
	"fmt"
	"math"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/activity"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
	"github.com/kijimaD/ruins/internal/render3d"
	"github.com/kijimaD/ruins/internal/widgets/hud"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// targetMode は敵を狙う画面で、射撃と投げるで違う部分
type targetMode interface {
	// bindings はこの狙い方のキーの束縛
	bindings() []keybind.Binding
	canTarget(world w.World, player, target ecs.Entity) bool
	hitRate(world w.World, player, target ecs.Entity) int
	execute(world w.World, player, target ecs.Entity) error
	// handle は狙い方に固有のアクションを処理する。処理しないアクションなら handled=false を返す
	handle(world w.World, player ecs.Entity, action inputmapper.ActionID) (handled bool, err error)
	// title は情報パネルの見出し
	title(world w.World) string
	// drawSource は情報パネルに、使う武器や物の行を描く
	drawSource(world w.World, player ecs.Entity, drawText func(string))
	// unavailable は狙えない理由を返す。狙えるなら空文字
	unavailable(world w.World, player ecs.Entity) string
	noTarget(world w.World) string
	help(world w.World) string
}

// TargetingState は敵を狙う画面のステート。射撃と投げるで共有する。
// 視界内の狙える敵をTabで巡回し、Enterで実行し、Escapeでキャンセルする
type TargetingState struct {
	es.BaseState[w.World]
	mode           targetMode
	enemies        []ecs.Entity // 視界内の敵一覧
	targetIndex    int          // 現在選択中の敵インデックス
	blinkCounter   int          // カーソル点滅用カウンタ
	cachedHitRate  int          // キャッシュ済み命中率
	cachedDistance float64      // キャッシュ済み距離
}

var _ es.State[w.World] = &TargetingState{}

// NewTargetingState は狙い方を指定して敵を狙う画面を作る
func NewTargetingState(mode targetMode) *TargetingState {
	return &TargetingState{mode: mode}
}

// OnPause はステートが一時停止される際に呼ばれる
func (st *TargetingState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *TargetingState) OnResume(_ w.World) error { return nil }

// OnStart はステートが開始される際に呼ばれる
func (st *TargetingState) OnStart(world w.World) error {
	if err := st.refreshEnemies(world); err != nil {
		return err
	}
	st.updateTargetCache(world)
	return nil
}

// OnStop はステートが終了する際に呼ばれる
func (st *TargetingState) OnStop(_ w.World) error { return nil }

// Update はステートの更新処理
func (st *TargetingState) Update(world w.World) (es.Transition[w.World], error) {
	st.blinkCounter++

	if action, ok := keybind.ReadInput(world, st.mode.bindings()); ok {
		return st.doAction(world, action)
	}

	return st.ConsumeTransition(), nil
}

// targetingBindings は狙う画面に共通の束縛。Tab で標的を切り替え、Enter で実行する。
// Shift+Tab の行を Tab の行より先に置き、Shift 併用を先に判定する
var targetingBindings = []keybind.Binding{
	{Key: ebiten.KeyEscape, Action: inputmapper.ActionCloseMenu},
	{Key: ebiten.KeyTab, Shift: keybind.ShiftRequired, Action: inputmapper.ActionMenuTabPrev},
	{Key: ebiten.KeyTab, Action: inputmapper.ActionMenuTabNext},
	{Key: ebiten.KeyEnter, Action: inputmapper.ActionShoot},
}

// doAction はActionIDを実行する
func (st *TargetingState) doAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
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
			if err := st.mode.execute(world, playerEntity, st.enemies[st.targetIndex]); err != nil {
				return es.Transition[w.World]{}, err
			}
			return es.Transition[w.World]{Type: es.TransPop}, nil
		}

	default:
		playerEntity, err := query.GetPlayerEntity(world)
		if err != nil {
			return es.Transition[w.World]{}, err
		}
		handled, err := st.mode.handle(world, playerEntity, action)
		if err != nil {
			return es.Transition[w.World]{}, err
		}
		if !handled {
			return es.Transition[w.World]{}, fmt.Errorf("unsupported action: %s", action)
		}
		return es.Transition[w.World]{Type: es.TransPop}, nil
	}

	return st.ConsumeTransition(), nil
}

// refreshEnemies は射撃可能な敵一覧を距離順で更新する。
// 視界内の敵から死亡済み・射程外・射線遮断の敵を除外する
func (st *TargetingState) refreshEnemies(world w.World) error {
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
		if st.mode.canTarget(world, playerEntity, e) {
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
func (st *TargetingState) updateTargetCache(world w.World) {
	if len(st.enemies) == 0 {
		return
	}
	playerEntity, err := query.GetPlayerEntity(world)
	if err != nil {
		return
	}
	target := st.enemies[st.targetIndex]
	st.cachedHitRate = st.mode.hitRate(world, playerEntity, target)
	st.cachedDistance = activity.EntityDistance(playerEntity, target, world)
}

// Draw はステートの描画処理
func (st *TargetingState) Draw(world w.World, screen *ebiten.Image) error {
	if len(st.enemies) > 0 {
		if err := st.drawTargetCursor(world, screen); err != nil {
			return err
		}
	}
	return st.drawShootingPanel(world, screen)
}

// drawTargetCursor は選択中の敵にカーソルを描画する
func (st *TargetingState) drawTargetCursor(world w.World, screen *ebiten.Image) error {
	target := st.enemies[st.targetIndex]
	if !world.Components.GridElement.Has(target) {
		return nil
	}
	targetCoord := world.Components.GridElement.Get(target).Coord

	projector, err := render3d.WorldProjector(world)
	if err != nil {
		return err
	}
	corners, ok := projector.TileCorners(targetCoord, render3d.TileTopHeight(world, targetCoord))
	if !ok {
		return nil
	}

	cursorColor := theme.CursorShoot
	if !world.Resources.Config.DisableAnimation {
		cursorColor = hud.ScaleAlpha(cursorColor, 0.6+0.4*math.Sin(float64(st.blinkCounter)*0.15))
	}
	hud.TileFrame(screen, corners, cursorFrameWidth, cursorColor)
	return nil
}

// drawShootingPanel は射撃情報パネルを描画する
func (st *TargetingState) drawShootingPanel(world w.World, screen *ebiten.Image) error {
	face := world.Resources.UIResources.Text.BodyFace

	const panelHeight = 250
	panel := hud.NewInfoPanel(uicore.NewEbitenCanvas(screen), hud.Chrome{}, face, screen.Bounds().Dx(), panelHeight)
	drawText := panel.Line

	playerEntity, err := query.GetPlayerEntity(world)
	if err != nil {
		drawText(query.T(world, "Error: player not found"))
		return err
	}

	drawText(st.mode.title(world))
	panel.Gap(5)
	st.mode.drawSource(world, playerEntity, drawText)
	panel.Gap(5)

	// ターゲット情報
	if msg := st.mode.unavailable(world, playerEntity); msg != "" {
		drawText(msg)
	} else if len(st.enemies) == 0 {
		drawText(st.mode.noTarget(world))
	} else {
		target := st.enemies[st.targetIndex]
		st.drawTargetInfo(world, target, drawText)
	}

	// 操作説明
	panel.SeekBottom(30)
	drawText(st.mode.help(world))

	return nil
}

// drawTargetInfo はターゲット情報を描画する。キャッシュ済みの値を使用する
func (st *TargetingState) drawTargetInfo(world w.World, target ecs.Entity, drawText func(string)) {
	drawText(fmt.Sprintf("%s: %s (%d/%d)", query.T(world, "Target"),
		query.GetEntityName(target, world),
		st.targetIndex+1, len(st.enemies)))

	// HP
	if world.Components.HP.Has(target) {
		hp := world.Components.HP.Get(target)
		drawText(fmt.Sprintf("%s: %d/%d", query.T(world, "HP"), hp.Current, hp.Max))
	}

	drawText(fmt.Sprintf("%s: %d%%", query.T(world, "Hit rate"), st.cachedHitRate))
	drawText(fmt.Sprintf("%s: %.1f", query.T(world, "Distance"), st.cachedDistance))
}

// ShootMode は装備中の遠距離武器で撃つ狙い方。R でリロードする
type ShootMode struct{}

func (ShootMode) bindings() []keybind.Binding {
	return append(slices.Clone(targetingBindings), keybind.Binding{Key: ebiten.KeyR, Action: inputmapper.ActionReload})
}

func (ShootMode) canTarget(world w.World, player, target ecs.Entity) bool {
	return activity.CanShootTarget(player, target, world)
}

func (ShootMode) hitRate(world w.World, player, target ecs.Entity) int {
	return activity.CalculateShootHitRate(player, target, world)
}

func (ShootMode) execute(world w.World, player, target ecs.Entity) error {
	_, err := activity.Execute(activity.NewShootActivity(target), player, world)
	return err
}

func (ShootMode) handle(world w.World, player ecs.Entity, action inputmapper.ActionID) (bool, error) {
	if action != inputmapper.ActionReload {
		return false, nil
	}
	_, err := activity.Execute(activity.NewReloadActivity(player, world), player, world)
	return true, err
}

func (ShootMode) noTarget(world w.World) string { return query.T(world, "No shooting target") }

func (ShootMode) help(world w.World) string {
	return query.T(world, "Tab: Switch  Enter: Fire  R: Reload  Esc: Back")
}

// unavailable は選択中の武器で撃てない理由を返す。撃てるなら空文字
func (ShootMode) unavailable(world w.World, playerEntity ecs.Entity) string {
	weapon, ok := query.SelectedHeld(world, playerEntity)
	if !ok || !world.Components.Fire.Has(weapon) {
		return query.T(world, "No ranged weapon equipped")
	}
	if world.Components.Fire.Get(weapon).Magazine <= 0 {
		return query.T(world, "Not loaded")
	}
	return ""
}

func (ShootMode) title(world w.World) string { return query.T(world, "== Shooting Mode ==") }

// drawSource は選んだ欄の射撃武器と残弾を描く。射撃を持たない物は射撃に使えないので出さない
func (ShootMode) drawSource(world w.World, playerEntity ecs.Entity, drawText func(string)) {
	weapon, ok := query.SelectedHeld(world, playerEntity)
	if !ok || !world.Components.Fire.Has(weapon) {
		drawText(query.T(world, "Weapon: none"))
		return
	}
	fire := world.Components.Fire.Get(weapon)
	drawText(fmt.Sprintf("%s: %s", query.T(world, "Weapon"), query.GetEntityName(weapon, world)))
	drawText(fmt.Sprintf("%s: %d/%d", query.T(world, "Ammo"), fire.Magazine, fire.MagazineSize))
}
