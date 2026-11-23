package overworld

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/mapspawner"
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/stage"
	"github.com/kijimaD/ruins/internal/worldstream"
)

// NewChunkGen は Band に渡す worldstream.ChunkGen を返す。
// チャンク座標ごとに (runSeed, 座標) から決定的に、地形→地物の層の順で生成し、
// 帯ローカルの (offsetX, offsetY) へ配置する。南北は北へ無限にストリーミングし、東西は帯の列数に有界。
// cols は帯の列数で当選列の抽選に使う。
func NewChunkGen(world w.World, runSeed uint64, chunkW, chunkH consts.Tile, cols consts.Chunk, planner mapplanner.PlannerType) worldstream.ChunkGen {
	return func(c consts.Coord[consts.Chunk], offsetX, offsetY consts.Tile) error {
		zone, ok := raw.GetScatterZone(world.Resources.RawMaster, string(outdoorZoneAt(world.Resources.RawMaster, runSeed, c, cols, spawnRowOf(world))))
		if !ok {
			return fmt.Errorf("chunk (x=%d, y=%d): %w", c.X, c.Y, errScatterZoneMissing)
		}
		zoned := planner
		zoned.Ground = zone.Ground
		plan, err := mapplanner.Plan(world, chunkW, chunkH, consts.Mix2D(runSeed, c.X, c.Y), zoned)
		if err != nil {
			return fmt.Errorf("failed to generate chunk (x=%d, y=%d): %w", c.X, c.Y, err)
		}
		if _, err := mapspawner.SpawnAt(world, plan, offsetX, offsetY); err != nil {
			return fmt.Errorf("failed to place chunk (x=%d, y=%d): %w", c.X, c.Y, err)
		}
		// 地形の上に地物の層を重ねる。集落などの当選判定は (runSeed, 座標) の純関数
		if err := PlaceFeatures(world, runSeed, c, cols, offsetX, offsetY, chunkW, chunkH); err != nil {
			return err
		}
		// 地物がタイルを置換した後、チャンク全域のオートタイルを実状態から再計算する。
		// 置換タイル自身と、隣接する土の添字がここで揃う
		RecalcAutotileInXRange(world, offsetX, offsetX+chunkW)
		// 生成したチャンクのフィールドエンティティをオーバーワールドステージへ束縛する。
		// ステージをまたいで束縛するので、遺跡へ入るとき帯のエンティティをまとめて退避できる。
		// シフトで生成される新チャンクもここで束縛される。Player・既束縛は
		// Bind が自然に除外する
		stage.Bind(world, gc.NewOverworldStage())
		// このチャンクの4境界を接合後に再計算して継ぎ目を消す。片側が空の帯端は自己スキップ
		// されるため、東西南北を無条件にまとめて呼べる。北シフトは南境界が新チャンク側の実境界になる。
		RecalcChunkSeams(world, offsetX, offsetY, chunkW, chunkH)
		return nil
	}
}
