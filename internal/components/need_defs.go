package components

// HungerStage は空腹の段階1つの定義。raw の hungerStages の1行から組み立てる
type HungerStage struct {
	Level     HungerLevel
	From      float64  // 満腹度の割合がこれ以上の段階のうち From が最大のものになる
	Condition Severity // この段階で立つ栄養失調の重症度。SeverityNone なら立たない
}

// FatigueStage は疲労の段階1つの定義。raw の fatigueStages の1行から組み立てる
type FatigueStage struct {
	Level     FatigueLevel
	From      float64  // 疲労の割合がこれ以上の段階のうち From が最大のものになる
	Condition Severity // この段階で立つ過労の重症度。SeverityNone なら立たない
}

// NeedDefs は空腹と疲労の段階の境目と、段階ごとに立つ不調の表。読み込み時に raw から1度だけ組み立て、以後は変えない
type NeedDefs struct {
	hunger  []HungerStage
	fatigue []FatigueStage
}

// NewNeedDefs は空腹と疲労の段階から表を作る。渡したスライスは表が所有する
func NewNeedDefs(hunger []HungerStage, fatigue []FatigueStage) NeedDefs {
	return NeedDefs{hunger: hunger, fatigue: fatigue}
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
	s, ok := stageAt(d.hunger, func(s HungerStage) float64 { return s.From }, ratio)
	if !ok {
		return HungerStage{Level: HungerSatiated}
	}
	return s
}

// fatigueStage は疲労の割合に当たる疲労の段階を返す。該当が無ければ快調とする
func (d NeedDefs) fatigueStage(ratio float64) FatigueStage {
	s, ok := stageAt(d.fatigue, func(s FatigueStage) float64 { return s.From }, ratio)
	if !ok {
		return FatigueStage{Level: FatigueRested}
	}
	return s
}

// HungerSeverity は空腹段階を栄養失調の不調の重症度へ写す。ok=false なら不調は立たない
func (d NeedDefs) HungerSeverity(level HungerLevel) (Severity, bool) {
	for _, s := range d.hunger {
		if s.Level == level {
			return s.Condition, s.Condition != SeverityNone
		}
	}
	return SeverityNone, false
}

// HungerFrom は空腹の段階が始まる満腹度の割合を返す。未登録なら 0
func (d NeedDefs) HungerFrom(level HungerLevel) float64 {
	for _, s := range d.hunger {
		if s.Level == level {
			return s.From
		}
	}
	return 0
}

// FatigueFrom は疲労の段階が始まる疲労の割合を返す。未登録なら 0
func (d NeedDefs) FatigueFrom(level FatigueLevel) float64 {
	for _, s := range d.fatigue {
		if s.Level == level {
			return s.From
		}
	}
	return 0
}
