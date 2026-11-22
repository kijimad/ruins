package states_test

import (
	"errors"
	"fmt"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/messagedata"
	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/kijimaD/ruins/internal/rawschema"
	gs "github.com/kijimaD/ruins/internal/states"
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/vrt/replay"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"

	"github.com/stretchr/testify/require"
)

// replayStep はリプレイの1手。shot が真なら action を適用した直後のフレームを撮る。
// action のゼロ値は入力なしで1フレーム進める待ちの手で、state を組んだ直後の画は
// 待ち1手に shot を付けて撮る。golden 名はケース名から導出し、1ケースで複数撮るときは
// suffix で区別して TestGolden_<ケース名>_<suffix> になる。単発は suffix を空にする。
// 全タブを撮るときは ActionMenuTabNext を並べ、各手に suffix でタブ名を付ける
type replayStep struct {
	action inputmapper.ActionID
	shot   bool
	suffix string
}

// TestGolden はステートの実描画を本番の MainGame ループで駆動して固定する VRT をまとめて回す。
// 各ケースは build が返すステート列を組み、steps の Action 列で操作し、shot の手の直後の
// フレームを golden と比較する。組んだ直後の静止画も操作後の画も同じリプレイ機構で撮る。
// カーソルの移動先やメニューの開閉が壊れると、遷移のテストが通っても見た目で落ちる。
//
// 撮るのは互いに異なる画だけにする。既存 golden と同一の画を
// 別名で残しても、資産が増えて README のギャラリーが重複するだけで検出力は上がらない
// overworldAtTurn は指定した経過ターンのオーバーワールド実画面を撮るビルド関数を返す。
// 時間帯ごとの日照と色味を狙って GameTime を差し替える。GameTime は world 生成時に1度だけ
// 初期化され、新規ゲームでリセットされないので、ここで設定した値が描画に効く。
func overworldAtTurn(turns consts.Turn) func(world w.World) ([]es.State[w.World], error) {
	return func(world w.World) ([]es.State[w.World], error) {
		s, err := gs.NewOverworldState(
			mapplanner.PlannerTypeOverworldField,
			dungeon.NewOverworldDefinition("オーバーワールド", 0, 30, 20, 3, 1),
			&overworld.NewGameParams{},
		)()
		if gt := query.GetGameTime(world); gt != nil {
			gt.TotalTurns = turns
		}
		return []es.State[w.World]{s}, err
	}
}

// remainsOverworldState は地上を始めた直後に、プレイヤーの左右の土の地面へ跡を置く。開始地点の墜落現場の
// 窪みは土でなく雪が積もらないので、絵の外の土のマスに置く
type remainsOverworldState struct {
	*gs.DungeonState
}

// OnStart は地上を始めてから、跡を置く
func (s remainsOverworldState) OnStart(world w.World) error {
	if err := s.DungeonState.OnStart(world); err != nil {
		return err
	}
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return err
	}
	pc := world.Components.GridElement.Get(player).Coord
	for dx, id := range map[consts.Tile]string{-5: "wood_splinters", 5: "metal_fragments", 6: "stone_rubble"} {
		if _, err := lifecycle.SpawnRemains(world, id, consts.Coord[consts.Tile]{X: pc.X + dx, Y: pc.Y}); err != nil {
			return err
		}
	}
	return nil
}

// indoorOverworldState は地上を始めた直後に、プレイヤーを最も近い屋内の通れるマスへ動かす。
// 開始地点は屋根の無い墜落現場なので、屋根の下の明るさを撮るために使う。撮る場所は地上の生成で決まるので、
// 生成を変えると撮る建物も変わる
type indoorOverworldState struct {
	*gs.DungeonState
}

// OnStart は地上を始めてから、プレイヤーを屋内へ動かす
func (s indoorOverworldState) OnStart(world w.World) error {
	if err := s.DungeonState.OnStart(world); err != nil {
		return err
	}
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return err
	}
	from := world.Components.GridElement.Get(player).Coord
	var best consts.Coord[consts.Tile]
	bestDist := -1
	q := query.ActiveFilter2[gc.GridElement, gc.TileEnvironment](world).Query()
	for q.Next() {
		e := q.Entity()
		if world.Components.BlockPass.Has(e) || world.Components.TileEnvironment.Get(e).Shelter != rawschema.Indoor {
			continue
		}
		c := world.Components.GridElement.Get(e).Coord
		d := int((c.X-from.X)*(c.X-from.X) + (c.Y-from.Y)*(c.Y-from.Y))
		if bestDist < 0 || d < bestDist || (d == bestDist && (c.Y < best.Y || (c.Y == best.Y && c.X < best.X))) {
			best, bestDist = c, d
		}
	}
	if bestDist < 0 {
		return errNoIndoorTile
	}
	return lifecycle.MovePlayerToPosition(world, best)
}

var errNoIndoorTile = errors.New("no indoor tile in the band")

// goldenCases は TestGolden が撮る場面の一覧。build は fixture の error をそのまま返し、サブテストで require する。
// *testing.T を取らないことで golangci-lint の thelper の誤検知を避ける
var goldenCases = []struct {
	name  string
	build func(world w.World) ([]es.State[w.World], error)
	steps []replayStep
}{
	// ================ 組んだ直後の画を待ち1手で撮るケース ================
	{
		name: "MainMenu",
		build: func(w.World) ([]es.State[w.World], error) {
			return []es.State[w.World]{&gs.MainMenuState{}}, nil
		},
		steps: []replayStep{{shot: true}},
	},
	// MainMenu_LoadFailed はセーブを読めずにメインメニューへ戻ったとき、知らせのモーダルを重ねた画を固定する
	{
		name: "MainMenu_LoadFailed",
		build: func(w.World) ([]es.State[w.World], error) {
			s, err := gs.NewMainMenuAfterLoadFailure()
			return []es.State[w.World]{s}, err
		},
		steps: []replayStep{{}, {shot: true}},
	},
	{
		name: "SettingsMenu",
		build: func(w.World) ([]es.State[w.World], error) {
			return []es.State[w.World]{&gs.MainMenuState{}, &gs.SettingsMenuState{}}, nil
		},
		steps: []replayStep{{shot: true}},
	},
	{
		name: "CharacterNaming",
		build: func(w.World) ([]es.State[w.World], error) {
			return []es.State[w.World]{&gs.CharacterNamingState{}}, nil
		},
		steps: []replayStep{{shot: true}},
	},
	{
		name: "CharacterJob",
		build: func(w.World) ([]es.State[w.World], error) {
			s, err := gs.NewCharacterJobState("Ash")()
			return []es.State[w.World]{s}, err
		},
		steps: []replayStep{{shot: true}},
	},
	// OverworldMap は N キーで開く種別俯瞰図の描画を固定する。記号の色表・凡例・現在地マーカー・
	// 荒れ地の文字非重畳を含む描画経路を覆う。実際のプレイどおり下段の世界の上に地図UIを重ねて撮る。
	{
		name: "OverworldMap",
		build: func(w.World) ([]es.State[w.World], error) {
			// 背景は開始チャンクのオーバーワールド。決定的な RunSeed で golden を安定させる
			backdrop, err := gs.NewOverworldState(mapplanner.PlannerTypeOverworldField,
				dungeon.NewOverworldDefinition("オーバーワールド", 0, 30, 20, 3, 1), &overworld.NewGameParams{RunSeed: 42})()
			if err != nil {
				return nil, err
			}
			return []es.State[w.World]{backdrop, &gs.OverworldMapState{}}, nil
		},
		steps: []replayStep{{shot: true}},
	},
	// ItemAction は動詞タブ画面の全タブを撮る。動詞ごとにバックパックのアイテムが
	// 適用可否で絞られる。回復薬は調べる・置く・食べる・使うに出て、読むには出ない
	{
		name: "ItemAction",
		build: func(world w.World) ([]es.State[w.World], error) {
			if _, err := lifecycle.SpawnBackpackItem(world, "healing_potion", 3); err != nil {
				return nil, err
			}
			return []es.State[w.World]{&gs.ItemActionState{}}, nil
		},
		steps: []replayStep{
			{shot: true, suffix: "Inspect"}, // 初期タブ
			{action: inputmapper.ActionMenuTabNext, shot: true, suffix: "Drop"},
			{action: inputmapper.ActionMenuTabNext, shot: true, suffix: "Eat"},
			{action: inputmapper.ActionMenuTabNext, shot: true, suffix: "Read"},
			{action: inputmapper.ActionMenuTabNext, shot: true, suffix: "Use"},
		},
	},
	// ItemAction_VerbJump は動詞キーでのタブジャンプが画面へ反映されることを固定する。
	// 動詞は SetTab で store を直接変えるので、再描画漏れがあると画面が古いままになる経路を撮る。
	{
		name: "ItemAction_VerbJump",
		build: func(world w.World) ([]es.State[w.World], error) {
			if _, err := lifecycle.SpawnBackpackItem(world, "healing_potion", 3); err != nil {
				return nil, err
			}
			return []es.State[w.World]{&gs.ItemActionState{}}, nil
		},
		steps: []replayStep{
			{action: inputmapper.ActionVerbConsume, shot: true}, // 食べるタブへ跳んだ画
		},
	},
	// Character は人物画面の全タブを撮る。装備は編集タブ、以降は読み取り専用の情報タブ。
	// スキルはページ送りとカテゴリ見出しを含む
	{
		name: "Character",
		build: func(w.World) ([]es.State[w.World], error) {
			return []es.State[w.World]{&gs.CharacterState{}}, nil
		},
		steps: []replayStep{
			{shot: true, suffix: "Equipment"}, // 初期タブ
			{action: inputmapper.ActionMenuTabNext, shot: true, suffix: "Abilities"},
			{action: inputmapper.ActionMenuTabNext, shot: true, suffix: "Skills"},
			{action: inputmapper.ActionMenuTabNext, shot: true, suffix: "Effects"},
			{action: inputmapper.ActionMenuTabNext, shot: true, suffix: "Health"},
			{action: inputmapper.ActionMenuTabNext, shot: true, suffix: "Basic"},
		},
	},
	// RunStats は道中の統計テーブル画面を撮る。統計と環境のタブを持つ
	{
		name: "RunStats",
		build: func(world w.World) ([]es.State[w.World], error) {
			// 見栄えのする統計値を仕込む。経過1740ターンで2日目の夜明けに乗る
			if s := query.GetRunStats(world); s != nil {
				s.EnemiesKilled = 12
				s.ItemsScavenged = 8
				s.SalesTotal = 3400
			}
			if gt := query.GetGameTime(world); gt != nil {
				gt.TotalTurns = 1740
			}
			st, err := gs.NewRunStatsState()
			if err != nil {
				return nil, err
			}
			return []es.State[w.World]{st}, nil
		},
		steps: []replayStep{
			{shot: true, suffix: "Table"},
			{action: inputmapper.ActionMenuTabNext, shot: true, suffix: "Environment"}, // 環境タブ。気温・時間帯・季節
		},
	},
	// RunClear はクリアの結果画面を撮る。結果画面はタブなしで統計だけを並べる。死の結果画面は見出しが違うだけなので撮り分けない
	{
		name: "RunClear",
		build: func(world w.World) ([]es.State[w.World], error) {
			if s := query.GetRunStats(world); s != nil {
				s.EnemiesKilled = 12
				s.ItemsScavenged = 8
				s.SalesTotal = 3400
			}
			if gt := query.GetGameTime(world); gt != nil {
				gt.TotalTurns = 1740
			}
			st, err := gs.NewRunClearState()
			if err != nil {
				return nil, err
			}
			return []es.State[w.World]{st}, nil
		},
		steps: []replayStep{{shot: true}},
	},
	{
		name: "CraftMenu",
		build: func(world w.World) ([]es.State[w.World], error) {
			// 回復薬の材料を持たせ、クラフト可能な行にチェックが付く様子を確認する
			if _, err := lifecycle.SpawnBackpackItem(world, "green_herb", 1); err != nil {
				return nil, err
			}
			if _, err := lifecycle.SpawnBackpackItem(world, "yellow_herb", 1); err != nil {
				return nil, err
			}
			return []es.State[w.World]{&gs.CraftMenuState{}}, nil
		},
		steps: []replayStep{
			{shot: true, suffix: "Consumables"}, // 初期タブ
			{action: inputmapper.ActionMenuTabNext, shot: true, suffix: "Weapons"},
			{action: inputmapper.ActionMenuTabNext, shot: true, suffix: "Armor"},
		},
	},
	{
		name: "ShopMenu",
		build: func(w.World) ([]es.State[w.World], error) {
			return []es.State[w.World]{&gs.ShopMenuState{}}, nil
		},
		steps: []replayStep{
			{shot: true, suffix: "Buy"}, // 初期タブ
			{action: inputmapper.ActionMenuTabNext, shot: true, suffix: "Sell"},
		},
	},
	{
		name: "SaveMenu",
		build: func(w.World) ([]es.State[w.World], error) {
			s, err := gs.NewSaveMenuState()
			return []es.State[w.World]{s}, err
		},
		steps: []replayStep{{shot: true}},
	},
	{
		name: "LoadMenu",
		build: func(w.World) ([]es.State[w.World], error) {
			s, err := gs.NewLoadMenuState()
			return []es.State[w.World]{s}, err
		},
		steps: []replayStep{{shot: true}},
	},
	{
		name: "DebugMenu",
		build: func(w.World) ([]es.State[w.World], error) {
			s, err := gs.NewDebugMenuState()
			return []es.State[w.World]{s}, err
		},
		steps: []replayStep{{shot: true}},
	},
	{
		name: "ComponentDebug",
		build: func(w.World) ([]es.State[w.World], error) {
			s, err := gs.NewComponentDebugState()
			return []es.State[w.World]{s}, err
		},
		steps: []replayStep{{shot: true}},
	},
	// LookAround は実際のプレイどおり、3D世界とHUDの上にカーソルと情報パネルを重ねて撮る。
	// カーソルを足元から離した画も撮る。プレイヤーの真下では投影のずれが最小になり、
	// 離した位置でこそカーソル枠が実際のタイルに乗っているかが分かる
	{
		name: "LookAround",
		build: func(w.World) ([]es.State[w.World], error) {
			return []es.State[w.World]{&gs.DungeonState{
				Depth:          1,
				DefinitionName: dungeon.DebugID,
				BuilderType:    mapplanner.LayoutPlannerType("small_room"),
			}, &gs.LookAroundState{}}, nil
		},
		steps: []replayStep{
			{shot: true},
			{action: inputmapper.ActionMoveNorth},
			{action: inputmapper.ActionMoveNorth},
			{action: inputmapper.ActionMoveEast},
			{action: inputmapper.ActionMoveEast, shot: true, suffix: "Away"},
		},
	},
	{
		name: "GameOver",
		build: func(w.World) ([]es.State[w.World], error) {
			s, err := gs.NewGameOverMessageState()
			return []es.State[w.World]{s}, err
		},
		steps: []replayStep{{shot: true}},
	},
	{
		name: "Message",
		build: func(world w.World) ([]es.State[w.World], error) {
			messageData := messagedata.NewDialogMessage(
				query.T(world, "This is a VRT test of the message window.\n\nA message to check the display state."),
				query.T(world, "VRT Test"),
			).WithChoice(
				query.T(world, "Choice 1"), func(_ w.World) error { return nil },
			).WithChoice(
				query.T(world, "Choice 2"), func(_ w.World) error { return nil },
			)
			s, err := gs.NewMessageState(messageData)
			return []es.State[w.World]{s}, err
		},
		steps: []replayStep{{shot: true}},
	},
	// Shooting は実際のプレイどおり、3D世界とHUDの上に照準と射撃パネルを重ねて撮る。
	{
		name: "Shooting",
		build: func(w.World) ([]es.State[w.World], error) {
			return []es.State[w.World]{&gs.DungeonState{
				Depth:          1,
				DefinitionName: dungeon.DebugID,
				BuilderType:    mapplanner.LayoutPlannerType("small_room"),
			}, gs.NewTargetingState(gs.ShootMode{})}, nil
		},
		steps: []replayStep{{shot: true}},
	},
	{
		name: "PersistentMessage",
		build: func(world w.World) ([]es.State[w.World], error) {
			messageData := messagedata.NewDialogMessage(query.T(world, "This is a VRT test of the persistent message."), query.T(world, "Test"))
			return []es.State[w.World]{gs.NewPersistentMessageState(messageData)}, nil
		},
		steps: []replayStep{{shot: true}},
	},
	{
		name: "StorageMenu",
		build: func(world w.World) ([]es.State[w.World], error) {
			storageEntity, err := lifecycle.SpawnProp(world, "wooden_crate", 3, 3)
			if err != nil {
				return nil, err
			}
			if _, err := lifecycle.SpawnStorageItem(world, "healing_potion", 1, storageEntity); err != nil {
				return nil, err
			}
			s, err := gs.NewStorageMenuState(storageEntity)
			return []es.State[w.World]{s}, err
		},
		steps: []replayStep{
			{shot: true, suffix: "Retrieve"}, // 初期タブ
			{action: inputmapper.ActionMenuTabNext, shot: true, suffix: "Store"},
		},
	},
	// FeedFuelMenu は火への給油メニューを固定する。収納やインベントリと同じ item-row 形式で、
	// アイコン・名前の先頭に右寄せの追加ターン数列が並ぶ。プレイヤーの初期バックパックの食料が燃料になる。
	{
		name: "FeedFuelMenu",
		build: func(world w.World) ([]es.State[w.World], error) {
			fire := world.ECS.NewEntity()
			world.Components.Burning.Add(fire, &gc.Burning{Remaining: 300})
			s, err := gs.NewFeedFuelMenuState(gs.FireFuelSink(fire))
			return []es.State[w.World]{s}, err
		},
		steps: []replayStep{{shot: true}},
	},
	// RefillMenu は補充メニューの満たすタブと抜くタブを固定する。減ったランタンは両方で選べ、満タンの懐中電灯は
	// 満たすタブで満タン、抜くタブで電池が無く入れ物が無いという理由の行になる
	{
		name: "RefillMenu",
		build: func(world w.World) ([]es.State[w.World], error) {
			lantern, err := lifecycle.SpawnBackpackItem(world, "hand_lantern", 1)
			if err != nil {
				return nil, err
			}
			world.Components.Charge.Get(lantern).Current = 600
			if _, err := lifecycle.SpawnBackpackItem(world, "flashlight", 1); err != nil {
				return nil, err
			}
			oil, err := lifecycle.SpawnBackpackItem(world, "lamp_oil", 1)
			if err != nil {
				return nil, err
			}
			world.Components.Charge.Get(oil).Current = 700
			s, err := gs.NewRefillMenuState()
			return []es.State[w.World]{s}, err
		},
		steps: []replayStep{
			{shot: true, suffix: "Fill"},
			{action: inputmapper.ActionMenuTabNext, shot: true, suffix: "Drain"},
		},
	},
	// ChoiceMenuMany は共通の選択メニューが多数の選択肢でもモーダルに収まりページ送りすることを覆う。
	{
		name: "ChoiceMenuMany",
		build: func(world w.World) ([]es.State[w.World], error) {
			choices := make([]gs.Choice, 0, 30)
			for i := range 30 {
				choices = append(choices, gs.Choice{Label: query.T(world, "Item %d", i+1)})
			}
			menu := gs.NewChoiceMenu(func(_ w.World) (string, []gs.Choice) { return query.T(world, "Select"), choices })
			return []es.State[w.World]{menu}, nil
		},
		steps: []replayStep{{shot: true}},
	},
	// ChoiceMenuHeaders は共通の選択メニューの見出し行とページ表示なしの短い一覧を覆う。
	{
		name: "ChoiceMenuHeaders",
		build: func(world w.World) ([]es.State[w.World], error) {
			choices := []gs.Choice{
				{Label: query.T(world, "Weapon"), Header: true},
				{Label: query.T(world, "Wooden Sword")},
				{Label: query.T(world, "Ray Gun")},
				{Label: query.T(world, "Armor"), Header: true},
				{Label: query.T(world, "Leather Armor")},
				{Label: query.T(world, "Back")},
			}
			menu := gs.NewChoiceMenu(func(_ w.World) (string, []gs.Choice) { return query.T(world, "Equipment"), choices })
			return []es.State[w.World]{menu}, nil
		},
		steps: []replayStep{{shot: true}},
	},
	// PickupMenuStorage は拾うメニューの、まだ探していない収納・空の収納・アイテムの行の表記を写す
	{
		name: "PickupMenuStorage",
		build: func(world w.World) ([]es.State[w.World], error) {
			choices := []gs.Choice{
				{Label: query.T(world, "Search %s (%s)", query.T(world, "Wooden Crate"), query.T(world, "up"))},
				{Label: query.T(world, "Open %s (empty) (%s)", query.T(world, "Wooden Crate"), query.T(world, "left"))},
				{Label: query.T(world, "Pick up %s (%s)", query.T(world, "Bread"), query.T(world, "here"))},
			}
			menu := gs.NewChoiceMenu(func(_ w.World) (string, []gs.Choice) { return "", choices })
			return []es.State[w.World]{menu}, nil
		},
		steps: []replayStep{{shot: true}},
	},
	// Overworld は新規ゲーム開始直後のオーバーワールド実画面を固定する。
	{
		name: "Overworld",
		build: func(w.World) ([]es.State[w.World], error) {
			s, err := gs.NewOverworldState(
				mapplanner.PlannerTypeOverworldField,
				dungeon.NewOverworldDefinition("オーバーワールド", 0, 30, 20, 3, 1),
				&overworld.NewGameParams{},
			)()
			return []es.State[w.World]{s}, err
		},
		steps: []replayStep{{shot: true}},
	},
	// Overworld_<n>_* は時間帯ごとの地上の見た目を固定する。各時間帯の中心を1枚ずつ撮り、
	// 連続補間が段差にならないことを回帰検知する。区間内の傾斜は単体テストで担保する。
	// 番号は朝を起点にした1日の並び順で、ファイル名とギャラリーが時刻順に並ぶ。
	{name: "Overworld_1_Morning", build: overworldAtTurn(375), steps: []replayStep{{shot: true}}},   // 朝の中心
	{name: "Overworld_2_Day", build: overworldAtTurn(625), steps: []replayStep{{shot: true}}},       // 昼の中心。最も明るく無彩色
	{name: "Overworld_3_Evening", build: overworldAtTurn(875), steps: []replayStep{{shot: true}}},   // 夕の中心。暖色
	{name: "Overworld_4_Night", build: overworldAtTurn(1125), steps: []replayStep{{shot: true}}},    // 夜の中心。寒色で暗い
	{name: "Overworld_5_Midnight", build: overworldAtTurn(1375), steps: []replayStep{{shot: true}}}, // 深夜の中心。最も暗い
	{name: "Overworld_6_Dawn", build: overworldAtTurn(125), steps: []replayStep{{shot: true}}},      // 夜明けの中心。次の朝へ続く
	// Overworld_Snowfall は秋の終わりの17日目の昼を撮る。その日の平均気温が0℃で、屋根の無い土の地面の半分がまだらに雪に覆われる
	{name: "Overworld_Snowfall", build: overworldAtTurn(16*1500 + 625), steps: []replayStep{{shot: true}}},
	// Overworld_Winter は冬の底の25日目の昼を撮る。屋根の無い土の地面の全てが雪に覆われる
	{name: "Overworld_Winter", build: overworldAtTurn(24*1500 + 625), steps: []replayStep{{shot: true}}},
	// Overworld_Winter_Remains は冬の底の昼に、雪の積もった土の上へ置いた跡を撮る。跡は雪の上に描かれる
	{
		name: "Overworld_Winter_Remains",
		build: func(world w.World) ([]es.State[w.World], error) {
			states, err := overworldAtTurn(24*1500 + 625)(world)
			if err != nil {
				return nil, err
			}
			ds, ok := states[0].(*gs.DungeonState)
			if !ok {
				return nil, fmt.Errorf("unexpected state type %T", states[0])
			}
			return []es.State[w.World]{remainsOverworldState{ds}}, nil
		},
		steps: []replayStep{{shot: true}},
	},
	// Overworld_Indoor は昼に建物の中に立つ画面を撮る。屋根の下は日照を減らして受け、屋外より暗い
	{
		name: "Overworld_Indoor",
		build: func(world w.World) ([]es.State[w.World], error) {
			// RunSeed 6 は開始チャンクのすぐ西が市街地になる
			s, err := gs.NewOverworldState(mapplanner.PlannerTypeOverworldField, dungeon.DungeonOverworld, &overworld.NewGameParams{RunSeed: 6})()
			if err != nil {
				return nil, err
			}
			ds, ok := s.(*gs.DungeonState)
			if !ok {
				return nil, fmt.Errorf("unexpected state type %T", s)
			}
			query.GetGameTime(world).TotalTurns = 625
			return []es.State[w.World]{indoorOverworldState{ds}}, nil
		},
		steps: []replayStep{{shot: true}},
	},
	// Dungeon は遺跡へ入った直後のダンジョン実画面を固定する。
	// プレイヤーは上り階段の上に湧く実スポーンのまま撮る。
	{
		name: "Dungeon",
		build: func(w.World) ([]es.State[w.World], error) {
			return []es.State[w.World]{&gs.DungeonState{
				Depth:          1,
				DefinitionName: dungeon.DebugID,
				BuilderType:    mapplanner.LayoutPlannerType("small_room"),
			}}, nil
		},
		steps: []replayStep{{shot: true}},
	},
	// StatusBadges は左下の状態バッジを固定する。空腹と、同じ種類の不調を1つにまとめたバッジを撮る
	{
		name: "StatusBadges",
		build: func(world w.World) ([]es.State[w.World], error) {
			player, err := query.GetPlayerEntity(world)
			if err != nil {
				return nil, err
			}
			world.Components.Hunger.Get(player).Current = 10
			hs := world.Components.HealthStatus.Get(player)
			for _, part := range []gc.BodyPart{gc.BodyPartArms, gc.BodyPartLegs} {
				hs.Parts[part].SetCondition(gc.HealthCondition{Type: gc.ConditionType("fracture"), Timer: 60, Severity: gc.TimerToSeverity(60)})
			}
			return []es.State[w.World]{&gs.DungeonState{
				Depth:          1,
				DefinitionName: dungeon.DebugID,
				BuilderType:    mapplanner.LayoutPlannerType("small_room"),
			}}, nil
		},
		steps: []replayStep{{shot: true}},
	},
	// ================ 操作した結果の画を撮るケース ================
	// 設定メニューのカーソルが Back へ移った画を撮る。開いた直後と閉じたあとの画は
	// SettingsMenu・MainMenu と完全に同一なので撮らない。
	// 閉じたあと1段になることは replay の遷移テストが押さえている
	{
		name: "SettingsMenuClose",
		build: func(w.World) ([]es.State[w.World], error) {
			return []es.State[w.World]{&gs.MainMenuState{}, &gs.SettingsMenuState{}}, nil
		},
		steps: []replayStep{
			{action: inputmapper.ActionMenuDown, shot: true}, // カーソルが Language から Back へ移った画を撮る
			{action: inputmapper.ActionMenuSelect},           // Back を決定して設定メニューを閉じる
		},
	},
	// メインメニューでカーソルが Settings に載った画を撮る。組んだ直後の画は先頭行に
	// カーソルがある状態しか撮れないので、移動後の見た目はここでしか固定できない。
	// push した先の画は SettingsMenu と同一なので撮らない
	{
		name: "MainMenuOpenSettings",
		build: func(w.World) ([]es.State[w.World], error) {
			return []es.State[w.World]{&gs.MainMenuState{}}, nil
		},
		steps: []replayStep{
			{action: inputmapper.ActionMenuDown},             // Start から Load へ
			{action: inputmapper.ActionMenuDown, shot: true}, // Load から Settings へ。移った画を撮る
			{action: inputmapper.ActionMenuSelect},           // Settings を開いて push する
		},
	},
	// ? で開くキー一覧ヘルプの描画を固定する。ヘルプの golden はこの1枚に絞る。
	// 文脈は最もキーが多いダンジョンにし、数字連結や記号の表記崩れまで一覧で検出する
	{
		name: "KeyHelp",
		build: func(w.World) ([]es.State[w.World], error) {
			return []es.State[w.World]{&gs.DungeonState{
				Depth:          1,
				DefinitionName: dungeon.DebugID,
				BuilderType:    mapplanner.LayoutPlannerType("small_room"),
			}}, nil
		},
		steps: []replayStep{
			{action: inputmapper.ActionOpenKeyHelp}, // ヘルプを push する。反映は次フレーム
			{shot: true},                            // 押し込まれたヘルプを撮る
		},
	},
	// DungeonMenu は M で開く選択メニューの項目を固定する。クラフトを含む項目の並びを覆う
	{
		name: "DungeonMenu",
		build: func(w.World) ([]es.State[w.World], error) {
			return []es.State[w.World]{&gs.DungeonState{
				Depth:          1,
				DefinitionName: dungeon.DebugID,
				BuilderType:    mapplanner.LayoutPlannerType("small_room"),
			}}, nil
		},
		steps: []replayStep{
			{action: inputmapper.ActionOpenDungeonMenu}, // M でメニューを push する。反映は次フレーム
			{shot: true}, // 押し込まれたメニューを撮る
		},
	},
	// x で開く詳細モーダルの描画を固定する。個数とタイトルバーが無く、性能・性質と説明が
	// 並ぶことを覆う。入力ゲートと overlay 重ねを含む本番経路で撮る
	{
		name: "ItemActionDetail",
		build: func(world w.World) ([]es.State[w.World], error) {
			if _, err := lifecycle.SpawnBackpackItem(world, "healing_potion", 3); err != nil {
				return nil, err
			}
			return []es.State[w.World]{&gs.ItemActionState{}}, nil
		},
		steps: []replayStep{
			{action: inputmapper.ActionOpenItemDetail, shot: true}, // 調べるタブ先頭アイテムの詳細モーダルを開いた画を撮る
		},
	},
	// 装備選択ポップアップが実プレイ経路で開く state 遷移を固定する。装備タブで空の武器
	// スロットを選ぶと開く。手持ち欄1は初期装備の松明で埋まるので1つ下げて空のスロット2で開く。
	// ポップアップの視覚の詳細は widget golden TestGolden_EquipSelect が厳密に固定する
	{
		name: "EquipSelectFlow",
		build: func(world w.World) ([]es.State[w.World], error) {
			if _, err := lifecycle.SpawnBackpackItem(world, "iron_sword", 1); err != nil {
				return nil, err
			}
			if _, err := lifecycle.SpawnBackpackItem(world, "ray_gun", 1); err != nil {
				return nil, err
			}
			return []es.State[w.World]{&gs.CharacterState{}}, nil
		},
		steps: []replayStep{
			{action: inputmapper.ActionMenuDown},   // 手持ち欄1から空のスロット2へ
			{action: inputmapper.ActionMenuSelect}, // 空スロットで装備選択を push する
			{shot: true},                           // push は次フレーム反映なので待ち手で撮る
		},
	},
	// EquipCompare は装備済みスロットの装備選択から詳細を開き、現装備と候補を2枚並べて
	// 比較する画を撮る。手持ち欄1は初期装備で埋まるので、そこで装備選択を開き、
	// 先頭の「外す」を1つ飛ばした候補へカーソルを移してから詳細を開く
	{
		name: "EquipCompare",
		build: func(world w.World) ([]es.State[w.World], error) {
			if _, err := lifecycle.SpawnBackpackItem(world, "claymore", 1); err != nil {
				return nil, err
			}
			return []es.State[w.World]{&gs.CharacterState{}}, nil
		},
		steps: []replayStep{
			{action: inputmapper.ActionMenuSelect}, // 手持ち欄1の初期装備で装備選択を push する
			{},                                     // push した装備選択メニューをシードさせる待ち手
			{shot: true, suffix: "List"},           // 「外す」を先頭に置く候補一覧を撮る
			{action: inputmapper.ActionMenuDown},   // 先頭「外す」から候補クレイモアへ
			{action: inputmapper.ActionOpenItemDetail, shot: true, suffix: "Detail"}, // x で詳細を開き現装備との比較を撮る
		},
	},
}

func TestGolden(t *testing.T) {
	t.Parallel()

	for _, tc := range goldenCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// 各撮影の golden 名をケース名と suffix から先に確定し、重複を弾く。
			// 複数撮るケースは suffix で区別する
			seen := map[string]bool{}
			actions := make([]inputmapper.ActionID, len(tc.steps))
			for i, s := range tc.steps {
				actions[i] = s.action
				if !s.shot {
					continue
				}
				name := "TestGolden_" + tc.name
				if s.suffix != "" {
					name += "_" + s.suffix
				}
				require.False(t, seen[name], "golden 名 %s が重複している。複数撮るなら suffix で区別する", name)
				seen[name] = true
			}
			replay.PlayScenario(t,
				func(world w.World) []es.State[w.World] {
					built, err := tc.build(world)
					require.NoError(t, err)
					return built
				},
				actions,
				func(frame int, _ w.World, shot vrt.Frame) {
					// フレーム f は steps[f] の action を適用した直後の画。カーソル移動や
					// タブ送りは同一フレームで効くのでその手で撮る。state を push する手は
					// 次フレームで反映されるので、末尾に待ち手を足してそこで撮る。
					// 末尾の settle フレームには対応する step が無いので撮らない
					if frame >= len(tc.steps) || !tc.steps[frame].shot {
						return
					}
					name := "TestGolden_" + tc.name
					if suffix := tc.steps[frame].suffix; suffix != "" {
						name += "_" + suffix
					}
					shot.AssertGolden(t, name)
				},
			)
		})
	}
}
