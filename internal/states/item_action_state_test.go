package states

import (
	"fmt"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"

	"github.com/kijimaD/ruins/internal/activity"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExecRead_スキル不足でもゲームを止めない は、読むのに必要なスキルが足りない本を読もうとしても
// エラーを返さず、メニューを閉じることを固定する。state のエラーは致命化するため、ユーザー入力の
// 検証失敗はログに出して吸収する。
func TestExecRead_スキル不足でもゲームを止めない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "ash")
	require.NoError(t, err)

	// rifle Lv2 が要る本。新規プレイヤーは rifle 0 なので読めない
	book, err := lifecycle.SpawnBackpackItem(world, "army_shooting_manual", 1)
	require.NoError(t, err)

	trans, err := execRead(world, book)
	require.NoError(t, err, "検証失敗でもエラーを返さない")
	assert.Equal(t, es.TransPop, trans.Type, "メニューを閉じる")
	assert.True(t, world.ECS.Alive(book), "読めないので本は消費されない")
}

func TestVerbByAction_直達アクションを動詞へ対応づける(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		action inputmapper.ActionID
		want   verbID
		ok     bool
	}{
		{"調べる", inputmapper.ActionVerbExamine, verbExamine, true},
		{"置く", inputmapper.ActionVerbPlace, verbPlace, true},
		{"食べる", inputmapper.ActionVerbConsume, verbConsume, true},
		{"読む", inputmapper.ActionVerbRead, verbRead, true},
		{"使う", inputmapper.ActionVerbUse, verbUse, true},
		{"投げる", inputmapper.ActionVerbThrow, verbThrow, true},
		{"動詞でないアクションは対応なし", inputmapper.ActionMenuSelect, verbID(""), false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := verbByAction(tt.action)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestVerbList_各動詞が直達キーとアクションを持ち往復する は verbList を単一の真実にした不変条件を守る。
// KeyBindings の束縛表と verbByAction はこの一覧から導くので、動詞を1行足すだけで直達が効く。
// キーやアクションを欠く行や、アクションが自身へ戻らない不整合を検知し、silent failure を防ぐ
func TestVerbList_各動詞が直達キーとアクションを持ち往復する(t *testing.T) {
	t.Parallel()
	seenKey := map[string]verbID{}
	seenAction := map[inputmapper.ActionID]verbID{}
	for _, v := range verbList {
		require.NotZero(t, v.Key, "動詞 %s は直達キーを持つ", v.ID)
		require.NotEmpty(t, v.Action, "動詞 %s は直達アクションを持つ", v.ID)
		require.NotNil(t, v.Accept, "動詞 %s は Accept を持つ。nil だと Fetch で panic する", v.ID)

		// 直達キーは一意。重複すると束縛表の評価で先勝ちして片方が黙って隠れる
		keyID := fmt.Sprintf("%d/%d", v.Key, v.Shift)
		prevKey, dupKey := seenKey[keyID]
		assert.False(t, dupKey, "直達キーが重複する: %s と %s", prevKey, v.ID)
		seenKey[keyID] = v.ID

		// アクションも一意。verbByAction は先勝ちなので重複すると片方へ届かない
		prevAction, dupAction := seenAction[v.Action]
		assert.False(t, dupAction, "アクションが重複する: %s と %s", prevAction, v.ID)
		seenAction[v.Action] = v.ID

		got, ok := verbByAction(v.Action)
		assert.True(t, ok, "動詞 %s のアクションが対応づく", v.ID)
		assert.Equal(t, v.ID, got, "動詞 %s のアクションは自身の動詞へ戻る", v.ID)
	}
}

func TestVerbTabIndex_動詞の表示順を返す(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 0, verbTabIndex(verbExamine))
	assert.Equal(t, 1, verbTabIndex(verbPlace))
	assert.Equal(t, 2, verbTabIndex(verbConsume))
	assert.Equal(t, 3, verbTabIndex(verbRead))
	assert.Equal(t, 4, verbTabIndex(verbUse))
	// 未知の動詞は先頭タブへ寄せる
	assert.Equal(t, 0, verbTabIndex(verbID("unknown")))
}

func TestVerbs_調べるは実行を持たず詳細モーダルを開く(t *testing.T) {
	t.Parallel()
	for _, v := range verbList {
		if v.ID == verbExamine {
			assert.Nil(t, v.Exec, "調べるは Exec を持たず Enter で詳細モーダルを開く")
			return
		}
	}
	assert.Fail(t, "調べるタブが verbList に存在しない")
}

func TestAcceptEatUse_食べると使うの対象は効果の組で決まる(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		item string
		eat  bool
		use  bool
	}{
		{"パンは栄養を持つので食べる対象", "bread", true, false},
		{"回復薬は回復を持つので使う対象", "healing_potion", false, true},
		{"包帯は治療だけを持つので使う対象", "bandage", false, true},
		{"蜂蜜は栄養と治療を持つので両方の対象", "honey", true, true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "ash")
			require.NoError(t, err)
			entity, err := lifecycle.SpawnBackpackItem(world, tt.item, 1)
			require.NoError(t, err)
			assert.Equal(t, tt.eat, activity.CanEat(world, entity), "食べる対象の判定")
			assert.Equal(t, tt.use, activity.CanUse(world, entity), "使う対象の判定")
		})
	}
}

func TestExecEat_食べるの動詞は食べるの行動を動かす(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "ash")
	require.NoError(t, err)
	world.Components.Hunger.Get(player).Current = 100
	bread, err := lifecycle.SpawnBackpackItem(world, "bread", 1)
	require.NoError(t, err)

	var exec func(w.World, ecs.Entity) (es.Transition[w.World], error)
	for _, v := range verbList {
		if v.ID == verbConsume {
			exec = v.Exec
		}
	}
	require.NotNil(t, exec)
	trans, err := exec(world, bread)
	require.NoError(t, err)
	assert.Equal(t, es.TransPop, trans.Type)
	assert.Greater(t, world.Components.Hunger.Get(player).Current, 100)
}

func TestItemActionBindings_条件が重ならない(t *testing.T) {
	t.Parallel()

	assert.NotPanics(t, func() { keybind.MustMerge(itemActionBindings) })
}

func TestExecThrow_狙う画面へ切り替える(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "ash")
	require.NoError(t, err)
	stone, err := lifecycle.SpawnBackpackItem(world, "stone", 1)
	require.NoError(t, err)

	trans, err := execThrow(world, stone)
	require.NoError(t, err)
	assert.Equal(t, es.TransSwitch, trans.Type)
	require.Len(t, trans.NewStateFuncs, 1)
	state, err := trans.NewStateFuncs[0]()
	require.NoError(t, err)
	targeting, ok := state.(*TargetingState)
	require.True(t, ok)
	assert.Equal(t, throwMode{item: stone, attack: gc.ThrowAttack{Damage: 6, Accuracy: 70}}, targeting.mode)
}

func TestThrowMode_unavailable(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "ash")
	require.NoError(t, err)
	stone, err := lifecycle.SpawnBackpackItem(world, "stone", 1)
	require.NoError(t, err)
	mode := throwMode{item: stone}

	assert.Empty(t, mode.unavailable(world, player))
	lifecycle.MoveMembersToField(world, []ecs.Entity{stone}, consts.Coord[consts.Tile]{X: 2, Y: 2}, player)
	assert.Equal(t, query.T(world, "Nothing to throw"), mode.unavailable(world, player))
}
