package aiinput

import (
	"math/rand/v2"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testscene"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/require"
)

// BenchmarkProcessAll は AI フェーズ（Processor.ProcessAll）の実時間を敵数・配置分布別に計測する。
//
// 移動・走りの各ステップ後には必ずこの AI フェーズが1回走る。60fps のフレーム予算は約16.67ms/frame
// なので、ms/op がこれを大きく下回っているほど移動が滑らかになる（UX 指標）。
//
// 配置分布：
//   - clustered: プレイヤー周辺（activationRadius 以内）に密集＝全 SoloAI が処理される（カリング無効の最悪ケース）
//   - spread:    大マップ全域に散布＝多くが圏外でスキップされる（カリングが効く現実的な探索ケース）
//
// clustered と spread の差分が、状態つきアクティベーション半径カリングの効果を表す。
// レポートする custom metric：
//   - processed: 初期状態でカリング後に実際に処理される SoloAI 数
//   - ms/op:     1回の ProcessAll あたりのミリ秒（16.67ms 予算との比較用）
func BenchmarkProcessAll(b *testing.B) {
	const (
		mapSize = 300 // シームレスワールド想定の大マップ（テスト既定の 50 では全敵が圏内になりカリングが効かない）
		cx      = mapSize / 2
		cy      = mapSize / 2
	)

	scenarios := []struct {
		name   string
		count  int
		spread bool
	}{
		{"clustered/25", 25, false},
		{"spread/25", 25, true},
		{"clustered/100", 100, false},
		{"spread/100", 100, true},
		{"clustered/400", 400, false},
		{"spread/400", 400, true},
	}

	for _, sc := range scenarios {
		b.Run(sc.name, func(b *testing.B) {
			world, _ := testscene.InitDungeonWorld(b, mapSize, cx, cy)

			// 敵配置は固定 seed で再現性を持たせる
			rng := rand.New(rand.NewPCG(1, 2))
			for range sc.count {
				var x, y int
				if sc.spread {
					x, y = rng.IntN(mapSize), rng.IntN(mapSize)
				} else {
					x = cx + rng.IntN(2*activationRadius+1) - activationRadius
					y = cy + rng.IntN(2*activationRadius+1) - activationRadius
				}
				testscene.MustSpawnEnemy(b, world, x, y)
			}

			// 参考値：初期状態でカリング後に処理される SoloAI 数
			var allSolo []ecs.Entity
			soloQuery := ecs.NewFilter2[gc.SoloAI, gc.GridElement](world.ECS).Query()
			for soloQuery.Next() {
				allSolo = append(allSolo, soloQuery.Entity())
			}
			processed := len(cullDistantSolo(world, allSolo))

			proc := NewProcessor(rand.New(rand.NewPCG(3, 4)))

			b.ResetTimer()
			for range b.N {
				// AP 回復（毎ターン末に走る本来の前処理）は計測対象外
				b.StopTimer()
				require.NoError(b, query.RestoreAllActionPoints(world))
				b.StartTimer()

				require.NoError(b, proc.ProcessAll(world))
			}
			b.StopTimer()

			b.ReportMetric(float64(processed), "processed")
			b.ReportMetric(b.Elapsed().Seconds()*1000/float64(b.N), "ms/op")
		})
	}
}
