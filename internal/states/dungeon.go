package states

import (
	"fmt"
	"math/rand/v2"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/config"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/mapspawner"
	"github.com/kijimaD/ruins/internal/messagedata"
	gs "github.com/kijimaD/ruins/internal/systems"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// DungeonState はダンジョン探索中のゲームステート
type DungeonState struct {
	es.BaseState[w.World]
	// baseImage は下に敷く背景
	baseImage *ebiten.Image
	Depth     int
	// BuilderType は使用するマップビルダーのタイプ（BuilderTypeRandom の場合はランダム選択）
	BuilderType mapplanner.PlannerType
	// DefinitionName はダンジョン定義名。設定されていればOnStartでリソースに反映する
	DefinitionName string
	// Resume はセーブからの復帰モード。trueならマップ再生成とプレイヤー再配置を行わず、
	// 復元済みのワールド（地形・エンティティ・プレイヤー位置）をそのまま使う
	Resume bool
}

func (st DungeonState) String() string {
	return "Dungeon"
}

// State interface ================

var _ es.State[w.World] = &DungeonState{}
var _ es.ActionHandler[w.World] = &DungeonState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *DungeonState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *DungeonState) OnResume(_ w.World) error { return nil }

// OnStart はステートが開始される際に呼ばれる
func (st *DungeonState) OnStart(world w.World) error {
	screenWidth := world.Resources.ScreenDimensions.Width
	screenHeight := world.Resources.ScreenDimensions.Height
	if screenWidth > 0 && screenHeight > 0 {
		st.baseImage = ebiten.NewImage(screenWidth, screenHeight)
		st.baseImage.Fill(theme.ScreenBackground)
	}

	query.GetDungeon(world).Depth = st.Depth

	// 設定されていればリソースに反映する
	if st.DefinitionName != "" {
		query.GetDungeon(world).DefinitionName = st.DefinitionName
	}
	// ダンジョン定義を取得する
	def, found := dungeon.GetDungeon(query.GetDungeon(world).DefinitionName)
	if !found {
		return fmt.Errorf("ダンジョン定義が見つかりません: %s", query.GetDungeon(world).DefinitionName)
	}
	// 復帰モードでは再生成せず、復元済みの地形・エンティティ・プレイヤー位置をそのまま使う
	if !st.Resume {
		key := dungeonStageKey(st.Depth)
		playerPos, err := st.spawnFloor(world, st.Depth, def, key)
		if err != nil {
			return err
		}
		// プレイヤーを配置する
		if err := lifecycle.MovePlayerToPosition(world, playerPos); err != nil {
			return err
		}
		// フロア移動時に探索済みマップをリセットし、現ステージを確定する
		resetExploredTiles(world)
		query.GetDungeon(world).CurrentStage = key
	}

	// 前フロア・復元前のSpatialIndexが残っている可能性があるため無効化して作り直す。
	// SpatialIndexはTurnPhaseEndでのみ無効化されるが、フロア遷移はTurnPhasePlayer中に
	// 発生するため、古いデータが残り移動不能になることがある
	query.InvalidateSpatialIndex(world)

	// ダンジョンタイトルエフェクト用エンティティを作成する
	screenW, screenH := world.Resources.GetScreenDimensions()
	titleText := def.Name
	if st.Depth > 0 {
		titleText = fmt.Sprintf("%s %dF", def.Name, st.Depth)
	}
	splashFace := world.Resources.UIResources.Text.SplashFontFace
	titleEffect := gc.NewSplashTextEffect(titleText, splashFace, screenW, screenH)
	titleEntity := world.ECS.NewEntity()
	world.Components.VisualEffects.Add(titleEntity, &gc.VisualEffects{
		Effects: []gc.VisualEffect{titleEffect},
	})

	// 街に帰還した際の全クリア判定
	if def.Name == dungeon.DungeonTown.Name {
		gp := query.GetGameProgress(world)
		dungeonNames := dungeon.GetAllDungeonNames()
		if gp.IsAllCleared(dungeonNames) {
			gp.SetEventActive(gc.EventAllCleared)
		}
	}

	return nil
}

// dungeonStageKey は指定深度のダンジョン階を表すステージキーを返す。
// ストアは1回の潜行スコープなので、同一潜行内では深度だけで階を一意に識別できる
func dungeonStageKey(depth int) gc.StageKey {
	return gc.StageKey{Kind: gc.StageKindDungeon, Depth: depth}
}

// spawnFloor は depth のフロアを生成して world に配置し、生成物に StageMember を付ける。
// プレイヤー開始位置を返す。プレイヤー配置・探索リセット・現ステージ更新は呼び出し側が行う
func (st *DungeonState) spawnFloor(world w.World, depth int, def dungeon.Definition, key gc.StageKey) (consts.Coord[consts.Tile], error) {
	var zero consts.Coord[consts.Tile]

	// ステージ用シードを生成する
	stageSeed := world.Config.RNG.Uint64()
	stageRNG := rand.New(rand.NewPCG(stageSeed, 0))

	// ビルダータイプを決定する。最終階層かつBossPlannerTypeがあればボスフロアにする
	var builderType mapplanner.PlannerType
	switch {
	case def.BossPlannerType != nil && depth == def.TotalFloors:
		builderType = *def.BossPlannerType
	case st.BuilderType.Name == mapplanner.PlannerTypeRandom.Name:
		var err error
		builderType, err = dungeon.SelectPlanner(def, stageRNG)
		if err != nil {
			return zero, err
		}
	default:
		builderType = st.BuilderType
	}

	// テーブル名と階層をプランナーに渡す。エントリの解決はプランナーが行う
	builderType.EnemyTableName = def.EnemyTableName
	builderType.ItemTableName = def.ItemTableName
	builderType.Depth = depth

	plan, err := mapplanner.Plan(world, consts.MapTileWidth, consts.MapTileHeight, stageSeed, builderType)
	if err != nil {
		return zero, err
	}
	level, err := mapspawner.Spawn(world, plan)
	if err != nil {
		return zero, err
	}
	query.GetDungeon(world).Level = level

	start, err := plan.GetPlayerStartPosition()
	if err != nil {
		return zero, err
	}

	// 上り階段を開始位置に置く。降りてきた場所が、上りで戻ってくる場所になる。
	// 最上階(floor 1)では上り階段がダンジョン脱出口を兼ねる。町(depth 0)には置かない
	if depth > 0 {
		if _, err := lifecycle.SpawnProp(world, "warp_prev", start.X, start.Y); err != nil {
			return zero, err
		}
	}

	// 生成物(上り階段を含む)をこのステージの一員として識別できるようにする
	tagStageMembers(world, key)

	return start, nil
}

// descend は1つ下の階へ swapTo で移動する。現階を退避し、未訪問なら生成、訪問済みなら再稼働する。
// 生成フローの swapTo 化。TransPush で新ステートを積む旧 WarpNext と違い、同一 State 内で入れ替える
func (st *DungeonState) descend(world w.World) error {
	nextDepth := st.Depth + 1
	target := dungeonStageKey(nextDepth)

	// 生成は swapTo の callback で行う。未訪問のときだけ呼ばれる。
	// def 参照も生成時だけに閉じ、訪問済みの再稼働では不要にする
	var playerPos consts.Coord[consts.Tile]
	var generated bool
	if err := swapTo(world, target, func(world w.World, key gc.StageKey) error {
		def, found := dungeon.GetDungeon(query.GetDungeon(world).DefinitionName)
		if !found {
			return fmt.Errorf("ダンジョン定義が見つかりません: %s", query.GetDungeon(world).DefinitionName)
		}
		var err error
		playerPos, err = st.spawnFloor(world, nextDepth, def, key)
		generated = true
		return err
	}); err != nil {
		return err
	}

	st.Depth = nextDepth
	query.GetDungeon(world).Depth = nextDepth

	// 生成フロアは開始位置(＝上り階段の位置)へ。訪問済みフロアの再訪は
	// そのフロアの上り階段、すなわち降りてくる側の位置へ戻す
	if generated {
		return lifecycle.MovePlayerToPosition(world, playerPos)
	}
	if pos, ok := findPortalPosition(world, gc.InteractionPortalPrev); ok {
		return lifecycle.MovePlayerToPosition(world, pos)
	}
	return nil
}

// findPortalPosition は現ステージの指定種別ポータルプロップの位置を返す。
// 帰還位置の算出に使う。退避中ステージのポータルは ActiveFilter で除外される
func findPortalPosition(world w.World, kind gc.InteractionKind) (consts.Coord[consts.Tile], bool) {
	var pos consts.Coord[consts.Tile]
	found := false
	q := query.ActiveFilter2[gc.Interactable, gc.GridElement](world).Query()
	for q.Next() {
		e := q.Entity()
		if !found && slices.Contains(world.Components.Interactable.Get(e).Interactions, kind) {
			pos = world.Components.GridElement.Get(e).Coord
			found = true
		}
	}
	return pos, found
}

// ascend は1つ上の階へ swapTo で移動する。上り先は必ず訪問済みなので再稼働する。
// プレイヤーは上った先の下り階段、すなわち元々降りてきた場所へ戻す
func (st *DungeonState) ascend(world w.World) error {
	if st.Depth <= 1 {
		// 最上階からの脱出は呼び出し側が扱う。ここへは来ない前提
		return nil
	}
	prevDepth := st.Depth - 1
	target := dungeonStageKey(prevDepth)

	// 上り先は訪問済み前提。未訪問なら生成でなくエラーにする
	if err := swapTo(world, target, func(_ w.World, _ gc.StageKey) error {
		return fmt.Errorf("上り先の階が存在しません: 深度%d", prevDepth)
	}); err != nil {
		return err
	}

	st.Depth = prevDepth
	query.GetDungeon(world).Depth = prevDepth

	// 上った先の下り階段へプレイヤーを戻す
	if pos, ok := findPortalPosition(world, gc.InteractionPortalNext); ok {
		if err := lifecycle.MovePlayerToPosition(world, pos); err != nil {
			return err
		}
	}
	return nil
}

// OnStop はステートが停止される際に呼ばれる
func (st *DungeonState) OnStop(world w.World) error {
	var toRemove []ecs.Entity
	spriteRenderQuery := ecs.NewFilter1[gc.SpriteRender](world.ECS).
		Without(ecs.C[gc.Player](), ecs.C[gc.SquadMember](), ecs.C[gc.LocationInBackpack](), ecs.C[gc.LocationEquipped]()).Query()
	for spriteRenderQuery.Next() {
		toRemove = append(toRemove, spriteRenderQuery.Entity())
	}
	gridElementQuery := ecs.NewFilter1[gc.GridElement](world.ECS).
		Without(ecs.C[gc.Player](), ecs.C[gc.SquadMember]()).Query()
	for gridElementQuery.Next() {
		toRemove = append(toRemove, gridElementQuery.Entity())
	}
	for _, entity := range toRemove {
		if world.ECS.Alive(entity) {
			world.ECS.RemoveEntity(entity)
		}
	}

	// 未消費のステート遷移リクエストを破棄
	lifecycle.ConsumeStateChange(world)

	return nil
}

// Update はゲームステートの更新処理を行う
func (st *DungeonState) Update(world w.World) (es.Transition[w.World], error) {
	// 全クリアイベントの表示
	if query.GetGameProgress(world).IsEventUnseen(gc.EventAllCleared) {
		query.GetGameProgress(world).MarkEventSeen(gc.EventAllCleared)
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
			NewAllClearEventState,
		}}, nil
	}

	// キー入力をActionに変換
	if action, ok := st.HandleInput(world.Config); ok {
		if transition, err := st.DoAction(world, action); err != nil {
			return es.Transition[w.World]{}, err
		} else if transition.Type != es.TransNone {
			return transition, nil
		}
	}

	for _, updater := range []w.Updater{
		&gs.AnimationSystem{},
		&gs.DeadCleanupSystem{},
		&gs.TurnSystem{},
		&gs.VisionSystem{},
		&gs.CameraSystem{},
		&gs.HUDRenderingSystem{},
		&gs.StatsChangedSystem{},
		&gs.WeightDirtySystem{},
		&gs.VisualEffectSystem{},
	} {
		if sys, ok := world.Updaters[updater.String()]; ok {
			if err := sys.Update(world); err != nil {
				return es.Transition[w.World]{}, err
			}
		}
	}

	// プレイヤー死亡チェック
	if st.checkPlayerDeath(world) {
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{NewGameOverMessageState}}, nil
	}

	// ステート遷移リクエストを処理
	transition, err := st.handleStateChangeRequest(world)
	if err != nil {
		return es.Transition[w.World]{}, err
	}
	if transition.Type != es.TransNone {
		return transition, nil
	}

	// BaseStateの共通処理を使用
	return st.ConsumeTransition(), nil
}

// Draw はゲームステートの描画処理を行う
func (st *DungeonState) Draw(world w.World, screen *ebiten.Image) error {
	if st.baseImage != nil {
		screen.DrawImage(st.baseImage, nil)
	}

	for _, renderer := range []w.Renderer{
		&gs.RenderSpriteSystem{},
		&gs.FrostRenderSystem{},
		&gs.HUDRenderingSystem{},
		&gs.VisualEffectSystem{},
	} {
		if sys, ok := world.Renderers[renderer.String()]; ok {
			if err := sys.Draw(world, screen); err != nil {
				return err
			}
		}
	}

	return nil
}

// ================

// HandleInput はキー入力をActionに変換する
func (st *DungeonState) HandleInput(cfg *config.Config) (inputmapper.ActionID, bool) {
	keyboardInput := input.GetSharedKeyboardInput()

	if cfg.Debug && keyboardInput.IsKeyJustPressed(ebiten.KeySlash) {
		return inputmapper.ActionOpenDebugMenu, true
	}

	// ダンジョンメニュー
	if keyboardInput.IsKeyJustPressed(ebiten.KeyM) {
		return inputmapper.ActionOpenDungeonMenu, true
	}

	// インタラクションメニュー
	if keyboardInput.IsKeyJustPressed(ebiten.KeySpace) {
		return inputmapper.ActionOpenInteractionMenu, true
	}

	// 視界情報表示
	if keyboardInput.IsKeyJustPressed(ebiten.KeyX) {
		return inputmapper.ActionOpenFieldInfo, true
	}

	// 射撃モード
	if keyboardInput.IsKeyJustPressed(ebiten.KeyF) {
		return inputmapper.ActionShoot, true
	}

	// 拾うモード
	if keyboardInput.IsKeyJustPressed(ebiten.KeyG) {
		return inputmapper.ActionPickup, true
	}

	// 置くモード
	if keyboardInput.IsKeyJustPressed(ebiten.KeyP) {
		return inputmapper.ActionPlace, true
	}

	// 移動入力
	if action, ok := handleMoveInput(keyboardInput); ok {
		return action, true
	}

	// 待機キー（キーリピート対応）
	if keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyPeriod) {
		return inputmapper.ActionWait, true
	}

	// 相互作用キー（Enter）
	if keyboardInput.IsEnterJustPressedOnce() {
		return inputmapper.ActionInteract, true
	}

	// 武器スロット切り替え（1-5キー）
	if keyboardInput.IsKeyJustPressed(ebiten.Key1) {
		return inputmapper.ActionSwitchWeaponSlot1, true
	}
	if keyboardInput.IsKeyJustPressed(ebiten.Key2) {
		return inputmapper.ActionSwitchWeaponSlot2, true
	}
	if keyboardInput.IsKeyJustPressed(ebiten.Key3) {
		return inputmapper.ActionSwitchWeaponSlot3, true
	}
	if keyboardInput.IsKeyJustPressed(ebiten.Key4) {
		return inputmapper.ActionSwitchWeaponSlot4, true
	}
	if keyboardInput.IsKeyJustPressed(ebiten.Key5) {
		return inputmapper.ActionSwitchWeaponSlot5, true
	}

	return "", false
}

// DoAction はActionを実行する
//
//nolint:gocyclo // 多くのアクションを処理するためswitch文が大きくなる
func (st *DungeonState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	// UI系アクションは常に実行可能
	switch action {
	case inputmapper.ActionOpenDungeonMenu, inputmapper.ActionOpenDebugMenu, inputmapper.ActionOpenInventory, inputmapper.ActionOpenInteractionMenu, inputmapper.ActionOpenFieldInfo, inputmapper.ActionShoot, inputmapper.ActionPickup, inputmapper.ActionPlace:
		// UI系はターンチェック不要
	default:
		// ゲーム内アクション（移動、攻撃など）はターンチェックが必要
		if !query.CanPlayerAct(world) {
			return es.Transition[w.World]{Type: es.TransNone}, nil
		}
		// プレイヤーが継続アクション中は新しいアクションを受け付けない
		if playerEntity, err := query.GetPlayerEntity(world); err == nil && query.HasActivity(world, playerEntity) {
			return es.Transition[w.World]{Type: es.TransNone}, nil
		}
	}

	switch action {
	// UI系アクション（ステート遷移）
	case inputmapper.ActionOpenDungeonMenu:
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{NewDungeonMenuState}}, nil
	case inputmapper.ActionOpenDebugMenu:
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{NewDebugMenuState}}, nil
	case inputmapper.ActionOpenInventory:
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{NewInventoryMenuState}}, nil
	case inputmapper.ActionOpenInteractionMenu:
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
			func() (es.State[w.World], error) { return NewInteractionMenuState(world) },
		}}, nil
	case inputmapper.ActionOpenFieldInfo:
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
			func() (es.State[w.World], error) { return &LookAroundState{}, nil },
		}}, nil
	case inputmapper.ActionShoot:
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
			func() (es.State[w.World], error) { return &ShootingState{}, nil },
		}}, nil
	case inputmapper.ActionPickup:
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
			func() (es.State[w.World], error) { return &PickupState{}, nil },
		}}, nil
	case inputmapper.ActionPlace:
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
			func() (es.State[w.World], error) { return &PlaceState{}, nil },
		}}, nil

	// 移動系アクション
	case inputmapper.ActionMoveNorth:
		if err := activity.ExecuteMoveAction(world, gc.DirectionUp); err != nil {
			return es.Transition[w.World]{Type: es.TransNone}, err
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionMoveSouth:
		if err := activity.ExecuteMoveAction(world, gc.DirectionDown); err != nil {
			return es.Transition[w.World]{Type: es.TransNone}, err
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionMoveEast:
		if err := activity.ExecuteMoveAction(world, gc.DirectionRight); err != nil {
			return es.Transition[w.World]{Type: es.TransNone}, err
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionMoveWest:
		if err := activity.ExecuteMoveAction(world, gc.DirectionLeft); err != nil {
			return es.Transition[w.World]{Type: es.TransNone}, err
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionMoveNorthEast:
		if err := activity.ExecuteMoveAction(world, gc.DirectionUpRight); err != nil {
			return es.Transition[w.World]{Type: es.TransNone}, err
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionMoveNorthWest:
		if err := activity.ExecuteMoveAction(world, gc.DirectionUpLeft); err != nil {
			return es.Transition[w.World]{Type: es.TransNone}, err
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionMoveSouthEast:
		if err := activity.ExecuteMoveAction(world, gc.DirectionDownRight); err != nil {
			return es.Transition[w.World]{Type: es.TransNone}, err
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionMoveSouthWest:
		if err := activity.ExecuteMoveAction(world, gc.DirectionDownLeft); err != nil {
			return es.Transition[w.World]{Type: es.TransNone}, err
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionWait:
		if err := activity.ExecuteWaitAction(world); err != nil {
			return es.Transition[w.World]{Type: es.TransNone}, err
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil

	// 相互作用系アクション
	case inputmapper.ActionInteract:
		actions := GetSameTileManualActions(world)
		switch len(actions) {
		case 0:
			// 何もしない
		case 1:
			playerEntity, err := query.GetPlayerEntity(world)
			if err != nil {
				return es.Transition[w.World]{Type: es.TransNone}, err
			}
			if _, err := activity.ExecuteInteraction(playerEntity, actions[0].Target, actions[0].Interaction, world); err != nil {
				return es.Transition[w.World]{Type: es.TransNone}, err
			}
		default:
			return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
				func() (es.State[w.World], error) { return newActionChoiceMenu(actions), nil },
			}}, nil
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil

	// 武器スロット切り替え系アクション
	case inputmapper.ActionSwitchWeaponSlot1:
		st.switchWeaponSlot(world, 1)
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionSwitchWeaponSlot2:
		st.switchWeaponSlot(world, 2)
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionSwitchWeaponSlot3:
		st.switchWeaponSlot(world, 3)
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionSwitchWeaponSlot4:
		st.switchWeaponSlot(world, 4)
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionSwitchWeaponSlot5:
		st.switchWeaponSlot(world, 5)
		return es.Transition[w.World]{Type: es.TransNone}, nil

	default:
		return es.Transition[w.World]{}, fmt.Errorf("未知のアクション: %s", action)
	}
}

// ================

// checkPlayerDeath はプレイヤーの死亡状態をチェックする
func (st *DungeonState) checkPlayerDeath(world w.World) bool {
	playerDead := false
	playerDeadQuery := ecs.NewFilter2[gc.Player, gc.Dead](world.ECS).Query()
	for playerDeadQuery.Next() {
		playerDead = true
	}
	return playerDead
}

// handleStateChangeRequest はステート遷移リクエストを消費し、対応する遷移を返す
func (st *DungeonState) handleStateChangeRequest(world w.World) (es.Transition[w.World], error) {
	req := lifecycle.ConsumeStateChange(world)
	if req == nil {
		return es.Transition[w.World]{Type: es.TransNone}, nil
	}

	switch p := req.Payload.(type) {
	case gc.ShowDialog:
		// SpeakerEntityからNameを取得
		if !world.Components.Name.Has(p.SpeakerEntity) {
			return es.Transition[w.World]{}, fmt.Errorf("speaker entity does not have Name component")
		}
		nameComp := world.Components.Name.Get(p.SpeakerEntity)
		speakerName := nameComp.Name

		// NPCの種類に応じて専用ステートを返す
		switch p.MessageKey {
		case "merchant_greeting":
			return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
				func() (es.State[w.World], error) { return NewMerchantDialogState(speakerName) },
			}}, nil
		case "doctor_greeting":
			return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
				func() (es.State[w.World], error) { return NewDoctorDialogState(speakerName) },
			}}, nil
		case "tavern_keeper_greeting":
			return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
				func() (es.State[w.World], error) { return NewTavernKeeperDialogState(speakerName) },
			}}, nil
		default:
			// 通常の会話はdialoguesから取得
			dialogMessage := messagedata.GetDialogue(p.MessageKey, speakerName)
			return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
				func() (es.State[w.World], error) { return NewMessageState(dialogMessage) },
			}}, nil
		}
	case gc.WarpNext:
		// 次のフロアへ遷移する
		nextDepth := query.GetDungeon(world).Depth + 1
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
			NewFadeoutAnimationState(NewDungeonState(nextDepth)),
		}}, nil
	case gc.WarpDescend:
		// 共存方式の下り。同一 State 内で swapTo する。現階は退避され再訪で復元できる
		if err := st.descend(world); err != nil {
			return es.Transition[w.World]{}, err
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case gc.WarpAscend:
		if st.Depth <= 1 {
			// 最上階からの上りはダンジョン脱出。持ち帰り品はそのまま街へ帰還する
			return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
				NewFadeoutAnimationState(NewTownState()),
			}}, nil
		}
		// 共存方式の上り。上り先は訪問済みなので再稼働する
		if err := st.ascend(world); err != nil {
			return es.Transition[w.World]{}, err
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case gc.OpenDungeonSelect:
		// ダンジョン選択画面を開く
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{NewDungeonSelectState}}, nil
	case gc.OpenStorage:
		// 収納メニューを開く
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
			func() (es.State[w.World], error) { return NewStorageMenuState(p.StorageEntity) },
		}}, nil
	default:
		// GameClear 等、ここで扱わない種別
		return es.Transition[w.World]{}, fmt.Errorf("未処理のStateChangeRequest: %T", req.Payload)
	}
}

// switchWeaponSlot は指定されたスロット番号（1-5）に武器を切り替える
func (st *DungeonState) switchWeaponSlot(world w.World, slotNumber int) {
	query.GetDungeon(world).SelectedWeaponSlot = slotNumber

	// プレイヤーの武器スロット情報を取得してログメッセージを出力
	query.Player(world, func(playerEntity ecs.Entity) {
		weapons := query.GetWeapons(world, playerEntity)
		weaponIndex := slotNumber - 1 // 1-based to 0-based
		weapon := weapons[weaponIndex]

		if weapon != nil {
			// 武器が装備されている場合は武器名を表示
			if nameComp := world.Components.Name.Get(*weapon); nameComp != nil {
				weaponName := nameComp.Name
				gamelog.New(query.GetGameLog(world)).
					ItemName(weaponName).
					Append("を構えた").
					Log()
			}
		}
	})
}

// handleMoveInput は8方向移動のキー入力を処理する
func handleMoveInput(keyboardInput input.KeyboardInput) (inputmapper.ActionID, bool) {
	// Shift押下中は斜め移動モード。2キー同時押しの斜め移動のみ受け付ける。
	// IsKeyPressedWithRepeatは副作用があるため、Shift判定を先に行い不要な呼び出しを避ける
	if keyboardInput.IsKeyPressed(ebiten.KeyShift) {
		return handleShiftDiagonalInput(keyboardInput)
	}

	upPressed := keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyW) || keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyUp)
	downPressed := keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyS) || keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyDown)
	leftPressed := keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyA) || keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyLeft)
	rightPressed := keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyD) || keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyRight)

	if upPressed {
		return inputmapper.ActionMoveNorth, true
	}
	if downPressed {
		return inputmapper.ActionMoveSouth, true
	}
	if leftPressed {
		return inputmapper.ActionMoveWest, true
	}
	if rightPressed {
		return inputmapper.ActionMoveEast, true
	}

	return "", false
}

// handleShiftDiagonalInput はShift押下中の斜め移動入力を処理する。
// 縦軸のIsKeyPressedWithRepeatのみをリピートタイミングの制御に使い、横軸はIsKeyPressedで判定する。
// 両軸のリピートをOR条件にするとリピート頻度が2倍になるため、片軸のみをドライバーにする
func handleShiftDiagonalInput(keyboardInput input.KeyboardInput) (inputmapper.ActionID, bool) {
	upRepeat := keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyW) || keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyUp)
	downRepeat := keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyS) || keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyDown)
	leftHeld := keyboardInput.IsKeyPressed(ebiten.KeyA) || keyboardInput.IsKeyPressed(ebiten.KeyLeft)
	rightHeld := keyboardInput.IsKeyPressed(ebiten.KeyD) || keyboardInput.IsKeyPressed(ebiten.KeyRight)

	if upRepeat && leftHeld {
		return inputmapper.ActionMoveNorthWest, true
	}
	if upRepeat && rightHeld {
		return inputmapper.ActionMoveNorthEast, true
	}
	if downRepeat && leftHeld {
		return inputmapper.ActionMoveSouthWest, true
	}
	if downRepeat && rightHeld {
		return inputmapper.ActionMoveSouthEast, true
	}
	return "", false
}
