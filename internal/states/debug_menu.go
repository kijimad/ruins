package states

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	es "github.com/kijimaD/ruins/internal/engine/states"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/messagedata"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/gameaction"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
)

// debugEnterPlanners はデバッグでプランナー単位に生成して試すフロアプランナーの一覧。
// マップ生成の見た目を試す用途なので、遺跡定義でなくプランナー、すなわち大部屋・小部屋などで選ぶ。
var debugEnterPlanners = []mapplanner.PlannerType{
	mapplanner.PlannerTypeBigRoom,
	mapplanner.PlannerTypeSmallRoom,
	mapplanner.PlannerTypeCave,
	mapplanner.PlannerTypeRuins,
	mapplanner.PlannerTypeForest,
}

// NewDebugMenuState はデバッグメニューを選択メニューとして作る
func NewDebugMenuState() (es.State[w.World], error) {
	return NewChoiceMenu(debugMenuChoices), nil
}

// debugMenuChoices はデバッグメニューの選択肢を返す。開発用のスポーンやメッセージ確認をまとめる
func debugMenuChoices(_ w.World) (string, []Choice) {
	debugName := dungeon.DungeonDebug.Name()
	choices := []Choice{
		{Label: "Spawn healing potion (inventory)", Run: popAfter(func(world w.World) error {
			_, err := lifecycle.SpawnBackpackItem(world, "healing_potion", 1)
			return err
		})},
		{Label: "Spawn ray gun (inventory)", Run: popAfter(func(world w.World) error {
			_, err := lifecycle.SpawnBackpackItem(world, "ray_gun", 1)
			return err
		})},
		{Label: "Inflict conditions", Run: popAfter(debugInflictConditions)},
		{Label: "Treat all conditions", Run: popAfter(debugTreatAllConditions)},
		{Label: "Exhaust fatigue", Run: popAfter(debugExhaustFatigue)},
		{Label: "Damage self (-10 HP)", Run: popAfter(debugDamageSelf(10))},
		{Label: "Game over", Run: pushChoice(NewGameOverMessageState)},
		{Label: "Run result (death screen)", Run: func(world w.World) (es.Transition[w.World], error) {
			// 死因に目印の debug を入れて結果画面を確認する
			if s := query.GetRunStats(world); s != nil {
				s.Cause = gc.CauseDebug
			}
			return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{NewRunResultState}}, nil
		}},
		{Label: "Start overworld", Run: func(world w.World) (es.Transition[w.World], error) {
			return es.Transition[w.World]{Type: es.TransReplace, NewStateFuncs: []es.StateFactory[w.World]{newGameOverworldState(world)}}, nil
		}},
	}

	// プランナー単位でデバッグ遺跡へ入る選択肢を平坦に追加する。TransReplace ではなく TransPop で
	// ゲームへ戻し、DungeonState.Update が enterDungeonWith を指定プランナーで通す
	for _, pt := range debugEnterPlanners {
		choices = append(choices, Choice{Label: "Generate debug dungeon " + pt.Name, Run: popAfter(func(world w.World) error {
			return lifecycle.RequestStateChange(world, gc.WarpDungeonEnterWithPlannerEvent(debugName, pt.Name))
		})})
	}

	// 街用NPC・収納箱をスポーン地点の隣に固定配置したデバッグステージへ入る。
	// 街の会話・売買・収納を入ってすぐテストできる
	choices = append(choices, Choice{Label: "Generate debug stage", Run: popAfter(func(world w.World) error {
		return lifecycle.RequestStateChange(world, gc.WarpDungeonEnterWithPlannerEvent(dungeon.DungeonDebugTown.Name(), mapplanner.PlannerTypeDebugTown.Name))
	})})

	choices = append(choices,
		Choice{Label: "Message display test", Run: pushMessage(messagedata.NewSystemMessage("The game was saved automatically.\n\nYour progress has been recorded safely."))},
		Choice{Label: "Item acquisition event", Run: func(world w.World) (es.Transition[w.World], error) {
			for id, count := range map[string]int{"iron": 1, "wooden_stick": 1, "ferrite_core": 2} {
				if err := lifecycle.ChangeStackCount(world, id, count); err != nil {
					return es.Transition[w.World]{}, fmt.Errorf("failed to add item: %w", err)
				}
			}
			md := &messagedata.MessageData{Speaker: ""}
			md.AddText("Found a treasure chest.\n\nObtained iron.\nObtained a wooden stick.\nObtained 2 ferrite cores.\n")
			return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{func() (es.State[w.World], error) { return NewMessageState(md) }}}, nil
		}},
		Choice{Label: "Chained message test", Run: pushMessage(messagedata.NewSystemMessage("Battle begins.").SystemMessage("Sword clashes against sword.").SystemMessage("Victory."))},
		Choice{Label: "Long message test", Run: pushMessage(messagedata.NewSystemMessage("This is a test of a very long message.\n\nThe message window resizes automatically, and we confirm that even long text is displayed properly.\n\nThis test verifies that multi-line text and line breaks are handled correctly, and that the window background and border are drawn properly.\n\nMixed content should also display without any problems.\nLet us check punctuation, symbols, and numbers like 123 as well."))},
		Choice{Label: "Choice branch message test", Run: pushMessage(messagedata.NewDialogMessage("Encountered an enemy.", "").
			WithChoiceMessage("Fight", messagedata.NewSystemMessage("Fought.")).
			WithChoiceMessage("Negotiate", messagedata.NewSystemMessage("Negotiated.")).
			WithChoiceMessage("Flee", messagedata.NewSystemMessage("Fled.")))},
		Choice{Label: "Message with background test", Run: func(_ w.World) (es.Transition[w.World], error) {
			md := messagedata.NewDialogMessage("This is a test of a message with a background.", "System")
			md.BackgroundKey = "hospital1"
			return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{func() (es.State[w.World], error) { return NewMessageState(md) }}}, nil
		}},
		Choice{Label: "Toggle debug display", Run: popAfter(func(world w.World) error {
			world.Resources.Config.ShowMapDebug = !world.Resources.Config.ShowMapDebug
			world.Resources.Config.ShowAIDebug = !world.Resources.Config.ShowAIDebug
			world.Resources.Config.NoEncounter = !world.Resources.Config.NoEncounter
			return nil
		})},
		Choice{Label: "Advance time of day", Run: popAfter(func(world w.World) error {
			// 次の時間帯へ進める。色フィルタは Draw で毎回 GameTime から計算するので即反映されるが、
			// 環境光は vision が移動時しか再計算しないので、明るさを更新するため視界を要求する
			query.GetGameTime(world).AdvanceToNextTimeOfDay()
			query.GetVisionState(world).RequestUpdate()
			return nil
		})},
		Choice{Label: "Advance season", Run: popAfter(func(world w.World) error {
			query.GetGameTime(world).AdvanceToNextSeason()
			return nil
		})},
		Choice{Label: "Opening", Run: pushChoice(NewOpeningState)},
		Choice{Label: "Name input", Run: pushChoice(NewCharacterNamingState)},
		Choice{Label: "Job selection", Run: pushChoice(NewCharacterJobState("Ash"))},
		Choice{Label: "Spawn enemy: fireball (hostile)", Run: stayAfter(func(world w.World) error { return spawnEnemyNearPlayer(world, "fireball") })},
		Choice{Label: "Spawn enemy: moss turtle (neutral)", Run: stayAfter(func(world w.World) error { return spawnEnemyNearPlayer(world, "moss_turtle") })},
		Choice{Label: "Spawn enemy: rat (cowardly)", Run: stayAfter(func(world w.World) error { return spawnEnemyNearPlayer(world, "rat") })},
		Choice{Label: "Spawn enemy: iron sentinel (stationary)", Run: stayAfter(func(world w.World) error { return spawnEnemyNearPlayer(world, "iron_sentinel") })},
		Choice{Label: "Spawn enemy: poison spider (wallHug)", Run: stayAfter(func(world w.World) error { return spawnEnemyNearPlayer(world, "poison_spider") })},
		Choice{Label: "Spawn enemy: slime (swarm)", Run: stayAfter(func(world w.World) error { return spawnEnemyNearPlayer(world, "slime") })},
		Choice{Label: "Spawn enemy: skeleton soldier (patrol)", Run: stayAfter(func(world w.World) error { return spawnEnemyNearPlayer(world, "skeleton_soldier") })},
		Choice{Label: "Spawn enemy: stray dog (territorial)", Run: stayAfter(func(world w.World) error { return spawnEnemyNearPlayer(world, "stray_dog") })},
		Choice{Label: "Spawn prop: moving_stone (PassCost)", Run: stayAfter(func(world w.World) error { return spawnPropNearPlayer(world, "moving_stone") })},
		Choice{Label: "Spawn prop: fire", Run: stayAfter(spawnLitFireNearPlayer)},
		Choice{Label: "Spawn prop: hearth", Run: stayAfter(func(world w.World) error { return spawnPropNearPlayer(world, "hearth") })},
		Choice{Label: "Spawn prop: barrel (destructible)", Run: stayAfter(func(world w.World) error { return spawnPropNearPlayer(world, "barrel") })},
		Choice{Label: "Spawn prop: construction_sign (impassable)", Run: stayAfter(func(world w.World) error { return spawnPropNearPlayer(world, "construction_sign") })},
		Choice{Label: "Spawn prop: wooden crate (storage, with items)", Run: stayAfter(spawnStorageWithItems)},
		Choice{Label: "Component list", Run: pushChoice(NewComponentDebugState)},
		Choice{Label: "Close", Run: func(_ w.World) (es.Transition[w.World], error) {
			return es.Transition[w.World]{Type: es.TransPop}, nil
		}},
	)
	return "", choices
}

// debugInflictConditions はデバッグでプレイヤーへ怪我と病気と低体温をまとめて付ける
func debugInflictConditions(world w.World) error {
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return err
	}
	if !world.Components.HealthStatus.Has(player) {
		return nil
	}
	hs := world.Components.HealthStatus.Get(player)
	set := func(part gc.BodyPart, ct gc.ConditionType, timer float64) {
		hs.Parts[part].SetCondition(gc.HealthCondition{Type: ct, Timer: timer, Severity: gc.TimerToSeverity(timer)})
	}
	set(gc.BodyPartArms, gc.ConditionFracture, 60)
	set(gc.BodyPartArms, gc.ConditionLaceration, 60)
	set(gc.BodyPartTorso, gc.ConditionLiverIllness, 60)
	set(gc.BodyPartWholeBody, gc.ConditionHypothermia, 90)
	return nil
}

// debugExhaustFatigue はデバッグでプレイヤーの疲労を過労まで上げる。入眠をすぐ試すため
func debugExhaustFatigue(world w.World) error {
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return err
	}
	if !world.Components.Fatigue.Has(player) {
		return nil
	}
	fatigue := world.Components.Fatigue.Get(player)
	fatigue.Current = fatigue.Max
	return nil
}

// debugTreatAllConditions はデバッグでプレイヤーの全不調を治療済みにする。回復軌道の確認用。
// 低体温は TemperatureSystem が体温から進めるので TendQuality では治らない
func debugTreatAllConditions(world w.World) error {
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return err
	}
	if !world.Components.HealthStatus.Has(player) {
		return nil
	}
	hs := world.Components.HealthStatus.Get(player)
	for p := range hs.Parts {
		for i := range hs.Parts[p].Conditions {
			// 150% は回復1.5倍の良質な治療。回復軌道に乗る様子を確認する
			hs.Parts[p].Conditions[i].TendQuality = 150
		}
	}
	return nil
}

// debugDamageSelf はデバッグでプレイヤーの HP を削る。HP の自然回復を観察する用
func debugDamageSelf(amount int) func(w.World) error {
	return func(world w.World) error {
		player, err := query.GetPlayerEntity(world)
		if err != nil {
			return err
		}
		gameaction.ApplyDamage(world, player, amount, player)
		return nil
	}
}

// playerGridElement はプレイヤーの GridElement を返す。位置を持たない文脈ではエラーにする。
// GridElement 不在の Get は Ark で panic するため、デバッグスポーンの前に弾く
func playerGridElement(world w.World) (*gc.GridElement, error) {
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return nil, err
	}
	if !world.Components.GridElement.Has(player) {
		return nil, fmt.Errorf("cannot spawn because the player has no position")
	}
	return world.Components.GridElement.Get(player), nil
}

func spawnPropNearPlayer(world w.World, name string) error {
	playerGrid, err := playerGridElement(world)
	if err != nil {
		return err
	}
	_, err = lifecycle.SpawnProp(world, name, playerGrid.X+2, playerGrid.Y)
	return err
}

// spawnLitFireNearPlayer はプレイヤーの隣に燃えている火をスポーンする。
// 本番の着火と同じく fire prop へ Burning を付ける
func spawnLitFireNearPlayer(world w.World) error {
	playerGrid, err := playerGridElement(world)
	if err != nil {
		return err
	}
	fire, err := lifecycle.SpawnProp(world, "fire", playerGrid.X+2, playerGrid.Y)
	if err != nil {
		return err
	}
	world.Components.Burning.Add(fire, &gc.Burning{Remaining: 999})
	return nil
}

// spawnStorageWithItems はプレイヤーの隣にアイテム入り木箱をスポーンする
func spawnStorageWithItems(world w.World) error {
	playerGrid, err := playerGridElement(world)
	if err != nil {
		return err
	}
	storageEntity, err := lifecycle.SpawnProp(world, "wooden_crate", playerGrid.X+2, playerGrid.Y)
	if err != nil {
		return err
	}

	items := []struct {
		name  string
		count int
	}{
		{"healing_potion", 3},
		{"grenade", 1},
		{"torch", 1},
	}
	for _, item := range items {
		if _, err := lifecycle.SpawnStorageItem(world, item.name, item.count, storageEntity); err != nil {
			return fmt.Errorf("failed to spawn storage item: %w", err)
		}
	}
	return nil
}

// spawnEnemyNearPlayer はプレイヤーから少し離れた位置に敵をスポーンする
func spawnEnemyNearPlayer(world w.World, name string) error {
	playerGrid, err := playerGridElement(world)
	if err != nil {
		return err
	}
	_, err = lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: playerGrid.X + 8, Y: playerGrid.Y}, name)
	return err
}
