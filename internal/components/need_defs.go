package components

import (
	"image/color"

	"github.com/kijimaD/ruins/internal/i18n"

	"github.com/kijimaD/ruins/internal/oapi"
)

// HungerStage は空腹の段階1つの定義。raw の hungerStages の1行から組み立てる
type HungerStage struct {
	Level        oapi.HungerStageId
	Name         i18n.Msgid  // 表示名
	From         float64     // 満腹度の割合がこれ以上の段階のうち From が最大のものになる
	Condition    Severity    // この段階で立つ栄養失調の重症度。SeverityNone なら立たない
	ReachedLog   i18n.Msgid  // 食べてこの段階に達したときのログ。空なら出さない
	WakesSleeper bool        // 寝ている者を起こす段階か
	Badge        *color.RGBA // HUD のバッジの色。nil ならバッジを出さない
}

// FatigueStage は疲労の段階1つの定義。raw の fatigueStages の1行から組み立てる
type FatigueStage struct {
	Level     oapi.FatigueStageId
	Name      i18n.Msgid  // 表示名
	From      float64     // 疲労の割合がこれ以上の段階のうち From が最大のものになる
	Condition Severity    // この段階で立つ過労の重症度。SeverityNone なら立たない
	CanSleep  bool        // この段階で眠れるか
	Badge     *color.RGBA // HUD のバッジの色。nil ならバッジを出さない
}

// NeedDefs は空腹と疲労の段階の境目と、段階ごとに立つ不調と、鮮度の段階ごとの栄養の割合の表。
// 読み込み時に raw から1度だけ組み立て、以後は変えない
type NeedDefs struct {
	Hunger    []HungerStage
	Fatigue   []FatigueStage
	Freshness []FreshnessDef
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
func (d NeedDefs) HungerName(level oapi.HungerStageId) i18n.Msgid {
	for _, s := range d.Hunger {
		if s.Level == level {
			return s.Name
		}
	}
	return i18n.Msgid(level)
}

// FatigueName は疲労の段階の表示名 msgid を返す。未登録なら id をそのまま返す
func (d NeedDefs) FatigueName(level oapi.FatigueStageId) i18n.Msgid {
	for _, s := range d.Fatigue {
		if s.Level == level {
			return s.Name
		}
	}
	return i18n.Msgid(level)
}

// HungerStageOf は空腹の段階の定義を返す。未登録なら ok=false
func (d NeedDefs) HungerStageOf(level oapi.HungerStageId) (HungerStage, bool) {
	for _, s := range d.Hunger {
		if s.Level == level {
			return s, true
		}
	}
	return HungerStage{}, false
}

// FatigueStageOf は疲労の段階の定義を返す。未登録なら ok=false
func (d NeedDefs) FatigueStageOf(level oapi.FatigueStageId) (FatigueStage, bool) {
	for _, s := range d.Fatigue {
		if s.Level == level {
			return s, true
		}
	}
	return FatigueStage{}, false
}

// FreshnessAt は経過した段階数 stages に当たる鮮度の段階を返す。該当が無ければゼロ値
func (d NeedDefs) FreshnessAt(stages int) FreshnessDef {
	s, _ := stageAt(d.Freshness, func(s FreshnessDef) float64 { return float64(s.From) }, float64(stages))
	return s
}

// FreshnessOf は鮮度の段階の定義を返す。未登録ならゼロ値
func (d NeedDefs) FreshnessOf(level oapi.FreshnessStageId) FreshnessDef {
	for _, s := range d.Freshness {
		if s.Level == level {
			return s
		}
	}
	return FreshnessDef{}
}
