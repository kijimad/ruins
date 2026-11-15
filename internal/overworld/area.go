package overworld

import (
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/i18n"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// AreaKey は区域の同一性。区域の種類と原点で決まり、同じ種類の別の区域と区別する。ゼロ値は区域でないことを表す
type AreaKey struct {
	District string
	Anchor   consts.Coord[consts.Chunk]
}

// Area は地上のチャンクの場所。区域・集落・目印・遺跡の入口・荒れ地のいずれか
type Area struct {
	Key      AreaKey    // スプラッシュを出す大きなエリアの同一性。区域だけが持ち、それ以外はゼロ値
	Name     i18n.Msgid // エリアの表示名
	Building i18n.Msgid // 市街地の建物の表示名。市街地でなければ空
}

// AreaAt はチャンク c のエリアを返す。区域の Key は種類と原点から作り、同じ種類の別の区域と区別する。
// 集落・目印・遺跡の入口は1チャンクの小さな場所なので、荒れ地と同じく Key を持たない
func AreaAt(raws rawschema.Raws, runSeed uint64, c consts.Coord[consts.Chunk], cols, spawnRow consts.Chunk) Area {
	kind, r := classifyChunk(raws, runSeed, c, cols)
	switch kind {
	case chunkUrban:
		return Area{
			Key:      AreaKey{District: r.kind.Id, Anchor: r.anchor},
			Name:     i18n.Msgid(r.kind.Name),
			Building: glyphName(raws, urbanFacilityAt(raws, zoneCatalogFrom(raws), r, c, spawnRow)),
		}
	case chunkSettlement:
		return Area{Name: glyphName(raws, settlementAt(raws, runSeed, c).Id)}
	case chunkLandmark:
		return Area{Name: glyphName(raws, landmarkKindAt(raws, runSeed, c))}
	case chunkDungeonEntrance:
		return Area{Name: chunkGlyphName(raws, rawschema.DungeonEntrance)}
	case chunkWasteland:
		return Area{Name: chunkGlyphName(raws, rawschema.Wasteland)}
	}
	panic("unknown chunk type: " + string(kind))
}

// PlayerArea はプレイヤーのいるチャンクのエリアを返す。地上にいないかプレイヤーがいなければ ok=false
func PlayerArea(world w.World) (Area, bool) {
	if !query.IsOnOverworld(world) {
		return Area{}, false
	}
	sb := query.GetSeamlessBand(world)
	tile, ok := query.PlayerBandTile(world)
	if sb == nil || !ok || sb.ChunkW <= 0 {
		return Area{}, false
	}
	c := consts.Coord[consts.Chunk]{X: consts.Chunk(int(tile.X) / int(sb.ChunkW)), Y: sb.AbsChunkRow(tile.Y)}
	return AreaAt(world.Resources.RawMaster, sb.RunSeed, c, sb.Cols, sb.SpawnChunkY()), true
}

// glyphName は id の地図記号の表示名を返す。記号が無ければ空
func glyphName(raws rawschema.Raws, id string) i18n.Msgid {
	mg, ok := raw.GetMapGlyph(raws, id)
	if !ok {
		return ""
	}
	return i18n.Msgid(mg.Name)
}

// chunkGlyphName は持ち主の行を持たないチャンクの種類の地図記号の表示名を返す。記号が無ければ空
func chunkGlyphName(raws rawschema.Raws, chunk rawschema.GlyphChunk) i18n.Msgid {
	mg, ok := raw.GetChunkGlyph(raws, chunk)
	if !ok {
		return ""
	}
	return i18n.Msgid(mg.Name)
}
