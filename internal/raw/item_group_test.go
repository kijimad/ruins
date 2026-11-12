package raw

import (
	"math/rand/v2"
	"testing"

	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestRawsForItemGroup はアイテムとアイテムグループとアイテムコレクションだけを持つテスト用 Raws を作る。
func newTestRawsForItemGroup(items []rawschema.Item, groups []rawschema.ItemGroup, collections []rawschema.ItemCollection) rawschema.Raws {
	return rawschema.Raws{
		Items:           &items,
		ItemGroups:      &groups,
		ItemCollections: &collections,
	}
}

// testGroupItems は抽選対象のアイテム。
func testGroupItems() []rawschema.Item {
	return []rawschema.Item{
		{Id: "sword", Name: "sword", Description: "d", SpriteKey: "k", SpriteSheetName: "s", Value: 1},
		{Id: "potion", Name: "potion", Description: "d", SpriteKey: "k", SpriteSheetName: "s", Value: 1},
	}
}

func TestSelectFromItemGroup_グループは1エントリを個数ぶん返す(t *testing.T) {
	t.Parallel()

	groups := []rawschema.ItemGroup{
		{Id: "g", Name: "g", Entries: []rawschema.ItemGroupEntry{
			{Id: "sword", Weight: 1.0, Pack: "1d1"},
		}},
	}
	raws := newTestRawsForItemGroup(testGroupItems(), groups, nil)
	rng := rand.New(rand.NewPCG(1, 2))

	draws, err := SelectFromItemGroup(raws, "g", rng)
	require.NoError(t, err)
	assert.Equal(t, []DrawnItem{{Name: "sword", Count: 1}}, draws, "単一エントリのグループはそのアイテムを1個返す")
}

func TestSelectFromItemGroup_個数を1エントリにまとめる(t *testing.T) {
	t.Parallel()

	groups := []rawschema.ItemGroup{
		{Id: "g", Name: "g", Entries: []rawschema.ItemGroupEntry{
			{Id: "potion", Weight: 1.0, Pack: "3d1"},
		}},
	}
	raws := newTestRawsForItemGroup(testGroupItems(), groups, nil)
	rng := rand.New(rand.NewPCG(1, 2))

	draws, err := SelectFromItemGroup(raws, "g", rng)
	require.NoError(t, err)
	assert.Equal(t, []DrawnItem{{Name: "potion", Count: 3}}, draws, "stackable は個数を1エントリにまとめる")
}

func TestSelectFromItemGroup_コレクションは確率で独立判定する(t *testing.T) {
	t.Parallel()

	// 確率100は常に当選、ごく小さい確率はこのシードで落選する。当選ぶんだけが返る
	collections := []rawschema.ItemCollection{
		{Id: "c", Name: "c", Entries: []rawschema.ItemCollectionEntry{
			{Id: "sword", Chance: 100, Pack: "1d1"},
			{Id: "potion", Chance: 0.001, Pack: "1d1"},
		}},
	}
	raws := newTestRawsForItemGroup(testGroupItems(), nil, collections)
	rng := rand.New(rand.NewPCG(1, 2))

	draws, err := SelectFromItemGroup(raws, "c", rng)
	require.NoError(t, err)
	assert.Equal(t, []DrawnItem{{Name: "sword", Count: 1}}, draws, "確率100だけが当選する")
}

func TestSelectFromItemGroup_未存在グループはエラー(t *testing.T) {
	t.Parallel()

	raws := newTestRawsForItemGroup(testGroupItems(), nil, nil)
	rng := rand.New(rand.NewPCG(1, 2))

	_, err := SelectFromItemGroup(raws, "missing", rng)
	require.ErrorIs(t, err, errItemGroupNotExist)
}

func TestDrawGroup_エントリが空なら何も返さない(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(1, 2))

	draws, err := drawGroup(nil, rng)
	require.NoError(t, err)
	assert.Nil(t, draws)
}

func TestRollPack(t *testing.T) {
	t.Parallel()

	// 空表記も不正表記も ParseDice が弾き、errInvalidPackDice になる
	cases := []struct {
		name string
		pack rawschema.Dice
	}{
		{"空表記はエラー", ""},
		{"不正なダイス表記はエラー", "oops"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rng := rand.New(rand.NewPCG(1, 2))
			_, err := rollPack(tc.pack, rng)
			require.ErrorIs(t, err, errInvalidPackDice)
		})
	}
}

func TestExpandDraw_個数が0以下ならnil(t *testing.T) {
	t.Parallel()

	for _, count := range []int{0, -1} {
		assert.Nil(t, expandDraw("sword", count), "count=%d", count)
	}
}
