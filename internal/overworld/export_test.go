package overworld

import (
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
)

// IsSettlementChunk は外部のテストから、チャンク c が集落に分類されるかを問い合わせる。帯を作らずに生成するテスト向けに開始の行を0とみなす
func IsSettlementChunk(raws rawschema.Raws, runSeed uint64, c consts.Coord[consts.Chunk], cols consts.Chunk) bool {
	return chunkTypeAt(raws, runSeed, c, cols, 0) == chunkSettlement
}
