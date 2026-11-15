package mapplanner

import (
	"errors"
	"fmt"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
)

var (
	// ErrUnknownLayout は間取りの表に無い id を指したときのエラー
	ErrUnknownLayout = errors.New("unknown layout")
	// ErrNoLayout はランダムに選べる間取りの行が無いときのエラー
	ErrNoLayout = errors.New("no layout")
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

// LayoutPlannerTypes は全ての間取りの行のプランナータイプを表の順に返す
func LayoutPlannerTypes(raws rawschema.Raws) []PlannerType {
	ids := appendLayoutIDs(proceduralLayoutIDs(raws), raws.TemplateLayouts, func(l *rawschema.TemplateLayout) string { return l.Id })
	return plannerTypesOf(ids)
}

// ProceduralPlannerTypes は絵を使わない間取りの行のプランナータイプを表の順に返す
func ProceduralPlannerTypes(raws rawschema.Raws) []PlannerType {
	return plannerTypesOf(proceduralLayoutIDs(raws))
}

func plannerTypesOf(ids []string) []PlannerType {
	out := make([]PlannerType, 0, len(ids))
	for _, id := range ids {
		out = append(out, LayoutPlannerType(id))
	}
	return out
}

// PlannerTypeByName は間取りの行 id のプランナータイプを返す。行が無ければ ok=false を返す
func PlannerTypeByName(raws rawschema.Raws, name string) (PlannerType, bool) {
	for _, pt := range LayoutPlannerTypes(raws) {
		if pt.Name == name {
			return pt, true
		}
	}
	return PlannerType{}, false
}

// proceduralLayoutIDs は絵を使わない間取りの行の id を表の順に返す。ランダムに選ぶ候補になる
func proceduralLayoutIDs(raws rawschema.Raws) []string {
	var ids []string
	ids = appendLayoutIDs(ids, raws.RoomLayouts, func(l *rawschema.RoomLayout) string { return l.Id })
	ids = appendLayoutIDs(ids, raws.BigRoomLayouts, func(l *rawschema.BigRoomLayout) string { return l.Id })
	ids = appendLayoutIDs(ids, raws.CaveLayouts, func(l *rawschema.CaveLayout) string { return l.Id })
	ids = appendLayoutIDs(ids, raws.RuinsLayouts, func(l *rawschema.RuinsLayout) string { return l.Id })
	ids = appendLayoutIDs(ids, raws.ForestLayouts, func(l *rawschema.ForestLayout) string { return l.Id })
	return ids
}

func appendLayoutIDs[T any](ids []string, rows *[]T, id func(*T) string) []string {
	ls := raw.PtrSlice(rows)
	for i := range ls {
		ids = append(ids, id(&ls[i]))
	}
	return ids
}

// newLayoutChain は間取りの行 id を表から引き、その作り方のチェーンを作成する
func newLayoutChain(raws rawschema.Raws, id string, width, height consts.Tile, seed uint64) (*PlannerChain, error) {
	if l, ok := raw.FindByKey(raws.RoomLayouts, func(l rawschema.RoomLayout) string { return l.Id }, id); ok {
		return NewSmallRoomPlanner(l, width, height, seed)
	}
	if l, ok := raw.FindByKey(raws.BigRoomLayouts, func(l rawschema.BigRoomLayout) string { return l.Id }, id); ok {
		return NewBigRoomPlanner(l, width, height, seed)
	}
	if l, ok := raw.FindByKey(raws.CaveLayouts, func(l rawschema.CaveLayout) string { return l.Id }, id); ok {
		return NewCavePlanner(l, width, height, seed)
	}
	if l, ok := raw.FindByKey(raws.RuinsLayouts, func(l rawschema.RuinsLayout) string { return l.Id }, id); ok {
		return NewRuinsPlanner(l, width, height, seed)
	}
	if l, ok := raw.FindByKey(raws.ForestLayouts, func(l rawschema.ForestLayout) string { return l.Id }, id); ok {
		return NewForestPlanner(l, width, height, seed)
	}
	if l, ok := raw.FindByKey(raws.TemplateLayouts, func(l rawschema.TemplateLayout) string { return l.Id }, id); ok {
		return NewPlannerChainByTemplateName(l.Template, seed)
	}
	return nil, fmt.Errorf("%w: %s", ErrUnknownLayout, id)
}
