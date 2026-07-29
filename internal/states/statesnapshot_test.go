package states

// 各 State の論理内容スナップショットを1関数へ集める。VRT のゴールデンは画像でなく、ビューモデルの JSON で
// 退行を捉える。vrt.SnapshotFunc として注入され、テストバイナリ内からだけ呼ばれる。
// fetchProps を持つメニューはビューモデルをそのまま返し、整形は vrt の JSON 直列化に任せる。
// props を持たないオーバーレイは専用のスナップショット構造体をここで組む。
// 時刻や乱数に依存する値は避け、未ソートの列は名前で並べ替えて決定化する。

import (
	"sort"

	"github.com/mlange-42/ark/ecs"

	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/messagedata"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// StateSnapshot は state の論理内容を返す。対応しない純UIメニューは ok=false を返す。
// 外部テストパッケージの golden テストから vrt.SnapshotFunc として注入するため、テストファイル内で
// エクスポートする。テストバイナリ外にはリンクされない
func StateSnapshot(world w.World, s es.State[w.World]) (any, bool) {
	switch st := s.(type) {
	case *LookAroundState:
		// 実UIはカーソル下のタイルも名前表示するため、タイルを除外せず表示内容と揃える
		return lookAroundSnapshot{
			Cursor: st.cursor.String(),
			Names:  sortedNames(world, query.GetEntitiesAt(world, st.cursor.X, st.cursor.Y)),
		}, true
	case *PickupState:
		var pickable []ecs.Entity
		for _, e := range query.GetEntitiesAt(world, st.cursor.X, st.cursor.Y) {
			if query.IsPickable(e, world) {
				pickable = append(pickable, e)
			}
		}
		return pickupSnapshot{
			Cursor: st.cursor.String(),
			Dir:    offsetToLabel(int(st.cursor.X-st.playerPos.X), int(st.cursor.Y-st.playerPos.Y)),
			Names:  sortedNames(world, pickable),
		}, true
	case *PlaceState:
		snap := placeSnapshot{}
		switch st.phase {
		case placePhaseSelectItem:
			snap.Phase = "select-item"
			snap.Names = sortedNames(world, st.backpackItems)
		case placePhaseSelectTile:
			snap.Phase = "select-tile"
			snap.Dir = offsetToLabel(int(st.cursor.X-st.playerPos.X), int(st.cursor.Y-st.playerPos.Y))
		}
		return snap, true
	case *ShootingState:
		// 表示上の並びは距離順だが、同距離の並びは反復順に依存して揺れるため名前と ID で決定化する。
		// 命中率は乱数を含みうるので出さない
		return shootingSnapshot{
			TargetIndex: st.targetIndex,
			TargetCount: len(st.enemies),
			Enemies:     sortedNames(world, st.enemies),
		}, true
	case *PersistentMessageState:
		return messageSnapshotOf(st.messageData), true
	case *MessageState:
		// GameOver/Debug/Save/Load も *MessageState を共有する
		return messageSnapshotOf(st.messageData), true
	case *ComponentDebugState:
		return st.fetchProps(world), true
	case *StorageMenuState:
		return st.fetchProps(world), true
	case *SquadMenuState:
		return st.fetchProps(world), true
	case *FormationMenuState:
		return st.fetchProps(world), true
	case *StatusState:
		return st.fetchProps(world), true
	case *MemberStatusState:
		return st.fetchProps(world), true
	case *InventoryMenuState:
		return st.fetchProps(world), true
	case *EquipMenuState:
		return equipSnapshotOf(st.fetchSlotProps(world)), true
	case *CraftMenuState:
		return st.fetchProps(world), true
	case *ShopMenuState:
		return st.fetchProps(world), true
	case *TavernMenuState:
		return st.fetchProps(world), true
	case *OverworldMapState:
		glyphs := make([]string, len(st.glyphs))
		for i, row := range st.glyphs {
			glyphs[i] = string(row)
		}
		return overworldMapSnapshot{Player: st.playerAbs.String(), Glyphs: glyphs}, true
	}
	return nil, false
}

type lookAroundSnapshot struct {
	Cursor string
	Names  []string
}

type pickupSnapshot struct {
	Cursor string
	Dir    string
	Names  []string
}

type placeSnapshot struct {
	Phase string
	Dir   string   `json:",omitempty"`
	Names []string `json:",omitempty"`
}

type shootingSnapshot struct {
	TargetIndex int
	TargetCount int
	Enemies     []string
}

type messageSnapshot struct {
	Speaker string `json:",omitempty"`
	Lines   []string
	Choices []string `json:",omitempty"`
}

// messageSnapshotOf は MessageData を決定的なビューモデルにする。タイプライタの進行や背景画像は
// 描画側の可変状態なので含めない
func messageSnapshotOf(md *messagedata.MessageData) messageSnapshot {
	if md == nil {
		return messageSnapshot{}
	}
	snap := messageSnapshot{Speaker: md.Speaker}
	for _, line := range md.TextSegmentLines {
		var text string
		for _, seg := range line {
			text += seg.Text
		}
		snap.Lines = append(snap.Lines, text)
	}
	for _, c := range md.Choices {
		snap.Choices = append(snap.Choices, c.Text)
	}
	return snap
}

type equipSnapshot struct {
	Tabs []equipTabSnapshot
}

type equipTabSnapshot struct {
	Label string
	Slots []string
}

// equipSnapshotOf はスロット画面の props をビューモデルにする。props はエンティティ参照が多いので、
// 直接直列化せずスロットと装備名だけを取り出す。空きスロットは "-"
func equipSnapshotOf(p slotScreenProps) equipSnapshot {
	snap := equipSnapshot{}
	for _, tab := range p.Tabs {
		ts := equipTabSnapshot{Label: tab.Label}
		for _, it := range tab.Items {
			name := it.ItemName
			if name == "" {
				name = "-"
			}
			ts.Slots = append(ts.Slots, it.SlotLabel+": "+name)
		}
		snap.Tabs = append(snap.Tabs, ts)
	}
	return snap
}

type overworldMapSnapshot struct {
	Player string
	Glyphs []string
}

// sortedNames はエンティティ列を表示名の昇順で返す。同名は entity.ID() で並びを固定する。ark の反復順や
// query の未ソートに依存しないよう、名前を一度だけ引く decorate-sort で並べ替える
func sortedNames(world w.World, ents []ecs.Entity) []string {
	type named struct {
		name string
		id   uint32
	}
	ds := make([]named, len(ents))
	for i, e := range ents {
		ds[i] = named{name: query.GetEntityName(e, world), id: e.ID()}
	}
	sort.Slice(ds, func(i, j int) bool {
		if ds[i].name != ds[j].name {
			return ds[i].name < ds[j].name
		}
		return ds[i].id < ds[j].id
	})
	names := make([]string, len(ds))
	for i, d := range ds {
		names[i] = d.name
	}
	return names
}
