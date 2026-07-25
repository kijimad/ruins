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

// ChunkSeed2D は runSeed とチャンク座標 (cx, cy) から、そのチャンク固有の決定的な seed を導く。
// 地形や地物の per-chunk 抽選はすべてこの seed を源にする。
//
// cx と cy を別々の大きな奇数で混ぜてから splitmix64 の finalizer で撹拌する。狙いは2つ。
//   - 転置で散らす: cx と cy に別の定数を掛けるので (cx,cy) と (cy,cx) が別 seed になる。同じ
//     定数だと cx*K+cy*K が対称になり、転置した座標が衝突して世界に対角線状の反復が出る。
//   - 隣接で散らす: finalizer の雪崩効果で (cx,cy) と (cx+1,cy) のような1違いの入力が無相関の
//     出力になる。撹拌しないと隣接 seed が定数差の線形関係になり、隣り合うチャンクが似てしまう。
//
// 奇数を掛けるのは 2^64 を法として可逆で座標の情報を落とさないため。定数を変えると同じ runSeed
// でも別の世界になる。
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
// チャンク座標ごとに (runSeed, 座標) から決定的に、地形→地物の層の順で生成し、
// 帯ローカルの (offsetX, offsetY) へ配置する。東西は無限にストリーミングし、南北は帯の行数に有界。
// start は開始チャンクの座標で PlaceFeatures の開始特例に、rows は帯の行数で当選行の抽選に使う。
func NewChunkGen(world w.World, runSeed uint64, chunkW, chunkH consts.Tile, rows consts.Chunk, start worldstream.ChunkCoord, planner mapplanner.PlannerType) worldstream.ChunkGen {
	return func(c worldstream.ChunkCoord, offsetX, offsetY consts.Tile) error {
		plan, err := mapplanner.Plan(world, chunkW, chunkH, ChunkSeed2D(runSeed, c.X, c.Y), planner)
		if err != nil {
			return fmt.Errorf("チャンク生成失敗 (x=%d, y=%d): %w", c.X, c.Y, err)
		}
		if _, err := mapspawner.SpawnAt(world, plan, offsetX, offsetY); err != nil {
			return fmt.Errorf("チャンク配置失敗 (x=%d, y=%d): %w", c.X, c.Y, err)
		}
		// 地形の上に地物の層を重ねる。小集落などの当選判定は (runSeed, 座標) の純関数
		if err := PlaceFeatures(world, runSeed, c, start, rows, offsetX, offsetY, chunkW, chunkH); err != nil {
			return err
		}
		// 地物がタイルを置換した後、チャンク全域のオートタイルを実状態から再計算する。
		// 置換タイル自身と、隣接する土の添字がここで揃う
		RecalcAutotileInXRange(world, offsetX, offsetX+chunkW)
		// 生成したチャンクのフィールドエンティティをオーバーワールドステージへ束縛する。
		// ステージをまたいで束縛するので、遺跡へ入るとき帯のエンティティをまとめて退避できる。
		// シフトで生成される新チャンクもここで束縛される。Player・SquadMember・既束縛は
		// Bind が自然に除外する
		stage.Bind(world, gc.NewOverworldStage())
		// このチャンクの両境界を接合後に再計算して継ぎ目を消す。
		// 東シフトでは西境界の offsetX、西シフトでは東境界の offsetX+chunkW が実境界になる。
		// RecalcSeamAutotileX は隣チャンクが無い帯端では自己スキップするため無条件に呼べる。
		RecalcSeamAutotileX(world, offsetX)
		RecalcSeamAutotileX(world, offsetX+chunkW)
		// 南北境界も同様に再計算する。行が1つの帯では上下とも帯端なので自己スキップされる
		RecalcSeamAutotileY(world, offsetY)
		RecalcSeamAutotileY(world, offsetY+chunkH)
		return nil
	}
}
