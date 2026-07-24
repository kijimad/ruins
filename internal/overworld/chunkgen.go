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

// ChunkSeed は runSeed とチャンクの絶対インデックスから決定的なチャンク seed を導く。
// splitmix64 系の混合で、隣接インデックスでも seed が十分散る。
func ChunkSeed(runSeed uint64, chunkIndex consts.Chunk) uint64 {
	x := runSeed + uint64(chunkIndex)*0x9E3779B97F4A7C15
	x ^= x >> 30
	x *= 0xBF58476D1CE4E5B9
	x ^= x >> 27
	x *= 0x94D049BB133111EB
	x ^= x >> 31
	return x
}

// ChunkSeed2D は runSeed とチャンク座標 (cx, cy) から決定的なチャンク seed を導く。
// ChunkSeed の2次元版で、cx と cy を異なる奇数定数で混ぜてから splitmix64 系で撹拌する。
// 隣接や転置の座標でも seed は十分散る。
//
// 1次元の ChunkSeed との互換は保証しない。縦の行数を増やす移行は世界形状そのものの
// 変更なので全チャンクの再生成を許容し、互換の約束で定数設計を縛らない。
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
// chunkIndex ごとに (runSeed, chunkIndex) から決定的に生成し、帯ローカルの offsetX へ配置する。
// 高さ chunkH は固定。南北はストリーミングしない帯。
func NewChunkGen(world w.World, runSeed uint64, chunkW, chunkH consts.Tile, planner mapplanner.PlannerType) worldstream.ChunkGen {
	return func(chunkIndex consts.Chunk, offsetX consts.Tile) error {
		plan, err := mapplanner.Plan(world, chunkW, chunkH, ChunkSeed(runSeed, chunkIndex), planner)
		if err != nil {
			return fmt.Errorf("チャンク生成失敗 (index=%d): %w", chunkIndex, err)
		}
		if _, err := mapspawner.SpawnAt(world, plan, offsetX, 0); err != nil {
			return fmt.Errorf("チャンク配置失敗 (index=%d): %w", chunkIndex, err)
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
