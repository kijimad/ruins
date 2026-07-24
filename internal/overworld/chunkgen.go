package overworld

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/mapspawner"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/stage"
	"github.com/kijimaD/ruins/internal/worldstream"
)

// ChunkSeed2D は runSeed とチャンク座標 (cx, cy) から決定的なチャンク seed を導く。
// cx と cy を異なる奇数定数で混ぜてから splitmix64 系で撹拌し、隣接や転置の座標でも
// seed は十分散る。
//
// 行数を増やす移行では旧1次元シードとの互換を保証しない。世界形状そのものの変更なので
// 全チャンクの再生成を許容し、互換の約束で定数設計を縛らない。
func ChunkSeed2D(runSeed uint64, cx, cy consts.Chunk) uint64 {
	x := runSeed + uint64(cx)*0x9E3779B97F4A7C15 + uint64(cy)*0xC2B2AE3D27D4EB4F
	x ^= x >> 30
	x *= 0xBF58476D1CE4E5B9
	x ^= x >> 27
	x *= 0x94D049BB133111EB
	x ^= x >> 31
	return x
}

// NewChunkGen は Band に渡す worldstream.ChunkGen を返す。
// チャンク座標ごとに (runSeed, 座標) から決定的に生成し、帯ローカルの (offsetX, offsetY) へ配置する。
// 東西は無限にストリーミングし、南北は帯の行数に有界。
func NewChunkGen(world w.World, runSeed uint64, chunkW, chunkH consts.Tile, planner mapplanner.PlannerType) worldstream.ChunkGen {
	return func(c worldstream.ChunkCoord, offsetX, offsetY consts.Tile) error {
		plan, err := mapplanner.Plan(world, chunkW, chunkH, ChunkSeed2D(runSeed, c.X, c.Y), planner)
		if err != nil {
			return fmt.Errorf("チャンク生成失敗 (x=%d, y=%d): %w", c.X, c.Y, err)
		}
		if _, err := mapspawner.SpawnAt(world, plan, offsetX, offsetY); err != nil {
			return fmt.Errorf("チャンク配置失敗 (x=%d, y=%d): %w", c.X, c.Y, err)
		}
		// 生成したチャンクのフィールドエンティティをオーバーワールドステージへ束縛する。
		// 共存方式で遺跡へ入るとき帯を退避できるようにする。シフトで生成される新チャンクも
		// ここで束縛される。Player・SquadMember・既束縛は Bind が自然に除外する
		stage.Bind(world, gc.NewOverworldStage())
		// このチャンクの両境界を接合後に再計算して継ぎ目を消す。
		// 東シフトでは西境界の offsetX、西シフトでは東境界の offsetX+chunkW が実境界になる。
		// RecalcSeamAutotile は隣チャンクが無い帯端では自己スキップするため無条件に呼べる。
		RecalcSeamAutotile(world, offsetX)
		RecalcSeamAutotile(world, offsetX+chunkW)
		return nil
	}
}
