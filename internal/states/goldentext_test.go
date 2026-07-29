package states

// 各 State の GoldenText を1箇所へ集める。VRT のゴールデンは画像でなく、この決定的なテキストで退行を捉える。
// メソッドはテスト専用で、vrt.snapshotStates が構造的型アサーションによりテストバイナリ内から呼ぶ。
// _test.go に置くことで本体バイナリにはリンクされず、プロダクション API を増やさない。
// 各メソッドは State のビューモデルを決定的な文字列にする。時刻や乱数に依存する値は避け、
// 未ソートの列は名前で並べ替えて決定化する。

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mlange-42/ark/ecs"

	"github.com/kijimaD/ruins/internal/messagedata"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// sortedNames はエンティティ列を表示名の昇順で返す。同名は entity.ID() で並びを固定する。ark の反復順や
// query の未ソートに依存しないよう、ゴールデンに出す前に名前と ID で並べ替える。
func sortedNames(world w.World, ents []ecs.Entity) []string {
	sorted := append([]ecs.Entity(nil), ents...)
	sort.Slice(sorted, func(i, j int) bool {
		ni, nj := query.GetEntityName(sorted[i], world), query.GetEntityName(sorted[j], world)
		if ni != nj {
			return ni < nj
		}
		return sorted[i].ID() < sorted[j].ID()
	})
	names := make([]string, len(sorted))
	for i, e := range sorted {
		names[i] = query.GetEntityName(e, world)
	}
	return names
}

// writeNameLines は名前列をインデント付きの行としてゴールデンへ書く。
func writeNameLines(b *strings.Builder, names []string) {
	for _, name := range names {
		fmt.Fprintf(b, "  %s\n", name)
	}
}

// GoldenText はカーソルとカーソル上のエンティティを返す。ダンジョンのグリッド自体は Dungeon の
// ゴールデンが担保するので、ここではオーバーレイ固有のカーソルと対象だけを出す。
// 実UIはカーソル下のタイルも名前表示するため、ここでもタイルを除外せず表示内容と揃える。
func (st *LookAroundState) GoldenText(world w.World) string {
	var b strings.Builder
	fmt.Fprintf(&b, "look-around cursor %s\n", st.cursor)
	writeNameLines(&b, sortedNames(world, query.GetEntitiesAt(world, st.cursor.X, st.cursor.Y)))
	return b.String()
}

// GoldenText はカーソル位置と方向、そこにある拾得可能アイテムを返す。
func (st *PickupState) GoldenText(world w.World) string {
	var b strings.Builder
	dx := int(st.cursor.X - st.playerPos.X)
	dy := int(st.cursor.Y - st.playerPos.Y)
	fmt.Fprintf(&b, "pickup cursor %s dir %s\n", st.cursor, offsetToLabel(dx, dy))
	var pickable []ecs.Entity
	for _, e := range query.GetEntitiesAt(world, st.cursor.X, st.cursor.Y) {
		if query.IsPickable(e, world) {
			pickable = append(pickable, e)
		}
	}
	writeNameLines(&b, sortedNames(world, pickable))
	return b.String()
}

// GoldenText はフェーズと、アイテム選択フェーズではバックパックのアイテム、配置先選択フェーズでは
// カーソルの方向を返す。バックパックの列は未ソートなので名前で並べ替える。
func (st *PlaceState) GoldenText(world w.World) string {
	var b strings.Builder
	switch st.phase {
	case placePhaseSelectItem:
		b.WriteString("place select-item\n")
		writeNameLines(&b, sortedNames(world, st.backpackItems))
	case placePhaseSelectTile:
		dx := int(st.cursor.X - st.playerPos.X)
		dy := int(st.cursor.Y - st.playerPos.Y)
		fmt.Fprintf(&b, "place select-tile dir %s\n", offsetToLabel(dx, dy))
	}
	return b.String()
}

// GoldenText は選択中の対象インデックスと、視界内の射撃対象の敵を返す。表示上の並びは距離順だが、
// 同距離の並びは反復順に依存して揺れるため、ゴールデンでは名前と ID で並べ直して決定化する。
// 命中率は乱数を含みうるので出さない。
func (st *ShootingState) GoldenText(world w.World) string {
	var b strings.Builder
	fmt.Fprintf(&b, "shooting target %d/%d\n", st.targetIndex, len(st.enemies))
	writeNameLines(&b, sortedNames(world, st.enemies))
	return b.String()
}

// GoldenText はメッセージの話者・本文・選択肢を返す。GameOver/Debug/Save/Load は *MessageState なので
// このメソッドを共有する。PersistentMessageState は MessageState を埋め込むので同様。タイプライタの進行や
// 背景画像は描画側の可変状態なので出さず、messageData から決定的に組む。
func (st *MessageState) GoldenText(_ w.World) string {
	return messageDataText(st.messageData)
}

// messageDataText は MessageData を決定的なテキストにする。話者・本文行・選択肢を上から並べる。
func messageDataText(md *messagedata.MessageData) string {
	if md == nil {
		return "message (empty)\n"
	}
	var b strings.Builder
	if md.Speaker != "" {
		fmt.Fprintf(&b, "speaker %s\n", md.Speaker)
	}
	for _, line := range md.TextSegmentLines {
		var seg strings.Builder
		for _, s := range line {
			seg.WriteString(s.Text)
		}
		fmt.Fprintf(&b, "%s\n", seg.String())
	}
	for _, c := range md.Choices {
		fmt.Fprintf(&b, "- %s\n", c.Text)
	}
	return b.String()
}

// GoldenText はコンポーネント種別ごとの保有数を返す。Count 降順、同数は名前昇順で決定化する。
func (st *ComponentDebugState) GoldenText(world w.World) string {
	p := st.fetchProps(world)
	items := append([]componentDebugItem(nil), p.Items...)
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count != items[j].Count {
			return items[i].Count > items[j].Count
		}
		return items[i].Name < items[j].Name
	})
	var b strings.Builder
	fmt.Fprintf(&b, "component total %d\n", p.Total)
	for _, it := range items {
		fmt.Fprintf(&b, "  %s %d\n", it.Name, it.Count)
	}
	return b.String()
}

// GoldenText は収納名・重量と、タブごとのアイテム一覧を返す。アイテムは fetchProps が SortEntities で
// 名前順に並べているので決定的。
func (st *StorageMenuState) GoldenText(world w.World) string {
	p := st.fetchProps(world)
	var b strings.Builder
	fmt.Fprintf(&b, "storage %s weight %s\n", p.StorageName, p.WeightText)
	for _, tab := range p.Tabs {
		fmt.Fprintf(&b, "[%s]\n", tab.Label)
		for _, it := range tab.Items {
			fmt.Fprintf(&b, "  %s x%s\n", it.Name, it.Count)
		}
	}
	return b.String()
}

// GoldenText は一括コマンドと隊員の状態を返す。SquadMembers は未ソートなので名前で並べ替える。
func (st *SquadMenuState) GoldenText(world w.World) string {
	p := st.fetchProps(world)
	members := append([]squadMemberData(nil), p.Members...)
	sort.Slice(members, func(i, j int) bool {
		if members[i].Name != members[j].Name {
			return members[i].Name < members[j].Name
		}
		return members[i].Entity.ID() < members[j].Entity.ID()
	})
	var b strings.Builder
	fmt.Fprintf(&b, "squad commands %s\n", strings.Join(p.BatchCommands, "/"))
	for _, m := range members {
		fmt.Fprintf(&b, "  %s HP %s pos %s combat %s pickup %s handling %s\n",
			m.Name, m.HP, m.Position, m.Combat, m.ItemPickup, m.ItemHandling)
	}
	return b.String()
}

// GoldenText は部隊の隊員と HP を返す。SquadMembers は未ソートなので名前で並べ替える。
func (st *FormationMenuState) GoldenText(world w.World) string {
	p := st.fetchProps(world)
	members := append([]formationMemberData(nil), p.Members...)
	sort.Slice(members, func(i, j int) bool {
		if members[i].Name != members[j].Name {
			return members[i].Name < members[j].Name
		}
		return members[i].Entity.ID() < members[j].Entity.ID()
	})
	var b strings.Builder
	b.WriteString("formation\n")
	for _, m := range members {
		fmt.Fprintf(&b, "  %s HP %s\n", m.Name, m.HP)
	}
	return b.String()
}

// GoldenText はプレイヤー名と各タブの項目を返す。タブと項目の順序は fetchProps が決める固定順で決定的。
func (st *StatusState) GoldenText(world w.World) string {
	p := st.fetchProps(world)
	return tabbedText("status "+p.PlayerName, p.Tabs)
}

// tabbedText は statusProps のタブ群を決定的なテキストにする。ヘッダ行は # を付け、値と modifier を並べる。
func tabbedText(title string, tabs []statusTabData) string {
	var b strings.Builder
	b.WriteString(title + "\n")
	for _, tab := range tabs {
		fmt.Fprintf(&b, "[%s]\n", tab.Label)
		for _, it := range tab.Items {
			switch {
			case it.IsHeader:
				fmt.Fprintf(&b, "  # %s\n", it.Label)
			case it.Modifier != "":
				fmt.Fprintf(&b, "  %s %s %s\n", it.Label, it.Value, it.Modifier)
			default:
				fmt.Fprintf(&b, "  %s %s\n", it.Label, it.Value)
			}
		}
	}
	return b.String()
}

// GoldenText は隊員名と1タブの項目を返す。
func (st *MemberStatusState) GoldenText(world w.World) string {
	p := st.fetchProps(world)
	var b strings.Builder
	fmt.Fprintf(&b, "member %s\n", p.Name)
	for _, tab := range p.Tabs {
		fmt.Fprintf(&b, "[%s]\n", tab.Label)
		for _, it := range tab.Items {
			fmt.Fprintf(&b, "  %s %s\n", it.Label, it.Value)
		}
	}
	return b.String()
}

// GoldenText はタブ(プレイヤー・隊員)ごとのアイテム一覧を返す。アイテムは fetchProps が SortEntities で
// 名前順に並べているので決定的。
func (st *InventoryMenuState) GoldenText(world w.World) string {
	p := st.fetchProps(world)
	var b strings.Builder
	b.WriteString("inventory\n")
	for _, tab := range p.Tabs {
		fmt.Fprintf(&b, "[%s]\n", tab.Label)
		for _, it := range tab.Items {
			fmt.Fprintf(&b, "  %s x%s\n", it.Name, it.Count)
		}
	}
	return b.String()
}

// GoldenText はタブごとに装備スロットと装備品を返す。スロットは固定順で決定的。空きは "-"。
func (st *EquipMenuState) GoldenText(world w.World) string {
	p := st.fetchSlotProps(world)
	var b strings.Builder
	b.WriteString("equip\n")
	for _, tab := range p.Tabs {
		fmt.Fprintf(&b, "[%s]\n", tab.Label)
		for _, it := range tab.Items {
			name := it.ItemName
			if name == "" {
				name = "-"
			}
			fmt.Fprintf(&b, "  %s: %s\n", it.SlotLabel, name)
		}
	}
	return b.String()
}

// GoldenText はタブごとのレシピ一覧と作成可否を返す。レシピは名前順に並んでいるので決定的。
func (st *CraftMenuState) GoldenText(world w.World) string {
	p := st.fetchProps(world)
	var b strings.Builder
	b.WriteString("craft\n")
	for _, tab := range p.Tabs {
		fmt.Fprintf(&b, "[%s]\n", tab.Label)
		for _, it := range tab.Items {
			mark := "x"
			if it.CanCraft {
				mark = "o"
			}
			fmt.Fprintf(&b, "  %s %s\n", mark, it.RecipeName)
		}
	}
	return b.String()
}

// GoldenText は所持金とタブごとの品目と価格を返す。売却タブの列は未ソートなので名前で並べ替える。
func (st *ShopMenuState) GoldenText(world w.World) string {
	p := st.fetchProps(world)
	var b strings.Builder
	fmt.Fprintf(&b, "shop currency %d\n", p.Currency)
	for _, tab := range p.Tabs {
		items := append([]shopItemData(nil), tab.Items...)
		sort.Slice(items, func(i, j int) bool {
			if items[i].Label != items[j].Label {
				return items[i].Label < items[j].Label
			}
			return items[i].Entity.ID() < items[j].Entity.ID()
		})
		fmt.Fprintf(&b, "[%s]\n", tab.Label)
		for _, it := range items {
			fmt.Fprintf(&b, "  %s %d\n", it.Label, it.Price)
		}
	}
	return b.String()
}

// GoldenText は所持金と酒場の雇用候補を返す。候補は world.Config.RNG から生成されるが、固定シードの VRT
// world で決定的。生成順(Index 順)で出す。
func (st *TavernMenuState) GoldenText(world w.World) string {
	p := st.fetchProps(world)
	var b strings.Builder
	fmt.Fprintf(&b, "tavern currency %d\n", p.Currency)
	for _, c := range p.Candidates {
		fmt.Fprintf(&b, "  %s %s cost %d\n", c.Name, c.Stats, c.Cost)
	}
	return b.String()
}

// GoldenText はプレイヤーの絶対チャンク座標と俯瞰図の記号グリッドを返す。glyphs は RunSeed から純関数で
// 算出されるので決定的。
func (st *OverworldMapState) GoldenText(_ w.World) string {
	var b strings.Builder
	fmt.Fprintf(&b, "overworld-map player %s\n", st.playerAbs)
	for _, row := range st.glyphs {
		b.WriteString(string(row))
		b.WriteByte('\n')
	}
	return b.String()
}
