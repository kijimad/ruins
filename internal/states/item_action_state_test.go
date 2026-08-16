package states

import (
	"fmt"
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
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
		{"出品", inputmapper.ActionVerbList, verbTag, true},
		{"投げるは未実装なので対応なし", inputmapper.ActionVerbThrow, verbID(""), false},
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
// ExtraInput と verbByAction はこの一覧から導くので、動詞を1行足すだけで直達が効く。
// キーやアクションを欠く行や、アクションが自身へ戻らない不整合を検知し、silent failure を防ぐ
func TestVerbList_各動詞が直達キーとアクションを持ち往復する(t *testing.T) {
	t.Parallel()
	seenKey := map[string]verbID{}
	seenAction := map[inputmapper.ActionID]verbID{}
	for _, v := range verbList {
		require.NotZero(t, v.Key, "動詞 %s は直達キーを持つ", v.ID)
		require.NotEmpty(t, v.Action, "動詞 %s は直達アクションを持つ", v.ID)
		require.NotNil(t, v.Accept, "動詞 %s は Accept を持つ。nil だと Fetch で panic する", v.ID)

		// 直達キーは一意。重複すると ExtraInput のループで先勝ちして片方が黙って隠れる
		keyID := fmt.Sprintf("%d/%t", v.Key, v.Shift)
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
	assert.Equal(t, 5, verbTabIndex(verbTag))
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

// 食べるタブと使うタブの対象は排他で、同一アイテムが両方に現れないことを保証する。
// フラグを増やしたときの取りこぼしを検知する
func TestAcceptConsumeFood_食べる対象と使う対象は排他になる(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		item    string
		consume bool
		use     bool
	}{
		{"回復薬は回復を持つので食べる対象", "healing_potion", true, false},
		{"手榴弾は栄養も回復も持たないので使う対象", "grenade", false, true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			entity, err := lifecycle.SpawnBackpackItem(world, tt.item, 1)
			require.NoError(t, err)
			assert.Equal(t, tt.consume, acceptConsumeFood(world, entity), "食べる対象の判定")
			assert.Equal(t, tt.use, acceptUseTool(world, entity), "使う対象の判定")
			assert.False(t, acceptConsumeFood(world, entity) && acceptUseTool(world, entity), "食べると使うは同時に真にならない")
		})
	}
}
