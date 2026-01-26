package states

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/config"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/mapspawner"
	"github.com/kijimaD/ruins/internal/messagedata"
	"github.com/kijimaD/ruins/internal/resources"
	gs "github.com/kijimaD/ruins/internal/systems"
	"github.com/kijimaD/ruins/internal/turns"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

var (
	baseImage *ebiten.Image // 一番下にある黒背景
)

// DungeonState はダンジョン探索中のゲームステート
type DungeonState struct {
	es.BaseState[w.World]
	Depth int
	// Seed はマップ生成用のシード値（0の場合はDungeonリソースのシード値を使用）
	Seed uint64
	// BuilderType は使用するマップビルダーのタイプ（BuilderTypeRandom の場合はランダム選択）
	BuilderType mapplanner.PlannerType
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
	// デバッグデータを初期化する。プレイヤーが存在しない場合のみ実行される
	// メインメニューからの新規開始: 実行
	// セーブデータロード後の再開: 無視
	// 階層移動: 無視
	worldhelper.InitNewGameData(world)

	screenWidth := world.Resources.ScreenDimensions.Width
	screenHeight := world.Resources.ScreenDimensions.Height
	if screenWidth > 0 && screenHeight > 0 {
		baseImage = ebiten.NewImage(screenWidth, screenHeight)
		baseImage.Fill(color.Black)
	}

	world.Resources.Dungeon.Depth = st.Depth

	// ターンマネージャーを初期化
	if world.Resources.TurnManager == nil {
		world.Resources.TurnManager = turns.NewTurnManager()
	}

	// 計画作成する
	plan, err := mapplanner.Plan(world, consts.MapTileWidth, consts.MapTileHeight, st.Seed, st.BuilderType)
	if err != nil {
		return err
	}
	// スポーンする
	level, err := mapspawner.Spawn(world, plan)
	if err != nil {
		return err
	}
	world.Resources.Dungeon.Level = level

	// プレイヤー位置を設定する
	var playerX, playerY int
	foundBridgeD := false
	for _, bridge := range plan.Bridges {
		if bridge.BridgeID == "D" {
			playerX = bridge.X
			playerY = bridge.Y
			foundBridgeD = true
			break
		}
	}
	if !foundBridgeD {
		return fmt.Errorf("橋D（入口）が見つかりません")
	}

	// プレイヤーを配置する
	if err := worldhelper.MovePlayerToPosition(world, playerX, playerY); err != nil {
		return err
	}

	// フロア移動時に探索済みマップをリセット
	world.Resources.Dungeon.ExploredTiles = make(map[gc.GridElement]bool)

	// 視界キャッシュをクリア（新しい階のために）
	gs.ClearVisionCaches()

	// NextFloorSeed と SelectedBridgeID をリセット（次の階層で再利用されないように）
	world.Resources.Dungeon.NextFloorSeed = 0
	world.Resources.Dungeon.SelectedBridgeID = ""

	// StateEvent をリセット（橋遷移後のイベントが残らないように）
	world.Resources.Dungeon.SetStateEvent(resources.NoneEvent{})

	return nil
}

// OnStop はステートが停止される際に呼ばれる
func (st *DungeonState) OnStop(world w.World) error {
	world.Manager.Join(
		world.Components.SpriteRender,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		// プレイヤーエンティティ、バックパック内アイテム、装備中アイテムは次のフロアでも必要なので削除しない
		if !entity.HasComponent(world.Components.Player) &&
			!entity.HasComponent(world.Components.ItemLocationInPlayerBackpack) &&
			!entity.HasComponent(world.Components.ItemLocationEquipped) {
			world.Manager.DeleteEntity(entity)
		}
	}))
	world.Manager.Join(
		world.Components.GridElement,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		// プレイヤーエンティティは次のフロアでも必要なので削除しない
		if !entity.HasComponent(world.Components.Player) {
			world.Manager.DeleteEntity(entity)
		}
	}))

	// reset
	world.Resources.Dungeon.SetStateEvent(resources.NoneEvent{})

	// 視界キャッシュをクリア
	gs.ClearVisionCaches()
	return nil
}

// Update はゲームステートの更新処理を行う
func (st *DungeonState) Update(world w.World) (es.Transition[w.World], error) {
	// デバッグ: Update が呼ばれているか確認（橋遷移後の最初の数フレームのみ）

	// キー入力をActionに変換
	if action, ok := st.HandleInput(); ok {
		if transition, err := st.DoAction(world, action); err != nil {
			return es.Transition[w.World]{}, err
		} else if transition.Type != es.TransNone {
			return transition, nil
		}
	}

	for _, updater := range []w.Updater{
		&gs.AnimationSystem{},
		&gs.TurnSystem{},
		&gs.CameraSystem{},
		&gs.HUDRenderingSystem{},
		&gs.EquipmentChangedSystem{},
		&gs.InventoryChangedSystem{},
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

	// StateEvent処理をチェック
	transition, err := st.handleStateEvent(world)
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
		&gs.VisionSystem{},
		&gs.HUDRenderingSystem{},
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
func (st *DungeonState) HandleInput() (inputmapper.ActionID, bool) {
	keyboardInput := input.GetSharedKeyboardInput()

	cfg := config.MustGet()
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

	// 8方向移動キー入力（キーリピート対応）
	// 斜め移動は両方のキーがリピート判定で真になる場合のみ
	upPressed := keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyW) || keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyUp)
	downPressed := keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyS) || keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyDown)
	leftPressed := keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyA) || keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyLeft)
	rightPressed := keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyD) || keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyRight)

	if upPressed && leftPressed {
		return inputmapper.ActionMoveNorthWest, true
	}
	if upPressed && rightPressed {
		return inputmapper.ActionMoveNorthEast, true
	}
	if downPressed && leftPressed {
		return inputmapper.ActionMoveSouthWest, true
	}
	if downPressed && rightPressed {
		return inputmapper.ActionMoveSouthEast, true
	}
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

	// 待機キー（キーリピート対応）
	if keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyPeriod) {
		return inputmapper.ActionWait, true
	}

	// 相互作用キー（Enter）
	if keyboardInput.IsKeyJustPressed(ebiten.KeyEnter) {
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
	case inputmapper.ActionOpenDungeonMenu, inputmapper.ActionOpenDebugMenu, inputmapper.ActionOpenInventory, inputmapper.ActionOpenInteractionMenu, inputmapper.ActionOpenFieldInfo:
		// UI系はターンチェック不要
	default:
		// ゲーム内アクション（移動、攻撃など）はターンチェックが必要
		if world.Resources.TurnManager != nil {
			turnManager := world.Resources.TurnManager.(*turns.TurnManager)
			canAct := turnManager.CanPlayerAct()
			if !canAct {
				return es.Transition[w.World]{Type: es.TransNone}, nil
			}
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
			func() es.State[w.World] { return NewInteractionMenuState(world) },
		}}, nil
	case inputmapper.ActionOpenFieldInfo:
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
			func() es.State[w.World] { return &FieldInfoState{} },
		}}, nil

	// 移動系アクション（World状態を変更）
	case inputmapper.ActionMoveNorth:
		if err := ExecuteMoveAction(world, gc.DirectionUp); err != nil {
			return es.Transition[w.World]{Type: es.TransNone}, err
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionMoveSouth:
		if err := ExecuteMoveAction(world, gc.DirectionDown); err != nil {
			return es.Transition[w.World]{Type: es.TransNone}, err
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionMoveEast:
		if err := ExecuteMoveAction(world, gc.DirectionRight); err != nil {
			return es.Transition[w.World]{Type: es.TransNone}, err
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionMoveWest:
		if err := ExecuteMoveAction(world, gc.DirectionLeft); err != nil {
			return es.Transition[w.World]{Type: es.TransNone}, err
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionMoveNorthEast:
		if err := ExecuteMoveAction(world, gc.DirectionUpRight); err != nil {
			return es.Transition[w.World]{Type: es.TransNone}, err
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionMoveNorthWest:
		if err := ExecuteMoveAction(world, gc.DirectionUpLeft); err != nil {
			return es.Transition[w.World]{Type: es.TransNone}, err
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionMoveSouthEast:
		if err := ExecuteMoveAction(world, gc.DirectionDownRight); err != nil {
			return es.Transition[w.World]{Type: es.TransNone}, err
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionMoveSouthWest:
		if err := ExecuteMoveAction(world, gc.DirectionDownLeft); err != nil {
			return es.Transition[w.World]{Type: es.TransNone}, err
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionWait:
		if err := ExecuteWaitAction(world); err != nil {
			return es.Transition[w.World]{Type: es.TransNone}, err
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil

	// 相互作用系アクション
	case inputmapper.ActionInteract:
		if err := ExecuteEnterAction(world); err != nil {
			return es.Transition[w.World]{Type: es.TransNone}, err
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
	world.Manager.Join(
		world.Components.Player,
		world.Components.Dead,
	).Visit(ecs.Visit(func(_ ecs.Entity) {
		playerDead = true
	}))
	return playerDead
}

// handleStateEvent はStateEventを処理し、対応する遷移を返す
func (st *DungeonState) handleStateEvent(world w.World) (es.Transition[w.World], error) {
	event := world.Resources.Dungeon.ConsumeStateEvent()

	switch e := event.(type) {
	case resources.ShowDialogEvent:
		// SpeakerEntityからNameを取得
		if !e.SpeakerEntity.HasComponent(world.Components.Name) {
			return es.Transition[w.World]{}, fmt.Errorf("speaker entity does not have Name component")
		}
		nameComp := world.Components.Name.Get(e.SpeakerEntity).(*gc.Name)
		speakerName := nameComp.Name

		// NPCの種類に応じて専用ステートを返す
		switch e.MessageKey {
		case "merchant_greeting":
			return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
				func() es.State[w.World] { return NewMerchantDialogState(speakerName) },
			}}, nil
		case "doctor_greeting":
			return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
				func() es.State[w.World] { return NewDoctorDialogState(speakerName) },
			}}, nil
		case "dark_doctor_greeting":
			return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
				func() es.State[w.World] { return NewDarkDoctorDialogState(speakerName, world) },
			}}, nil
		default:
			// 通常の会話はdialoguesから取得
			dialogMessage := messagedata.GetDialogue(e.MessageKey, speakerName)
			return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
				func() es.State[w.World] { return NewMessageState(dialogMessage) },
			}}, nil
		}
	case resources.WarpNextEvent:
		// 橋を渡って次のフロアへ遷移（橋D位置にスポーン）
		nextDepth := world.Resources.Dungeon.Depth + 1
		seed := world.Resources.Dungeon.NextFloorSeed

		return es.Transition[w.World]{
			Type: es.TransSwitch,
			NewStateFuncs: []es.StateFactory[w.World]{
				func() es.State[w.World] {
					return &DungeonState{
						Depth:       nextDepth,
						Seed:        seed,
						BuilderType: mapplanner.PlannerTypeRandom,
					}
				},
			},
		}, nil
	case resources.WarpEscapeEvent:
		return es.Transition[w.World]{Type: es.TransSwitch, NewStateFuncs: []es.StateFactory[w.World]{NewTownState()}}, nil
	case resources.GameClearEvent:
		return es.Transition[w.World]{Type: es.TransSwitch, NewStateFuncs: []es.StateFactory[w.World]{NewDungeonCompleteEndingState}}, nil
	}

	// NoneEventまたは未知のイベントの場合は何もしない
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

// switchWeaponSlot は指定されたスロット番号（1-5）に武器を切り替える
func (st *DungeonState) switchWeaponSlot(world w.World, slotNumber int) {
	world.Resources.SelectedWeaponSlot = slotNumber

	// プレイヤーの武器スロット情報を取得してログメッセージを出力
	worldhelper.QueryPlayer(world, func(playerEntity ecs.Entity) {
		weapons := worldhelper.GetWeapons(world, playerEntity)
		weaponIndex := slotNumber - 1 // 1-based to 0-based
		weapon := weapons[weaponIndex]

		if weapon != nil {
			// 武器が装備されている場合は武器名を表示
			if nameComp := world.Components.Name.Get(*weapon); nameComp != nil {
				weaponName := nameComp.(*gc.Name).Name
				gamelog.New(gamelog.FieldLog).
					ItemName(weaponName).
					Append("を構えた").
					Log()
			}
		}
	})
}
