// Package archespike は Ark（Arche の後継）+ ark-serde への移行検証（スパイク）。
// 本体コードには一切依存せず、型安全アクセスとワールド丸ごとJSON往復のみを実証する。
package archespike

import (
	"testing"

	arkserde "github.com/mlange-42/ark-serde"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ゲームを模したコンポーネント（値型・型安全に格納される）
type Position struct{ X, Y int }
type Health struct{ Cur, Max int }
type Label struct{ Text string }

// Link は他エンティティへの参照を持つ。serde が再マッピングできるか検証する
type Link struct{ Other ecs.Entity }

func TestArkSaveLoadRoundtrip(t *testing.T) {
	t.Parallel()
	world := ecs.NewWorld()

	// 型安全な Map でエンティティを作る
	posHealth := ecs.NewMap2[Position, Health](world)
	labelMap := ecs.NewMap1[Label](world)
	linkMap := ecs.NewMap1[Link](world)

	// プレイヤー相当（Position + Health + Label）
	player := posHealth.NewEntity(&Position{X: 5, Y: 10}, &Health{Cur: 80, Max: 100})
	labelMap.Add(player, &Label{Text: "Ash"})

	// 敵相当（Position + Health）で、Link でプレイヤーを参照
	enemy := posHealth.NewEntity(&Position{X: 20, Y: 20}, &Health{Cur: 30, Max: 30})
	linkMap.Add(enemy, &Link{Other: player}) // ← エンティティ参照

	// --- 型安全なクエリ（goecsのJoin/Visit相当）---
	filter := ecs.NewFilter2[Position, Health](world)
	q := filter.Query()
	count := 0
	for q.Next() {
		_, hp := q.Get() // *Position, *Health（アサーション不要・コンパイラ保証）
		hp.Cur++         // 全員HP+1
		count++
	}
	assert.Equal(t, 2, count, "Position+Healthを持つ2体")

	// --- save: ワールド丸ごとJSON（現状の44関数+49スキーマ型に相当）---
	jsonData, err := arkserde.Serialize(world)
	require.NoError(t, err)
	t.Logf("シリアライズJSON (%d bytes):\n%s", len(jsonData), string(jsonData))

	// --- load: 新しいワールドに復元 ---
	newWorld := ecs.NewWorld()
	// 復元先はコンポーネント型の登録が必要
	ecs.NewMap2[Position, Health](newWorld)
	ecs.NewMap1[Label](newWorld)
	ecs.NewMap1[Link](newWorld)

	require.NoError(t, arkserde.Deserialize(jsonData, newWorld))

	// --- 往復検証 ---
	newPosHealth := ecs.NewMap2[Position, Health](newWorld)
	newLabel := ecs.NewMap1[Label](newWorld)
	newLink := ecs.NewMap1[Link](newWorld)

	var restoredPlayer, restoredEnemy ecs.Entity
	fq := ecs.NewFilter2[Position, Health](newWorld).Query()
	for fq.Next() {
		pos, _ := fq.Get()
		if pos.X == 5 {
			restoredPlayer = fq.Entity()
		} else {
			restoredEnemy = fq.Entity()
		}
	}

	pPos, pHP := newPosHealth.Get(restoredPlayer)
	assert.Equal(t, Position{X: 5, Y: 10}, *pPos)
	assert.Equal(t, 81, pHP.Cur, "HP+1が保存されている")
	assert.Equal(t, "Ash", newLabel.Get(restoredPlayer).Text)

	// 敵の Link がプレイヤーを指すよう再マッピングされているか（★核心）
	link := newLink.Get(restoredEnemy)
	t.Logf("復元後の Link.Other=%v / restoredPlayer=%v", link.Other, restoredPlayer)
	assert.Equal(t, restoredPlayer, link.Other,
		"エンティティ参照が新ワールドのIDへ再マッピングされている")
}
