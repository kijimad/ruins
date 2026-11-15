package overworld

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/i18n"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// Area は地上のチャンクの場所。区域・集落・目印・遺跡の入口・荒れ地のいずれか
type Area struct {
	Key      string     // エリアの同一性。同じなら同じエリア。荒れ地は空
	Name     i18n.Msgid // エリアの表示名
	Building i18n.Msgid // 市街地の建物の表示名。市街地でなければ空
}

// AreaAt はチャンク c のエリアを返す。区域の Key は種類と原点から作り、同じ種類の別の区域と区別する
func AreaAt(raws rawschema.Raws, runSeed uint64, c consts.Coord[consts.Chunk], cols, spawnRow consts.Chunk) Area {
	kind := chunkTypeAt(raws, runSeed, c, cols)
	switch kind {
	case chunkUrban:
		r, _ := districtAt(raws, runSeed, c, cols)
		return Area{
			Key:      fmt.Sprintf("%s:%s:%d,%d", kind, r.kind.Id, r.anchor.X, r.anchor.Y),
			Name:     i18n.Msgid(r.kind.Name),
			Building: glyphName(raws, urbanFacilityAt(raws, zoneCatalogFrom(raws), r, c, spawnRow)),
		}
	case chunkSettlement:
		return chunkArea(kind, c, glyphName(raws, settlementAt(raws, runSeed, c).Id))
	case chunkLandmark:
		return chunkArea(kind, c, glyphName(raws, landmarkKindAt(raws, runSeed, c)))
	case chunkDungeonEntrance:
		return chunkArea(kind, c, chunkGlyphName(raws, rawschema.DungeonEntrance))
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

// chunkArea は1チャンクで完結するエリアを返す
func chunkArea(kind chunkType, c consts.Coord[consts.Chunk], name i18n.Msgid) Area {
	return Area{Key: fmt.Sprintf("%s:%d,%d", kind, c.X, c.Y), Name: name}
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
