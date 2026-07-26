package interior

import (
	"math/rand/v2"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sampleContent はコンビニを模した Content。保証枠・N抽選・1抽選の3 Group を持つ。
func sampleContent() Content {
	return Content{
		ID: "conv_store",
		Groups: []Group{
			{Style: PickEach, Items: []Stuff{
				{Kind: KindFurniture, Ref: "register", Amount: Dice{Bonus: 1}},
				{Kind: KindFurniture, Ref: "gondola", Amount: Dice{Bonus: 3}},
			}},
			{Style: PickN, Pick: 2, Items: []Stuff{
				{Kind: KindLoot, Ref: "snacks", Weight: 3, Amount: Dice{Base: 2, Sides: 4}},
				{Kind: KindLoot, Ref: "drinks", Weight: 2, Amount: Dice{Base: 1, Sides: 4}},
				{Kind: KindLoot, Ref: "bento", Weight: 1, Amount: Dice{Base: 1, Sides: 3}},
			}},
			{Style: PickOne, Items: []Stuff{
				{Kind: KindDecor, Ref: "litter", Amount: Dice{Base: 1, Sides: 3, Bonus: 1}},
				{Kind: KindBeing, Ref: "looter", Chance: 30},
			}},
		},
	}
}

// TestContent_Resolve_同じseedで完全一致する は doc 70 が CI 必須にする決定性の不変条件を固定する。
// 内装は再訪で一致し serde 安全であること。抽選が map や可変乱数器に漏れると再訪ごとに家具が入れ替わる。
func TestContent_Resolve_同じseedで完全一致する(t *testing.T) {
	t.Parallel()

	c := sampleContent()
	first := c.Resolve(42)
	for range 5 {
		assert.Equal(t, first, c.Resolve(42), "同じ seed なら完全一致する")
	}
}

// TestContent_Resolve_seedで結果が変わる は、抽選が実際に seed に依存することを確認する。
// 全 seed で同一なら抽選が効いておらず、決定性テストが自明に通ってしまう退行を検知する。
func TestContent_Resolve_seedで結果が変わる(t *testing.T) {
	t.Parallel()

	c := sampleContent()
	base := c.Resolve(1)
	varied := false
	for s := uint64(2); s < 60 && !varied; s++ {
		if !reflect.DeepEqual(base, c.Resolve(s)) {
			varied = true
		}
	}
	assert.True(t, varied, "seed を変えると結果が変わる seed が存在する")
}

// TestContent_Resolve_PickEachは保証枠を全部置く は、店を店たらしめる保証セットが必ず出ることを固定する。
func TestContent_Resolve_PickEachは保証枠を全部置く(t *testing.T) {
	t.Parallel()

	c := Content{Groups: []Group{{Style: PickEach, Items: []Stuff{
		{Kind: KindFurniture, Ref: "register", Amount: Dice{Bonus: 1}},
		{Kind: KindFurniture, Ref: "gondola", Amount: Dice{Bonus: 3}},
	}}}}
	got := c.Resolve(7)
	require.Len(t, got, 2, "Chance の無い PickEach は全 Item を置く")
	assert.Equal(t, Selection{Kind: KindFurniture, Ref: "register", Count: 1}, got[0])
	assert.Equal(t, Selection{Kind: KindFurniture, Ref: "gondola", Count: 3}, got[1])
}

// TestContent_Resolve_PickNはちょうどN個の別種を置く は、N抽選が重複なく Pick 個を選ぶことを固定する。
func TestContent_Resolve_PickNはちょうどN個の別種を置く(t *testing.T) {
	t.Parallel()

	c := Content{Groups: []Group{{Style: PickN, Pick: 2, Items: []Stuff{
		{Kind: KindLoot, Ref: "a", Amount: Dice{Bonus: 1}},
		{Kind: KindLoot, Ref: "b", Amount: Dice{Bonus: 1}},
		{Kind: KindLoot, Ref: "c", Amount: Dice{Bonus: 1}},
	}}}}
	for s := range uint64(30) {
		got := c.Resolve(s)
		require.Lenf(t, got, 2, "PickN は Pick 個を置く (seed=%d)", s)
		assert.NotEqualf(t, got[0].Ref, got[1].Ref, "重複なく別種を選ぶ (seed=%d)", s)
	}
}

// TestContent_Resolve_PickOneは1つだけ置く は、変種抽選が常に1つを返すことを固定する。
func TestContent_Resolve_PickOneは1つだけ置く(t *testing.T) {
	t.Parallel()

	c := Content{Groups: []Group{{Style: PickOne, Items: []Stuff{
		{Kind: KindDecor, Ref: "a", Amount: Dice{Bonus: 1}},
		{Kind: KindDecor, Ref: "b", Amount: Dice{Bonus: 1}},
	}}}}
	for s := range uint64(30) {
		require.Lenf(t, c.Resolve(s), 1, "PickOne は1つだけ置く (seed=%d)", s)
	}
}

// TestContent_Resolve_Chance0は常に置かれる は、Chance を書かない保証 Stuff が全 seed で出ることを固定する。
func TestContent_Resolve_Chance0は常に置かれる(t *testing.T) {
	t.Parallel()

	c := Content{Groups: []Group{{Style: PickEach, Items: []Stuff{
		{Kind: KindFurniture, Ref: "must", Amount: Dice{Bonus: 1}},
	}}}}
	for s := range uint64(50) {
		got := c.Resolve(s)
		require.Lenf(t, got, 1, "Chance 0 は常置 (seed=%d)", s)
	}
}

// TestDice_roll_定数とダイスの範囲 は、個数抽選の定数表現とダイス範囲を固定する。
func TestDice_roll_定数とダイスの範囲(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(1, 2))
	assert.Equal(t, 5, Dice{Sides: 0, Bonus: 5}.roll(rng), "Sides<=0 は定数 Bonus")
	for range 100 {
		v := Dice{Base: 2, Sides: 4, Bonus: 1}.roll(rng)
		assert.GreaterOrEqual(t, v, 3, "2d4+1 の下限は 2*1+1")
		assert.LessOrEqual(t, v, 9, "2d4+1 の上限は 2*4+1")
	}
}
