package components

import "github.com/kijimaD/ruins/internal/oapi"

// HungerStage は空腹の段階1つの定義。raw の hungerStages の1行から組み立てる
type HungerStage struct {
	Level     oapi.HungerStageId
	Name      string   // 表示名 msgid
	From      float64  // 満腹度の割合がこれ以上の段階のうち From が最大のものになる
	Condition Severity // この段階で立つ栄養失調の重症度。SeverityNone なら立たない
}

// FatigueStage は疲労の段階1つの定義。raw の fatigueStages の1行から組み立てる
type FatigueStage struct {
	Level     oapi.FatigueStageId
	Name      string   // 表示名 msgid
	From      float64  // 疲労の割合がこれ以上の段階のうち From が最大のものになる
	Condition Severity // この段階で立つ過労の重症度。SeverityNone なら立たない
}

// NeedDefs は空腹と疲労の段階の境目と、段階ごとに立つ不調の表。読み込み時に raw から1度だけ組み立て、以後は変えない
type NeedDefs struct {
	Hunger  []HungerStage
	Fatigue []FatigueStage
}

// stageAt は割合 ratio が From 以上の段階のうち From が最大のものを返す。該当が無ければ ok=false
func stageAt[S any](stages []S, from func(S) float64, ratio float64) (S, bool) {
	var best S
	found := false
	for _, s := range stages {
		if from(s) <= ratio && (!found || from(s) > from(best)) {
			best, found = s, true
		}
	}
	return best, found
}

// hungerStage は満腹度の割合に当たる空腹の段階を返す。該当が無ければ満腹とする
func (d NeedDefs) hungerStage(ratio float64) HungerStage {
	s, ok := stageAt(d.Hunger, func(s HungerStage) float64 { return s.From }, ratio)
	if !ok {
		return HungerStage{Level: oapi.HungerStageIdSatiated}
	}
	return s
}

// fatigueStage は疲労の割合に当たる疲労の段階を返す。該当が無ければ快調とする
func (d NeedDefs) fatigueStage(ratio float64) FatigueStage {
	s, ok := stageAt(d.Fatigue, func(s FatigueStage) float64 { return s.From }, ratio)
	if !ok {
		return FatigueStage{Level: oapi.FatigueStageIdRested}
	}
	return s
}

// HungerSeverity は空腹段階を栄養失調の不調の重症度へ写す。ok=false なら不調は立たない
func (d NeedDefs) HungerSeverity(level oapi.HungerStageId) (Severity, bool) {
	for _, s := range d.Hunger {
		if s.Level == level {
			return s.Condition, s.Condition != SeverityNone
		}
	}
	return SeverityNone, false
}

// HungerFrom は空腹の段階が始まる満腹度の割合を返す。未登録なら 0
func (d NeedDefs) HungerFrom(level oapi.HungerStageId) float64 {
	for _, s := range d.Hunger {
		if s.Level == level {
			return s.From
		}
	}
	return 0
}

// FatigueFrom は疲労の段階が始まる疲労の割合を返す。未登録なら 0
func (d NeedDefs) FatigueFrom(level oapi.FatigueStageId) float64 {
	for _, s := range d.Fatigue {
		if s.Level == level {
			return s.From
		}
	}
	return 0
}

// HungerName は空腹の段階の表示名 msgid を返す。未登録なら id をそのまま返す
func (d NeedDefs) HungerName(level oapi.HungerStageId) string {
	for _, s := range d.Hunger {
		if s.Level == level {
			return s.Name
		}
	}
	return string(level)
}

// FatigueName は疲労の段階の表示名 msgid を返す。未登録なら id をそのまま返す
func (d NeedDefs) FatigueName(level oapi.FatigueStageId) string {
	for _, s := range d.Fatigue {
		if s.Level == level {
			return s.Name
		}
	}
	return string(level)
}
