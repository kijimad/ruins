package activity

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/mlange-42/ark/ecs"
)

// ExecuteInteraction は相互作用の種類に応じたアクティビティを実行する。
func ExecuteInteraction(actor ecs.Entity, target ecs.Entity, interaction gc.InteractionKind, world w.World) (*ActionResult, error) {
	config := interaction.Config()

	if err := config.ActivationRange.Valid(); err != nil {
		return nil, fmt.Errorf("invalid ActivationRange: %w", err)
	}
	if err := config.ActivationWay.Valid(); err != nil {
		return nil, fmt.Errorf("invalid ActivationWay: %w", err)
	}

	switch interaction {
	case gc.InteractionPortalNext:
		// 共存方式の下り。同一 State 内 swapTo で現階を退避し、再訪で復元できる
		return executePortal(world, gc.WarpDescendEvent(), "next floor warp state change request error", "portal move")
	case gc.InteractionPortalPrev:
		return executePortal(world, gc.WarpAscendEvent(), "previous floor warp state change request error", "portal move")
	case gc.InteractionDungeonEnter:
		return executeDungeonEnter(target, world)
	case gc.InteractionDoor:
		return executeDoor(actor, target, world)
	case gc.InteractionTalk:
		return executeTalk(actor, target, world)
	case gc.InteractionItem:
		return executeItem(actor, target, world)
	case gc.InteractionItemAll:
		return executeItemAll(actor, world)
	case gc.InteractionStorage:
		return executeStorage(target, world)
	case gc.InteractionMelee:
		return executeMelee(actor, target, world)
	case gc.InteractionDisassemble:
		return executeDisassemble(actor, target, world)
	case gc.InteractionEnterCube:
		// 入る対象のキューブ本体を載せて運ぶ。退場時の戻り先解決に使う
		return executePortal(world, gc.WarpCubeEnterEvent(target), "cube enter state change request error", "enter cube")
	case gc.InteractionExitCube:
		return executePortal(world, gc.WarpCubeExitEvent(), "cube exit state change request error", "exit cube")
	case gc.InteractionPullCube:
		return executePullCube(actor, target, world)
	case gc.InteractionCubePanel:
		return executePortal(world, gc.OpenCubePanelEvent(), "control panel state change request error", "opened control panel")
	case gc.InteractionAuction:
		return executePortal(world, gc.OpenAuctionEvent(target), "auction menu state change request error", "opened shipping station")
	case gc.InteractionIgnite:
		return executeIgnite(target, world)
	case gc.InteractionFeedFuel:
		return executeFeedFuel(target, world)
	}
	// default を置かず exhaustive に全種別を強制する。未知入力は raw/save 由来でありうるので
	// panic せず error で loud に落とす
	return nil, fmt.Errorf("unknown interaction type: %s", interaction)
}

// executePortal は状態機械へ遷移リクエストを投げる。実際の SwapTo や画面遷移は状態機械が担う。
// ポータル・キューブの入退場・コントロールパネルなど、リクエストを1つ出して終わる相互作用が共通で使う。
// failMsg はリクエスト失敗時にエラーへ前置する文脈、successMsg は成功時に ActionResult へ載せる表示。
func executePortal(world w.World, event gc.StateChangeRequest, failMsg, successMsg string) (*ActionResult, error) {
	if err := lifecycle.RequestStateChange(world, event); err != nil {
		return nil, fmt.Errorf("%s: %w", failMsg, err)
	}
	return &ActionResult{Success: true, ActivityName: gc.BehaviorPortal, Message: successMsg}, nil
}

// executeDungeonEnter は遺跡入口の進入先を入口プロップの DungeonEntrance から読み、遺跡進入を要求する。
// 入口ごとに進入先が違うため、イベントに定義名を載せて運ぶ。
func executeDungeonEnter(target ecs.Entity, world w.World) (*ActionResult, error) {
	if !world.Components.DungeonEntrance.Has(target) {
		return nil, fmt.Errorf("dungeon entrance has no destination dungeon definition")
	}
	defName := world.Components.DungeonEntrance.Get(target).DefinitionName
	if err := lifecycle.RequestStateChange(world, gc.WarpDungeonEnterEvent(defName)); err != nil {
		return nil, fmt.Errorf("dungeon entry state change request error: %w", err)
	}
	return &ActionResult{Success: true, ActivityName: gc.BehaviorPortal, Message: "dungeon entry"}, nil
}

// executePullCube はキューブを自分の側へ引く。後退スペースが無いなど引けないときは、
// Validate が理由を gamelog へ出して no-op にする。プレイヤーのできない操作は異常系でない。
func executePullCube(actor ecs.Entity, cube ecs.Entity, world w.World) (*ActionResult, error) {
	return Execute(NewPullActivity(cube, actor, world), actor, world)
}

func executeDoor(actor ecs.Entity, doorEntity ecs.Entity, world w.World) (*ActionResult, error) {
	if !world.Components.Door.Has(doorEntity) {
		return nil, fmt.Errorf("DoorInteraction but no Door component")
	}

	door := world.Components.Door.Get(doorEntity)

	if door.IsOpen {
		return Execute(NewCloseDoorActivity(doorEntity), actor, world)
	}
	return Execute(NewOpenDoorActivity(doorEntity), actor, world)
}

func executeTalk(actor ecs.Entity, npcEntity ecs.Entity, world w.World) (*ActionResult, error) {
	if !world.Components.Dialog.Has(npcEntity) {
		return nil, fmt.Errorf("TalkInteraction but no Dialog component")
	}

	result, err := Execute(NewTalkActivity(npcEntity), actor, world)
	if err != nil {
		return nil, fmt.Errorf("talk action failed: %w", err)
	}

	return result, nil
}

func executeItem(actor ecs.Entity, target ecs.Entity, world w.World) (*ActionResult, error) {
	return Execute(NewPickupStackActivity(world, target), actor, world)
}

func executeItemAll(actor ecs.Entity, world w.World) (*ActionResult, error) {
	if !world.Components.GridElement.Has(actor) {
		return nil, ErrPositionNotFound
	}
	gridElement := world.Components.GridElement.Get(actor)
	return Execute(NewPickupTileActivity(world, gridElement.Coord), actor, world)
}

// executeIgnite は隣接タイルの燃焼物に火をつける。target はそのタイルにある燃料の代表。
// タイルへ campfire を立て、タイル上の燃焼物をすべて残ターン数へ畳み込んで燃やし始める。
// 足元でなく隣接タイルに火が立つのは、足元だと自分が燃えるため。火種の所持は呼び出し側が判定済み。
func executeIgnite(target ecs.Entity, world w.World) (*ActionResult, error) {
	if !world.Components.GridElement.Has(target) {
		return nil, fmt.Errorf("ignite target has no position")
	}
	coord := world.Components.GridElement.Get(target).Coord

	fire, err := lifecycle.SpawnProp(world, "campfire", coord.X, coord.Y)
	if err != nil {
		return nil, fmt.Errorf("spawn campfire: %w", err)
	}

	// 着火する。Burning を付けてから、タイル上の燃焼物をすべて残ターン数へ畳み込む
	world.Components.Burning.Add(fire, &gc.Burning{})
	burned := 0
	for _, f := range fieldFuelOnTile(world, coord) {
		lifecycle.AddFuel(world, fire, f)
		burned++
	}
	if burned == 0 {
		// くべる燃料が無かった。火にならないので後始末する。
		// 呼び出し側が燃料のあるタイルだけを対象にするので通常は起きない
		world.Components.Burning.Remove(fire)
		world.ECS.RemoveEntity(fire)
		return nil, fmt.Errorf("no fuel to ignite at tile")
	}

	return &ActionResult{Success: true, ActivityName: gc.BehaviorIgnite, Message: "lit a fire"}, nil
}

// fieldFuelOnTile は指定タイルの地面にある燃焼物を集めて返す。
// 走査中の構造変更を避けるため、先に集めてから呼び出し側が処理する。火自身は燃料でないので混ざらない
func fieldFuelOnTile(world w.World, coord consts.Coord[consts.Tile]) []ecs.Entity {
	var fuels []ecs.Entity
	q := ecs.NewFilter2[gc.Fuel, gc.LocationOnField](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		if !world.Components.GridElement.Has(e) {
			continue
		}
		if world.Components.GridElement.Get(e).Coord == coord {
			fuels = append(fuels, e)
		}
	}
	return fuels
}

// executeFeedFuel は隣接の火への給油メニューを開く。target は燃料をくべる火。
// 実際のくべる操作はメニューが担い、選んだ燃料を残ターン数へ畳み込む
func executeFeedFuel(fire ecs.Entity, world w.World) (*ActionResult, error) {
	if err := lifecycle.RequestStateChange(world, gc.OpenFeedFuelEvent(fire)); err != nil {
		return nil, fmt.Errorf("feed fuel menu state change request error: %w", err)
	}
	return &ActionResult{Success: true, ActivityName: gc.BehaviorFeedFuel, Message: "opened fuel feeding"}, nil
}

func executeStorage(storageEntity ecs.Entity, world w.World) (*ActionResult, error) {
	if err := lifecycle.RequestStateChange(world, gc.OpenStorageEvent(storageEntity)); err != nil {
		return nil, fmt.Errorf("storage menu state change request error: %w", err)
	}
	return &ActionResult{Success: true, ActivityName: gc.BehaviorStorage, Message: "opened storage"}, nil
}

func executeMelee(actor ecs.Entity, target ecs.Entity, world w.World) (*ActionResult, error) {
	return Execute(NewMeleeActivity(target), actor, world)
}

// executeDisassemble は分解アクティビティを組んで実行する。工具不足や定義なしなどの
// 可否判定は Validate が担い、理由を gamelog へ出して no-op にする。
func executeDisassemble(actor ecs.Entity, target ecs.Entity, world w.World) (*ActionResult, error) {
	return Execute(NewDisassembleActivity(target, actor, world), actor, world)
}
