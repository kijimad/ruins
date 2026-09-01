package states

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/menuloop"
	"github.com/kijimaD/ruins/internal/messagedata"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// 入力・アクション・イベント処理を dungeon.go から分離する。DungeonState のメソッドはこのファイルにも置く。

// dungeonBindings はダンジョン操作の束縛表。モーダル開閉・動詞直達・移動・視点回転・待機・武器切替。
// 全行の条件は互いに素で行順に意味は無い
var dungeonBindings = []keybind.Binding{
	// モーダル開閉
	{Key: ebiten.KeyM, Action: inputmapper.ActionOpenDungeonMenu, Label: "Menu"},
	{Key: ebiten.KeySpace, Action: inputmapper.ActionOpenInteractionMenu, Label: "Interact"},
	// 視界情報表示。調べる X すなわち Shift+x へ X を譲り、フィールド情報は L に置く
	{Key: ebiten.KeyL, Action: inputmapper.ActionOpenFieldInfo, Label: "Field info"},
	{Key: ebiten.KeyN, Action: inputmapper.ActionOpenOverworldMap, Label: "Map"},
	{Key: ebiten.KeyF, Action: inputmapper.ActionShoot, Label: "Shoot"},
	{Key: ebiten.KeyG, Action: inputmapper.ActionPickup, Label: "Pick up"},
	// 動詞タブ画面への直達。調べる X は Shift+x で区別する
	{Key: ebiten.KeyX, Shift: keybind.ShiftRequired, Action: inputmapper.ActionVerbExamine, Label: "Inspect"},
	{Key: ebiten.KeyD, Action: inputmapper.ActionVerbPlace, Label: "Drop"},
	{Key: ebiten.KeyE, Action: inputmapper.ActionVerbConsume, Label: "Eat"},
	{Key: ebiten.KeyR, Action: inputmapper.ActionVerbRead, Label: "Read"},
	{Key: ebiten.KeyT, Action: inputmapper.ActionVerbUse, Label: "Use"},
	{Key: ebiten.KeyS, Action: inputmapper.ActionVerbList, Label: "List"},
	// 移動。WASD は動詞へ空けるため矢印キーのみを使う。斜めへは視点を回してから直進する
	{Key: ebiten.KeyUp, Press: keybind.PressRepeat, Action: inputmapper.ActionMoveNorth, Label: "Move"},
	{Key: ebiten.KeyDown, Press: keybind.PressRepeat, Action: inputmapper.ActionMoveSouth, Label: "Move"},
	{Key: ebiten.KeyLeft, Press: keybind.PressRepeat, Action: inputmapper.ActionMoveWest, Label: "Move"},
	{Key: ebiten.KeyRight, Press: keybind.PressRepeat, Action: inputmapper.ActionMoveEast, Label: "Move"},
	// 視点回転。左の Z を反時計回り、右の C を時計回りにしてキーの左右と回る向きをそろえる。
	// JIS でズレる記号は避ける
	{Key: ebiten.KeyZ, Action: inputmapper.ActionRotateLeft, Label: "Rotate"},
	{Key: ebiten.KeyC, Action: inputmapper.ActionRotateRight, Label: "Rotate"},
	// 待機・足元の相互作用
	{Key: ebiten.KeyPeriod, Press: keybind.PressRepeat, Action: inputmapper.ActionWait, Label: "Wait"},
	{Key: ebiten.KeyEnter, Action: inputmapper.ActionInteract, Label: "Use here"},
	// 武器スロット切り替え。同じラベルの連続行なのでヒントでは 12345 と連結される
	{Key: ebiten.Key1, Action: inputmapper.ActionSwitchWeaponSlot1, Label: "Weapon slot"},
	{Key: ebiten.Key2, Action: inputmapper.ActionSwitchWeaponSlot2, Label: "Weapon slot"},
	{Key: ebiten.Key3, Action: inputmapper.ActionSwitchWeaponSlot3, Label: "Weapon slot"},
	{Key: ebiten.Key4, Action: inputmapper.ActionSwitchWeaponSlot4, Label: "Weapon slot"},
	{Key: ebiten.Key5, Action: inputmapper.ActionSwitchWeaponSlot5, Label: "Weapon slot"},
	// キー一覧ヘルプ
	{Key: ebiten.KeySlash, Shift: keybind.ShiftRequired, Action: inputmapper.ActionOpenKeyHelp, Label: "Help"},
}

// dungeonDebugBindings はデバッグ設定のときだけ有効なキー。本番の表とは分けて合成する。
// Slash はキー一覧ヘルプの Shift+Slash とキーを共有するので Shift 無しに限定する。
// 条件が重なれば MustMerge が構築時に拒否する
var dungeonDebugBindings = []keybind.Binding{
	{Key: ebiten.KeySlash, Shift: keybind.ShiftForbidden, Action: inputmapper.ActionOpenDebugMenu},
}

// dungeonTable と dungeonDebugTable は合成済みの束縛表。デバッグ設定で使う表ごと分け、
// 実行時に表を重ねる階層を持たない。デバッグ行との重なりも構築時に検証される
var (
	dungeonTable      = keybind.MustMerge(dungeonBindings)
	dungeonDebugTable = keybind.MustMerge(dungeonDebugBindings, dungeonBindings)
)

// readAction は1フレームのダンジョン操作を Action として読む。供給源があればそこから読み、
// 再生ドライバがメニューと同じ注入点でダンジョンも駆動できる
func (st *DungeonState) readAction(world w.World) (inputmapper.ActionID, bool) {
	if world.Resources.Config.Debug {
		return keybind.ReadInput(world, dungeonDebugTable)
	}
	return keybind.ReadInput(world, dungeonTable)
}

// moveDir は移動方向を3Dカメラの向きへ回して合わせる。
func (st *DungeonState) moveDir(world w.World, base gc.Direction) gc.Direction {
	return st.three.moveDir(world, base)
}

// DoAction はActionを実行する
//
//nolint:gocyclo // 多くのアクションを処理するためswitch文が大きくなる
func (st *DungeonState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	// UI系アクションは常に実行可能
	switch action {
	case inputmapper.ActionOpenDungeonMenu, inputmapper.ActionOpenDebugMenu, inputmapper.ActionOpenInventory, inputmapper.ActionOpenInteractionMenu, inputmapper.ActionOpenFieldInfo, inputmapper.ActionOpenOverworldMap, inputmapper.ActionOpenKeyHelp, inputmapper.ActionShoot,
		inputmapper.ActionVerbExamine, inputmapper.ActionVerbPlace, inputmapper.ActionVerbConsume, inputmapper.ActionVerbRead, inputmapper.ActionVerbUse, inputmapper.ActionVerbThrow, inputmapper.ActionVerbList,
		inputmapper.ActionRotateLeft, inputmapper.ActionRotateRight:
		// UI系と視点操作はターンを消費しないのでターンチェック不要
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
	case inputmapper.ActionOpenKeyHelp:
		// ダンジョン文脈のキー一覧を開く。表示は束縛表から導出する
		return es.Transition[w.World]{Type: es.TransPush,
			NewStateFuncs: []es.StateFactory[w.World]{menuloop.NewKeyHelpState(dungeonTable)}}, nil
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
	case inputmapper.ActionVerbExamine, inputmapper.ActionVerbPlace, inputmapper.ActionVerbConsume, inputmapper.ActionVerbRead, inputmapper.ActionVerbUse, inputmapper.ActionVerbThrow, inputmapper.ActionVerbList:
		verb, ok := verbByAction(action)
		if !ok {
			return es.Transition[w.World]{Type: es.TransNone}, nil
		}
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{NewItemActionState(verb)}}, nil

	// 移動系アクション
	case inputmapper.ActionMoveNorth:
		if err := activity.ExecuteMoveAction(world, st.moveDir(world, gc.DirectionUp)); err != nil {
			return es.Transition[w.World]{Type: es.TransNone}, err
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionMoveSouth:
		if err := activity.ExecuteMoveAction(world, st.moveDir(world, gc.DirectionDown)); err != nil {
			return es.Transition[w.World]{Type: es.TransNone}, err
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionMoveEast:
		if err := activity.ExecuteMoveAction(world, st.moveDir(world, gc.DirectionRight)); err != nil {
			return es.Transition[w.World]{Type: es.TransNone}, err
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionMoveWest:
		if err := activity.ExecuteMoveAction(world, st.moveDir(world, gc.DirectionLeft)); err != nil {
			return es.Transition[w.World]{Type: es.TransNone}, err
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	// 視点回転。回してから直進すると斜めの world 方向へ動ける
	case inputmapper.ActionRotateLeft:
		st.three.rotate(world, 1)
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionRotateRight:
		st.three.rotate(world, -1)
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
		speakerName := query.GetEntityName(p.SpeakerEntity, world)

		// NPCの種類に応じて専用ステートを返す
		switch p.MessageKey {
		case "merchant_greeting":
			merchant := p.SpeakerEntity
			return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
				func() (es.State[w.World], error) { return NewMerchantDialogState(speakerName, merchant) },
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
	case gc.OpenFeedFuel:
		// 火への給油メニューを開く
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
			func() (es.State[w.World], error) { return NewFeedFuelMenuState(p.FireEntity) },
		}}, nil
	case gc.OpenAuction:
		// 出荷場所のメニューを開く
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
			func() (es.State[w.World], error) { return NewAuctionMenuState(p.StationEntity) },
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
			if world.Components.Name.Has(*weapon) {
				weaponName := query.GetEntityName(*weapon, world)
				gamelog.New(query.GetGameLog(world)).
					Markup(query.T(world, "Readied %s.", gamelog.Tag("item", weaponName))).
					Log()
			}
		}
	})
}
