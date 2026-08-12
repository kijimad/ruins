package raw

import (
	"math/rand/v2"
	"testing"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestRawsForItemGroup はアイテムとアイテムグループだけを持つテスト用 Raws を作る。
func newTestRawsForItemGroup(items []oapi.Item, groups []oapi.ItemGroup) oapi.Raws {
	return oapi.Raws{
		Items:      &items,
		ItemGroups: &groups,
	}
}

// testGroupItems は抽選対象のアイテム。sword は非 stackable、potion は stackable。
func testGroupItems() []oapi.Item {
	stk := oapi.Stackable(true)
	return []oapi.Item{
		{Id: "sword", Name: "sword", Description: "d", SpriteKey: "k", SpriteSheetName: "s", Value: 1},
		{Id: "potion", Name: "potion", Description: "d", SpriteKey: "k", SpriteSheetName: "s", Value: 1, Stackable: &stk},
	}
}

func TestSelectFromItemGroup_distributionは1エントリを個数ぶん返す(t *testing.T) {
	t.Parallel()

	groups := []oapi.ItemGroup{
		{Id: "g", Name: "g", Subtype: oapi.Distribution, Entries: []oapi.ItemGroupEntry{
			{Id: "sword", Weight: 1.0, Pack: "1d1"},
		}},
	}
	raws := newTestRawsForItemGroup(testGroupItems(), groups)
	rng := rand.New(rand.NewPCG(1, 2))

	draws, err := SelectFromItemGroup(raws, "g", rng)
	require.NoError(t, err)
	assert.Equal(t, []DrawnItem{{Name: "sword", Count: 1}}, draws, "単一エントリの distribution はそのアイテムを1個返す")
}

func TestSelectFromItemGroup_非stackableはpackぶん1個ずつに展開する(t *testing.T) {
	t.Parallel()

	groups := []oapi.ItemGroup{
		{Id: "g", Name: "g", Subtype: oapi.Distribution, Entries: []oapi.ItemGroupEntry{
			{Id: "sword", Weight: 1.0, Pack: "3d1"},
		}},
	}
	raws := newTestRawsForItemGroup(testGroupItems(), groups)
	rng := rand.New(rand.NewPCG(1, 2))

	draws, err := SelectFromItemGroup(raws, "g", rng)
	require.NoError(t, err)
	assert.Equal(t, []DrawnItem{{Name: "sword", Count: 1}, {Name: "sword", Count: 1}, {Name: "sword", Count: 1}}, draws,
		"非 stackable は pack ぶん1個ずつのエントリに展開する")
}

func TestSelectFromItemGroup_stackableは個数を1エントリにまとめる(t *testing.T) {
	t.Parallel()

	groups := []oapi.ItemGroup{
		{Id: "g", Name: "g", Subtype: oapi.Distribution, Entries: []oapi.ItemGroupEntry{
			{Id: "potion", Weight: 1.0, Pack: "3d1"},
		}},
	}
	raws := newTestRawsForItemGroup(testGroupItems(), groups)
	rng := rand.New(rand.NewPCG(1, 2))

	draws, err := SelectFromItemGroup(raws, "g", rng)
	require.NoError(t, err)
	assert.Equal(t, []DrawnItem{{Name: "potion", Count: 3}}, draws, "stackable は個数を1エントリにまとめる")
}

func TestSelectFromItemGroup_collectionは確率で独立判定する(t *testing.T) {
	t.Parallel()

	// weight=100 は常に当選、weight=0 は常に落選。当選ぶんだけが返る
	groups := []oapi.ItemGroup{
		{Id: "g", Name: "g", Subtype: oapi.Collection, Entries: []oapi.ItemGroupEntry{
			{Id: "sword", Weight: 100.0, Pack: "1d1"},
			{Id: "potion", Weight: 0.0, Pack: "1d1"},
		}},
	}
	raws := newTestRawsForItemGroup(testGroupItems(), groups)
	rng := rand.New(rand.NewPCG(1, 2))

	draws, err := SelectFromItemGroup(raws, "g", rng)
	require.NoError(t, err)
	assert.Equal(t, []DrawnItem{{Name: "sword", Count: 1}}, draws, "確率100のみ当選し確率0は落選する")
}

func TestSelectFromItemGroup_未存在グループはエラー(t *testing.T) {
	t.Parallel()

	raws := newTestRawsForItemGroup(testGroupItems(), []oapi.ItemGroup{})
	rng := rand.New(rand.NewPCG(1, 2))

	_, err := SelectFromItemGroup(raws, "missing", rng)
	require.ErrorIs(t, err, errItemGroupNotExist)
}

func TestSelectFromItemGroup_未知のSubtypeはエラー(t *testing.T) {
	t.Parallel()

	groups := []oapi.ItemGroup{
		{Id: "g", Name: "g", Subtype: oapi.ItemGroupSubtype("mystery"), Entries: []oapi.ItemGroupEntry{
			{Id: "sword", Weight: 1.0, Pack: "1d1"},
		}},
	}
	raws := newTestRawsForItemGroup(testGroupItems(), groups)
	rng := rand.New(rand.NewPCG(1, 2))

	_, err := SelectFromItemGroup(raws, "g", rng)
	require.ErrorIs(t, err, errUnknownItemGroupSubtype)
}

func TestDrawDistribution_エントリが空なら何も返さない(t *testing.T) {
	t.Parallel()

	raws := newTestRawsForItemGroup(testGroupItems(), nil)
	rng := rand.New(rand.NewPCG(1, 2))

	draws, err := drawDistribution(raws, nil, rng)
	require.NoError(t, err)
	assert.Nil(t, draws)
}

func TestDrawDistribution_選ばれたエントリのIdが空なら何も返さない(t *testing.T) {
	t.Parallel()

	raws := newTestRawsForItemGroup(testGroupItems(), nil)
	rng := rand.New(rand.NewPCG(1, 2))

	draws, err := drawDistribution(raws, []oapi.ItemGroupEntry{{Id: "", Weight: 1.0, Pack: "1d1"}}, rng)
	require.NoError(t, err)
	assert.Nil(t, draws)
}

func TestRollPack_空表記は1個とみなす(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(1, 2))
	count, err := rollPack("", rng)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestRollPack_不正なダイス表記はエラー(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(1, 2))
	_, err := rollPack("oops", rng)
	require.ErrorIs(t, err, errInvalidPackDice)
}

func TestExpandDraw_個数が0以下ならnil(t *testing.T) {
	t.Parallel()

	raws := newTestRawsForItemGroup(testGroupItems(), nil)
	assert.Nil(t, expandDraw(raws, "sword", 0))
}

func TestIsStackableItem(t *testing.T) {
	t.Parallel()

	raws := newTestRawsForItemGroup(testGroupItems(), nil)

	t.Run("未存在アイテムはfalse", func(t *testing.T) {
		t.Parallel()
		assert.False(t, isStackableItem(raws, "存在しないアイテム"))
	})

	t.Run("Stackable未設定はfalse", func(t *testing.T) {
		t.Parallel()
		assert.False(t, isStackableItem(raws, "sword"))
	})

	t.Run("Stackable=trueはtrue", func(t *testing.T) {
		t.Parallel()
		assert.True(t, isStackableItem(raws, "potion"))
	})
}
