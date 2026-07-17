package overworld

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/consts"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/mapspawner"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldstream"
)

// ChunkSeed は runSeed とチャンクの絶対インデックスから決定的なチャンク seed を導く。
// 同じ (runSeed, chunkIndex) なら常に同じ地形になり、破棄したチャンクを保存せず再生成できる。
// splitmix64 系の混合で、隣接インデックスでも seed が十分散る。
func ChunkSeed(runSeed uint64, chunkIndex int) uint64 {
	x := runSeed + uint64(chunkIndex)*0x9E3779B97F4A7C15
	x ^= x >> 30
	x *= 0xBF58476D1CE4E5B9
	x ^= x >> 27
	x *= 0x94D049BB133111EB
	x ^= x >> 31
	return x
}

// NewChunkGen は Band に渡す worldstream.ChunkGen を返す。
// chunkIndex ごとに (runSeed, chunkIndex) から決定的に生成し、帯ローカルの offsetX へ配置する。
// 高さ chunkH は固定（南北はストリーミングしない帯）。
func NewChunkGen(world w.World, runSeed uint64, chunkW, chunkH consts.Tile, planner mapplanner.PlannerType) worldstream.ChunkGen {
	return func(chunkIndex int, offsetX consts.Tile) error {
		plan, err := mapplanner.Plan(world, chunkW, chunkH, ChunkSeed(runSeed, chunkIndex), planner)
		if err != nil {
			return fmt.Errorf("チャンク生成失敗 (index=%d): %w", chunkIndex, err)
		}
		if _, err := mapspawner.SpawnAt(world, plan, offsetX, 0); err != nil {
			return fmt.Errorf("チャンク配置失敗 (index=%d): %w", chunkIndex, err)
		}
		return nil
	}
}
