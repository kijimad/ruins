package mapplanner

import (
	"errors"
	"fmt"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
)

var (
	// ErrUnknownLayout は間取りの表に無い id を指したときのエラー
	ErrUnknownLayout = errors.New("unknown layout")
	// ErrNoLayout はランダムに選べる間取りの行が無いときのエラー
	ErrNoLayout = errors.New("no layout")
	// ErrUnknownStep は段の表に無い id を指したときのエラー
	ErrUnknownStep = errors.New("unknown layout step")
	// ErrFloorCountUnset は敵やアイテムの表を指すのにフロアに置く数を持たないプランナーのエラー
	ErrFloorCountUnset = errors.New("floor count is not set")
)

// LayoutPlannerType は raw の間取りの行 id のプランナータイプを返す。行は計画を作るときに引く
func LayoutPlannerType(id string) PlannerType {
	return PlannerType{
		Name: id,
		PlannerFunc: func(raws rawschema.Raws, width, height consts.Tile, seed uint64) (*PlannerChain, error) {
			return newLayoutChain(raws, id, width, height, seed)
		},
	}
}

// LayoutPlannerTypes は全ての間取りの行のプランナータイプを行の順に返す
func LayoutPlannerTypes(raws rawschema.Raws) []PlannerType {
	layouts := raw.PtrSlice(raws.Layouts)
	out := make([]PlannerType, 0, len(layouts))
	for i := range layouts {
		out = append(out, LayoutPlannerType(layouts[i].Id))
	}
	return out
}

// ProceduralPlannerTypes は絵を使わない間取りの行のプランナータイプを行の順に返す
func ProceduralPlannerTypes(raws rawschema.Raws) []PlannerType {
	ids := proceduralLayoutIDs(raws)
	out := make([]PlannerType, 0, len(ids))
	for _, id := range ids {
		out = append(out, LayoutPlannerType(id))
	}
	return out
}

// PlannerTypeByName は間取りの行 id のプランナータイプを返す。行が無ければ ok=false を返す
func PlannerTypeByName(raws rawschema.Raws, name string) (PlannerType, bool) {
	if _, ok := raw.FindByKey(raws.Layouts, func(l rawschema.Layout) string { return l.Id }, name); !ok {
		return PlannerType{}, false
	}
	return LayoutPlannerType(name), true
}

// proceduralLayoutIDs は最初の段が絵でない間取りの行の id を行の順に返す。ランダムに選ぶ候補になる
func proceduralLayoutIDs(raws rawschema.Raws) []string {
	layouts := raw.PtrSlice(raws.Layouts)
	var ids []string
	for i := range layouts {
		if _, ok := templateStart(raws, layouts[i].Start); ok {
			continue
		}
		ids = append(ids, layouts[i].Id)
	}
	return ids
}

func templateStart(raws rawschema.Raws, id string) (rawschema.TemplateStart, bool) {
	return raw.FindByKey(raws.TemplateStarts, func(s rawschema.TemplateStart) string { return s.Id }, id)
}

// newLayoutChain は間取りの行 id を引き、最初の段と中の段を並べ、最後に環境の段を足したチェーンを作る。
// 絵の最初の段は地図の大きさを絵から決めるので width と height を使わない
func newLayoutChain(raws rawschema.Raws, id string, width, height consts.Tile, seed uint64) (*PlannerChain, error) {
	l, ok := raw.FindByKey(raws.Layouts, func(l rawschema.Layout) string { return l.Id }, id)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownLayout, id)
	}
	var chain *PlannerChain
	if t, ok := templateStart(raws, l.Start); ok {
		c, err := NewPlannerChainByTemplateName(t.Template, seed)
		if err != nil {
			return nil, err
		}
		chain = c
	} else {
		start, err := newStart(raws, l.Start, l.Tiles)
		if err != nil {
			return nil, err
		}
		chain = NewPlannerChain(width, height, seed)
		chain.StartWith(start)
	}
	chain.PlanData.NeedsGround = usesGround(l.Tiles)
	for _, sid := range l.Steps {
		step, err := newStep(raws, sid, l.Tiles)
		if err != nil {
			return nil, err
		}
		chain.With(step)
	}
	chain.With(EnvironmentPlanner{})
	return chain, nil
}

// newStart は絵でない最初の段の行 id を表から引き、間取りのタイル tiles を渡して段を作る。
// 段の表を足すときは、schema.cue の表と startIds_unique、refs.cue の _startIDs、この分岐、TestNewStart の表の列挙を揃える
func newStart(raws rawschema.Raws, id string, tiles rawschema.LayoutTiles) (InitialMapPlanner, error) {
	if r, ok := raw.FindByKey(raws.RectRoomStarts, func(r rawschema.RectRoomStart) string { return r.Id }, id); ok {
		return RectRoomPlanner{Row: r}, nil
	}
	if r, ok := raw.FindByKey(raws.NoiseStarts, func(r rawschema.NoiseStart) string { return r.Id }, id); ok {
		return CavePlanner{Row: r, Tiles: tiles}, nil
	}
	if r, ok := raw.FindByKey(raws.BuildingStarts, func(r rawschema.BuildingStart) string { return r.Id }, id); ok {
		return RuinsPlanner{Row: r}, nil
	}
	if r, ok := raw.FindByKey(raws.ClearingStarts, func(r rawschema.ClearingStart) string { return r.Id }, id); ok {
		return ForestPlanner{Row: r}, nil
	}
	if r, ok := raw.FindByKey(raws.SimpleStarts, func(r rawschema.SimpleStart) string { return r.Id }, id); ok {
		if r.Kind == rawschema.SimpleStartKindFullRoom {
			return BigRoomPlanner{}, nil
		}
		return nil, fmt.Errorf("%w: %s kind %s", ErrUnknownStep, id, r.Kind)
	}
	return nil, fmt.Errorf("%w: %s", ErrUnknownStep, id)
}

// newStep は中の段の行 id を表から引き、間取りのタイル tiles を渡して段を作る。
// 段の表を足すときは、schema.cue の表と stepIds_unique、refs.cue の _stepIDs、この分岐、TestNewStep の表の列挙を揃える
func newStep(raws rawschema.Raws, id string, tiles rawschema.LayoutTiles) (MetaMapPlanner, error) {
	if r, ok := raw.FindByKey(raws.CorridorSteps, func(r rawschema.CorridorStep) string { return r.Id }, id); ok {
		return LineCorridorPlanner{FloorTile: tiles.Floor, Width: r.Width}, nil
	}
	if r, ok := raw.FindByKey(raws.DoorSteps, func(r rawschema.DoorStep) string { return r.Id }, id); ok {
		return DoorPlanner{DoorChance: r.Chance}, nil
	}
	if r, ok := raw.FindByKey(raws.BigRoomSteps, func(r rawschema.BigRoomStep) string { return r.Id }, id); ok {
		return BigRoomDraw{Row: r, Tiles: tiles}, nil
	}
	if r, ok := raw.FindByKey(raws.AutomataSteps, func(r rawschema.AutomataStep) string { return r.Id }, id); ok {
		return CaveCellularAutomata{Row: r, Tiles: tiles}, nil
	}
	if r, ok := raw.FindByKey(raws.WidenSteps, func(r rawschema.WidenStep) string { return r.Id }, id); ok {
		return Widen{Row: r, Tiles: tiles}, nil
	}
	if r, ok := raw.FindByKey(raws.ScatterWallSteps, func(r rawschema.ScatterWallStep) string { return r.Id }, id); ok {
		return ScatterWalls{Row: r, Tiles: tiles}, nil
	}
	if r, ok := raw.FindByKey(raws.RuinedWallSteps, func(r rawschema.RuinedWallStep) string { return r.Id }, id); ok {
		return RuinsDraw{Row: r, Tiles: tiles}, nil
	}
	if r, ok := raw.FindByKey(raws.RuinedPathSteps, func(r rawschema.RuinedPathStep) string { return r.Id }, id); ok {
		return RuinsCorridors{Row: r, Tiles: tiles}, nil
	}
	if r, ok := raw.FindByKey(raws.TreeSteps, func(r rawschema.TreeStep) string { return r.Id }, id); ok {
		return ForestTrees{Row: r, Tiles: tiles}, nil
	}
	if r, ok := raw.FindByKey(raws.TrailSteps, func(r rawschema.TrailStep) string { return r.Id }, id); ok {
		return ForestPaths{Row: r}, nil
	}
	if r, ok := raw.FindByKey(raws.GladeSteps, func(r rawschema.GladeStep) string { return r.Id }, id); ok {
		return ForestWildlife{Row: r, Tiles: tiles}, nil
	}
	if r, ok := raw.FindByKey(raws.SimpleSteps, func(r rawschema.SimpleStep) string { return r.Id }, id); ok {
		return newSimpleStep(r, tiles)
	}
	return nil, fmt.Errorf("%w: %s", ErrUnknownStep, id)
}

// newSimpleStep は欄を持たない中の段を種類から作る
func newSimpleStep(r rawschema.SimpleStep, tiles rawschema.LayoutTiles) (MetaMapPlanner, error) {
	switch r.Kind {
	case rawschema.SimpleStepKindFillWall:
		return NewFillAll(tiles.Wall), nil
	case rawschema.SimpleStepKindDrawRooms:
		return RoomDraw{FloorTile: tiles.Floor}, nil
	case rawschema.SimpleStepKindConnectRooms:
		return ConnectRooms{Tiles: tiles}, nil
	case rawschema.SimpleStepKindDebris:
		return RuinsDebris{Tiles: tiles}, nil
	case rawschema.SimpleStepKindForestFloor:
		return ForestTerrain{Tiles: tiles}, nil
	case rawschema.SimpleStepKindIsolatedWalls:
		return ConvertIsolatedWalls{ReplacementTile: tiles.Void}, nil
	}
	return nil, fmt.Errorf("%w: %s kind %s", ErrUnknownStep, r.Id, r.Kind)
}

// usesGround は間取りのタイルが下地を指すかを返す
func usesGround(t rawschema.LayoutTiles) bool {
	return t.Floor == maptemplate.GroundTerrain || t.Wall == maptemplate.GroundTerrain || t.Void == maptemplate.GroundTerrain
}
