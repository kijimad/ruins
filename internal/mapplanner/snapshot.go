package mapplanner

import (
	"slices"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/kijimaD/ruins/internal/oapi"
)

// Snapshot はマップ生成の各フェーズ完了時点でのMetaPlanの状態を保存する
type Snapshot struct {
	Label       string
	Tiles       []oapi.Tile
	Rooms       []gc.Rect
	Corridors   [][]gc.TileIdx
	NPCs        []NPCSpec
	Items       []ItemSpec
	Props       []PropsSpec
	Doors       []DoorSpec
	NextPortals []consts.Coord[consts.Tile]
	SpawnPoints []maptemplate.SpawnPoint
}

// takeSnapshot はRecordingが有効な場合にMetaPlanの現在の状態をスナップショットとして保存する
func (b *PlannerChain) takeSnapshot(label string) {
	if !b.Recording {
		return
	}
	d := &b.PlanData
	b.Snapshots = append(b.Snapshots, Snapshot{
		Label:       label,
		Tiles:       slices.Clone(d.Tiles),
		Rooms:       slices.Clone(d.Rooms),
		Corridors:   deepCloneCorridors(d.Corridors),
		NPCs:        slices.Clone(d.NPCs),
		Items:       slices.Clone(d.Items),
		Props:       slices.Clone(d.Props),
		Doors:       slices.Clone(d.Doors),
		NextPortals: slices.Clone(d.NextPortals),
		SpawnPoints: slices.Clone(d.SpawnPoints),
	})
}

// deepCloneCorridors は二次元スライスを深くコピーする。
// slices.Cloneは外側のスライスのみコピーし内側は参照を共有するため、スナップショット間の干渉を防ぐ
func deepCloneCorridors(src [][]gc.TileIdx) [][]gc.TileIdx {
	dst := make([][]gc.TileIdx, len(src))
	for i, c := range src {
		dst[i] = slices.Clone(c)
	}
	return dst
}
