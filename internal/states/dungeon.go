package states

import (
	"fmt"
	"math/rand/v2"

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

var (
	baseImage *ebiten.Image // 一番下にある黒背景
)

// DungeonState はダンジョン探索中のゲームステート
type DungeonState struct {
	es.BaseState[w.World]
	Depth int
	// BuilderType は使用するマップビルダーのタイプ（BuilderTypeRandom の場合はランダム選択）
	BuilderType mapplanner.PlannerType
	// DefinitionName はダンジョン定義名。設定されていればOnStartでリソースに反映する
	DefinitionName string
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
		baseImage = ebiten.NewImage(screenWidth, screenHeight)
		baseImage.Fill(theme.ScreenBackground)
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
	// ステージ用シードを生成する
	stageSeed := world.Config.RNG.Uint64()
	stageRNG := rand.New(rand.NewPCG(stageSeed, 0))

	// ビルダータイプを決定
	var builderType mapplanner.PlannerType
	// 最終階層かつBossPlannerTypeが設定されている場合はボスフロアを使用する
	switch {
	case def.BossPlannerType != nil && st.Depth == def.TotalFloors:
		builderType = *def.BossPlannerType
	case st.BuilderType.Name == mapplanner.PlannerTypeRandom.Name:
		var err error
		builderType, err = dungeon.SelectPlanner(def, stageRNG)
		if err != nil {
			return err
		}
	default:
		builderType = st.BuilderType
	}

	// テーブル名と階層をプランナーに渡す。エントリの解決はプランナーが行う
	builderType.EnemyTableName = def.EnemyTableName
	builderType.ItemTableName = def.ItemTableName
	builderType.Depth = st.Depth

	// 計画作成する
	plan, err := mapplanner.Plan(world, consts.MapTileWidth, consts.MapTileHeight, stageSeed, builderType)
	if err != nil {
		return err
	}
	// スポーンする
	level, err := mapspawner.Spawn(world, plan)
	if err != nil {
		return err
	}
	query.GetDungeon(world).Level = level

	// 前フロアのSpatialIndexが残っている可能性があるため無効化する
	// SpatialIndexはTurnPhaseEndでのみ無効化されるが、フロア遷移はTurnPhasePlayer中に
	// 発生するため、古いフロアのデータが残り移動不能になることがある
	query.InvalidateSpatialIndex(world)

	// プレイヤーを配置する
	playerPos, err := plan.GetPlayerStartPosition()
	if err != nil {
		return err
	}
	if err := lifecycle.MovePlayerToPosition(world, playerPos.X, playerPos.Y); err != nil {
		return err
	}

	// フロア移動時に探索済みマップをリセット
	query.GetDungeon(world).ExploredTiles = make(map[gc.GridElement]bool)

	// 新しい階のために視界キャッシュをクリアする
	if vs, ok := world.Updaters[(&gs.VisionSystem{}).String()]; ok {
		vs.(*gs.VisionSystem).ClearCaches()
	}

	// ダンジョンタイトルエフェクト用エンティティを作成する
	screenW, screenH := world.Resources.GetScreenDimensions()
	titleText := def.Name
	if st.Depth > 0 {
		titleText = fmt.Sprintf("%s %dF", def.Name, st.Depth)
	}
	splashFace := world.Resources.UIResources.Text.SplashFontFace
	titleEffect := gc.NewSplashTextEffect(titleText, splashFace, screenW, screenH)
	titleEntity := world.World.NewEntity()
	world.Components.VisualEffect.Add(titleEntity, &gc.VisualEffects{
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

// OnStop はステートが停止される際に呼ばれる
func (st *DungeonState) OnStop(world w.World) error {
	// Ark はクエリ反復中ワールドをロックするため、削除対象を集めてから削除する
	var toRemove []ecs.Entity
	spriteRenderQuery := ecs.NewFilter1[gc.SpriteRender](world.World).
		Without(ecs.C[gc.Player](), ecs.C[gc.SquadMember](), ecs.C[gc.LocationInBackpack](), ecs.C[gc.LocationEquipped]()).Query()
	for spriteRenderQuery.Next() {
		toRemove = append(toRemove, spriteRenderQuery.Entity())
	}
	gridElementQuery := ecs.NewFilter1[gc.GridElement](world.World).
		Without(ecs.C[gc.Player](), ecs.C[gc.SquadMember]()).Query()
	for gridElementQuery.Next() {
		toRemove = append(toRemove, gridElementQuery.Entity())
	}
	for _, entity := range toRemove {
		if world.World.Alive(entity) {
			world.World.RemoveEntity(entity)
		}
	}

	// 未消費のステート遷移リクエストを破棄
	lifecycle.ConsumeStateChange(world)

	// 視界キャッシュをクリア
	if vs, ok := world.Updaters[(&gs.VisionSystem{}).String()]; ok {
		vs.(*gs.VisionSystem).ClearCaches()
	}
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
	screen.DrawImage(baseImage, nil)

	for _, renderer := range []w.Renderer{
		&gs.RenderSpriteSystem{},
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
	playerDeadQuery := ecs.NewFilter2[gc.Player, gc.Dead](world.World).Query()
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

	switch req.Kind {
	case gc.EventShowDialog:
		// SpeakerEntityからNameを取得
		if !world.Components.Name.Has(req.SpeakerEntity) {
			return es.Transition[w.World]{}, fmt.Errorf("speaker entity does not have Name component")
		}
		nameComp := world.Components.Name.Get(req.SpeakerEntity)
		speakerName := nameComp.Name

		// NPCの種類に応じて専用ステートを返す
		switch req.MessageKey {
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
			dialogMessage := messagedata.GetDialogue(req.MessageKey, speakerName)
			return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
				func() (es.State[w.World], error) { return NewMessageState(dialogMessage) },
			}}, nil
		}
	case gc.EventWarpNext:
		// 次のフロアへ遷移する
		nextDepth := query.GetDungeon(world).Depth + 1
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
			NewFadeoutAnimationState(NewDungeonState(nextDepth)),
		}}, nil
	case gc.EventWarpEscape:
		// 精算画面を経由して街へ帰還する
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
			NewFadeoutAnimationState(NewAutoSellState()),
		}}, nil
	case gc.EventOpenDungeonSelect:
		// ダンジョン選択画面を開く
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{NewDungeonSelectState}}, nil
	case gc.EventOpenStorage:
		// 収納メニューを開く
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
			func() (es.State[w.World], error) { return NewStorageMenuState(req.StorageEntity) },
		}}, nil
	default:
		// EventGameClear 等、ここで扱わない種別
		return es.Transition[w.World]{}, fmt.Errorf("未処理のStateChangeRequest: %s", req.Kind)
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
