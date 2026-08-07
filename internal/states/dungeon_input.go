package states

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/config"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/messagedata"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// 入力・アクション・イベント処理を dungeon.go から分離する。DungeonState のメソッドはこのファイルにも置く。

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

	// 視界情報表示。調べる X すなわち Shift+x へ X を譲り、フィールド情報は L へ移す
	if keyboardInput.IsKeyJustPressed(ebiten.KeyL) {
		return inputmapper.ActionOpenFieldInfo, true
	}

	// オーバーワールド地図
	if keyboardInput.IsKeyJustPressed(ebiten.KeyN) {
		return inputmapper.ActionOpenOverworldMap, true
	}

	// 射撃モード
	if keyboardInput.IsKeyJustPressed(ebiten.KeyF) {
		return inputmapper.ActionShoot, true
	}

	// 拾うモード
	if keyboardInput.IsKeyJustPressed(ebiten.KeyG) {
		return inputmapper.ActionPickup, true
	}

	// 動詞タブ画面への直達。調べる X は Shift+x で区別する
	if keyboardInput.IsKeyJustPressed(ebiten.KeyX) && keyboardInput.IsKeyPressed(ebiten.KeyShift) {
		return inputmapper.ActionVerbExamine, true
	}
	if keyboardInput.IsKeyJustPressed(ebiten.KeyD) {
		return inputmapper.ActionVerbPlace, true
	}
	if keyboardInput.IsKeyJustPressed(ebiten.KeyE) {
		return inputmapper.ActionVerbConsume, true
	}
	if keyboardInput.IsKeyJustPressed(ebiten.KeyR) {
		return inputmapper.ActionVerbRead, true
	}
	if keyboardInput.IsKeyJustPressed(ebiten.KeyT) {
		return inputmapper.ActionVerbUse, true
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
	case inputmapper.ActionOpenDungeonMenu, inputmapper.ActionOpenDebugMenu, inputmapper.ActionOpenInventory, inputmapper.ActionOpenInteractionMenu, inputmapper.ActionOpenFieldInfo, inputmapper.ActionOpenOverworldMap, inputmapper.ActionShoot, inputmapper.ActionPickup,
		inputmapper.ActionVerbExamine, inputmapper.ActionVerbPlace, inputmapper.ActionVerbConsume, inputmapper.ActionVerbRead, inputmapper.ActionVerbUse, inputmapper.ActionVerbThrow:
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
		// 所持品は動詞タブ画面の調べるタブで一覧する
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{NewItemActionState(verbExamine)}}, nil
	case inputmapper.ActionOpenInteractionMenu:
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
			func() (es.State[w.World], error) { return NewInteractionMenuState(world) },
		}}, nil
	case inputmapper.ActionOpenFieldInfo:
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
			func() (es.State[w.World], error) { return &LookAroundState{}, nil },
		}}, nil
	case inputmapper.ActionOpenOverworldMap:
		// 地図は今まさにオーバーワールドにいるときだけ開く。ダンジョンやキューブ内部では
		// 帯が現ステージにないので無視する。State 属性の isSeamless でなく現ステージで判定する
		if !query.IsOnOverworld(world) {
			return es.Transition[w.World]{Type: es.TransNone}, nil
		}
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
			func() (es.State[w.World], error) { return &OverworldMapState{}, nil },
		}}, nil
	case inputmapper.ActionShoot:
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
			func() (es.State[w.World], error) { return &ShootingState{}, nil },
		}}, nil
	case inputmapper.ActionPickup:
		// 足元のタイルにある拾得可能物をまとめて拾う。拾う位置は指定しない
		playerEntity, err := query.GetPlayerEntity(world)
		if err != nil {
			return es.Transition[w.World]{Type: es.TransNone}, err
		}
		coord := world.Components.GridElement.Get(playerEntity).Coord
		if _, err := activity.Execute(activity.NewPickupTileActivity(world, coord), playerEntity, world); err != nil {
			return es.Transition[w.World]{Type: es.TransNone}, err
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionVerbExamine, inputmapper.ActionVerbPlace, inputmapper.ActionVerbConsume, inputmapper.ActionVerbRead, inputmapper.ActionVerbUse, inputmapper.ActionVerbThrow:
		verb, ok := verbByAction(action)
		if !ok {
			return es.Transition[w.World]{Type: es.TransNone}, nil
		}
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{NewItemActionState(verb)}}, nil

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
				func() (es.State[w.World], error) { return NewChoiceMenu(sameTileActionChoices), nil },
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
		return es.Transition[w.World]{}, fmt.Errorf("unknown action: %s", action)
	}
}

// ================

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
			dialogMessage := messagedata.GetDialogue(world, p.MessageKey, speakerName)
			return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
				func() (es.State[w.World], error) { return NewMessageState(dialogMessage) },
			}}, nil
		}
	case gc.WarpDescend:
		// 共存方式の下り。同一 State 内で swapTo する。現階は退避され再訪で復元できる
		if err := st.descend(world); err != nil {
			return es.Transition[w.World]{}, err
		}
		return st.completeSwap(world)
	case gc.WarpAscend:
		// 上り階段に結線があればそこへ移動する。浅い階でも遺跡→地上でも同一機構。
		// 全ダンジョンはオーバーワールド入口から入り、生成時に戻り先が結線される。よって
		// handled=false は結線焼き込みの取りこぼし＝バグであり、黙って握り潰さず error で落とす
		handled, err := st.ascend(world)
		if err != nil {
			return es.Transition[w.World]{}, err
		}
		if !handled {
			return es.Transition[w.World]{}, fmt.Errorf("top floor up stairs has no return link")
		}
		return st.completeSwap(world)
	case gc.WarpDungeonEnter:
		// オーバーワールドから遺跡へ入る。同一 State 内 swapTo で帯を退避し遺跡へ切り替える。
		// プランナー名の指定があれば固定して生成する。デバッグのプランナー単位進入で使う
		if p.PlannerName != "" {
			builderType, ok := mapplanner.PlannerTypeByName(p.PlannerName)
			if !ok {
				return es.Transition[w.World]{}, fmt.Errorf("unknown planner name: %s", p.PlannerName)
			}
			// デバッグはプランナーを変えて見た目を試す用途なので、選ぶたびに作り直す
			if err := st.enterDebugPlannerFloor(world, p.DefinitionName, builderType); err != nil {
				return es.Transition[w.World]{}, err
			}
			return st.completeSwap(world)
		}
		if err := st.enterDungeon(world, p.DefinitionName); err != nil {
			return es.Transition[w.World]{}, err
		}
		return st.completeSwap(world)
	case gc.WarpCubeEnter:
		// 移動拠点キューブの内部へ入る。同一 State 内 swapTo でオーバーワールドを退避する
		if err := enterCube(world, p.Cube); err != nil {
			return es.Transition[w.World]{}, err
		}
		return st.completeSwap(world)
	case gc.WarpCubeExit:
		// キューブ内部からオーバーワールドへ戻る
		if err := exitCube(world); err != nil {
			return es.Transition[w.World]{}, err
		}
		return st.completeSwap(world)
	case gc.OpenCubePanel:
		// キューブ内部のコントロールパネルを開く
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
			func() (es.State[w.World], error) { return &CubePanelState{}, nil },
		}}, nil
	case gc.OpenStorage:
		// 収納メニューを開く
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
			func() (es.State[w.World], error) { return NewStorageMenuState(p.StorageEntity) },
		}}, nil
	default:
		// この switch で扱わない種別。未実装の scaffold もここに落ちる
		return es.Transition[w.World]{}, fmt.Errorf("unhandled StateChangeRequest: %T", req.Payload)
	}
}

// switchWeaponSlot は指定されたスロット番号（1-5）に武器を切り替える
func (st *DungeonState) switchWeaponSlot(world w.World, slotNumber int) {
	query.GetWeaponSelection(world).Slot = slotNumber

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
					Markup(query.T(world, "Readied %s.", gamelog.Tag("item", weaponName))).
					Log()
			}
		}
	})
}

// handleMoveInput は8方向移動のキー入力を処理する。移動は矢印キーのみで、英字は動詞コマンドへ空ける
func handleMoveInput(keyboardInput input.KeyboardInput) (inputmapper.ActionID, bool) {
	// Shift押下中は斜め移動モード。2キー同時押しの斜め移動のみ受け付ける。
	// IsKeyPressedWithRepeatは副作用があるため、Shift判定を先に行い不要な呼び出しを避ける
	if keyboardInput.IsKeyPressed(ebiten.KeyShift) {
		return handleShiftDiagonalInput(keyboardInput)
	}

	upPressed := keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyUp)
	downPressed := keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyDown)
	leftPressed := keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyLeft)
	rightPressed := keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyRight)

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

// handleShiftDiagonalInput はShift押下中の斜め移動入力を処理する。移動は矢印キーのみを使う。
// 縦軸のIsKeyPressedWithRepeatのみをリピートタイミングの制御に使い、横軸はIsKeyPressedで判定する。
// 両軸のリピートをOR条件にするとリピート頻度が2倍になるため、片軸のみをドライバーにする
func handleShiftDiagonalInput(keyboardInput input.KeyboardInput) (inputmapper.ActionID, bool) {
	upRepeat := keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyUp)
	downRepeat := keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyDown)
	leftHeld := keyboardInput.IsKeyPressed(ebiten.KeyLeft)
	rightHeld := keyboardInput.IsKeyPressed(ebiten.KeyRight)

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
