package vrt

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mlange-42/ark/ecs"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// WorldSnapshot は spawn 後の world ECS を決定的にテキスト化したゴールデン対象。画像に依らず、
// タイル配置とエンティティの内容で退行を捉える。世界主体の state ゴールデンが GoldenText の代わりに使う。
type WorldSnapshot struct {
	Grid        []string         // Level の各行をタイル名で。y→x の固定順
	Entities    []EntitySnapshot // タイル以外を座標→名前の固定順で
	OutOfBounds []string         // Level 寸法の外に置かれたタイル。正常な生成では空
}

// EntitySnapshot はタイル以外のエンティティ1体の決定的要約。座標と種別と表示名を持つ。
type EntitySnapshot struct {
	Pos  consts.Coord[consts.Tile]
	Kind string
	Name string
}

// SnapshotWorld は現ステージの world を WorldSnapshot にする。現ステージにフィールドが無ければ空を返し、
// 呼び出し側はそれを「マップの無い state」の目印にできる。map を反復に使わず、グリッドは座標の二重ループ、
// エンティティは明示ソートで決定的順序にする。ark の Query 反復順は決定的でないため必ずソートする。
func SnapshotWorld(world w.World) WorldSnapshot {
	var snap WorldSnapshot
	field := query.GetCurrentStageField(world)
	if field == nil {
		return snap
	}

	// タイル名を座標で引けるようにする。タイルは Tile タグ + GridElement + Name を持つ
	tileName := map[gc.GridElement]string{}
	tq := query.ActiveFilter1[gc.Tile](world).Query()
	for tq.Next() {
		e := tq.Entity()
		g := world.Components.GridElement.Get(e)
		tileName[gc.GridElement{Coord: g.Coord}] = query.GetEntityName(e, world)
	}

	// グリッドを y→x で走査する。タイルの無いセルは "-"
	for y := consts.Tile(0); y < field.Level.TileHeight; y++ {
		cells := make([]string, 0, field.Level.TileWidth)
		for x := consts.Tile(0); x < field.Level.TileWidth; x++ {
			key := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: x, Y: y}}
			if n, ok := tileName[key]; ok {
				cells = append(cells, n)
			} else {
				cells = append(cells, "-")
			}
		}
		snap.Grid = append(snap.Grid, strings.Join(cells, " "))
	}

	// Level 寸法の外に置かれたタイルは上の二重ループから漏れる。エンティティは範囲無制限に列挙するのに
	// タイルだけ落ちる非対称を塞ぐため、範囲外タイルを別枠で拾う。map 反復順に依存しないよう後でソートする
	for key, name := range tileName {
		if key.X < 0 || key.X >= field.Level.TileWidth || key.Y < 0 || key.Y >= field.Level.TileHeight {
			snap.OutOfBounds = append(snap.OutOfBounds, fmt.Sprintf("%s %s", key.Coord, name))
		}
	}
	sort.Strings(snap.OutOfBounds)

	// タイル以外のエンティティを集め、(Y, X, Name, ID) の順で決定化する。ID は同一セル同一名の並びを
	// 安定させるためだけに使い、出力には含めない。ID を出すと無関係な spawn 変更でゴールデンが揺れるため
	type sortableEntity struct {
		snap EntitySnapshot
		ent  ecs.Entity
	}
	var rows []sortableEntity
	eq := query.ActiveFilter1[gc.GridElement](world).Query()
	for eq.Next() {
		e := eq.Entity()
		if world.Components.Tile.Has(e) {
			continue
		}
		g := world.Components.GridElement.Get(e)
		rows = append(rows, sortableEntity{
			snap: EntitySnapshot{Pos: g.Coord, Kind: entityKind(world, e), Name: query.GetEntityName(e, world)},
			ent:  e,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a.snap.Pos.Y != b.snap.Pos.Y {
			return a.snap.Pos.Y < b.snap.Pos.Y
		}
		if a.snap.Pos.X != b.snap.Pos.X {
			return a.snap.Pos.X < b.snap.Pos.X
		}
		if a.snap.Name != b.snap.Name {
			return a.snap.Name < b.snap.Name
		}
		return a.ent.ID() < b.ent.ID()
	})
	snap.Entities = make([]EntitySnapshot, len(rows))
	for i, r := range rows {
		snap.Entities[i] = r.snap
	}
	return snap
}

// String は WorldSnapshot を .txt ゴールデン用の決定的な文字列にする。グリッドとエンティティを見出しで分ける。
func (s WorldSnapshot) String() string {
	var b strings.Builder
	b.WriteString("# grid\n")
	for _, line := range s.Grid {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	b.WriteString("# entities\n")
	for _, e := range s.Entities {
		fmt.Fprintf(&b, "%s %s %s\n", e.Pos, e.Kind, e.Name)
	}
	if len(s.OutOfBounds) > 0 {
		b.WriteString("# out-of-bounds\n")
		for _, line := range s.OutOfBounds {
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// entityKind はエンティティの種別を代表的なタグから決める。タグを持たなければ "entity"。
func entityKind(world w.World, e ecs.Entity) string {
	switch {
	case world.Components.Player.Has(e):
		return "player"
	case world.Components.SoloAI.Has(e):
		return "ai"
	case world.Components.Door.Has(e):
		return "door"
	case world.Components.Prop.Has(e):
		return "prop"
	case world.Components.Interactable.Has(e):
		return "interactable"
	default:
		return "entity"
	}
}
