package activity

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// SleepBehavior は疲労が抜けるまで眠り続けるアクティビティ。rest と違い必要 AP を消化するのでなく、
// 毎ターン疲労が減り、疲労が尽きるか外因で中断されると終わる。起きる時間は指定できない。
type SleepBehavior struct{}

// Info はBehaviorの実装
func (sb *SleepBehavior) Info() Info {
	return Info{
		Name:          "Sleep",
		Description:   "Sleep until fatigue is gone",
		Interruptible: true,
		// 中断は Cancel で目覚めとして扱い、再入眠は B キーで新規に Execute する。
		// Pause からの Resume 経路は使わないので false にし、寝具 Quality の再解決漏れも避ける
		Resumable:       false,
		ActionPointCost: consts.StandardActionCost,
		TotalRequiredAP: 0,
	}
}

// Name はBehaviorの実装
func (sb *SleepBehavior) Name() gc.BehaviorName {
	return gc.BehaviorSleep
}

// NewSleepActivity は睡眠アクティビティを組む。進行は疲労側で管理するので必要 AP は 0
func NewSleepActivity() *gc.Activity {
	return NewActivity(gc.BehaviorSleep, 0)
}

// Validate は入眠可否を疲労・気温・安全の全条件で検証する。気温計算は query に降りたので
// この層で全条件を評価でき、プロンプト表示と同じ EvaluateSleepConditions を唯一の判定点にする。
// 失敗の種別ごとに理由を返し、呼び出し側が gamelog へ出す
func (sb *SleepBehavior) Validate(_ *gc.Activity, actor ecs.Entity, world w.World) error {
	sc := EvaluateSleepConditions(world, actor)
	switch {
	case sc.TooTired():
		return &UserError{Msg: query.T(world, "Not tired enough to sleep.")}
	case !sc.TemperatureOK:
		return &UserError{Msg: query.T(world, "Too cold or hot to sleep.")}
	case !sc.AreaSafe:
		return &UserError{Msg: query.T(world, "Cannot sleep because enemies are nearby.")}
	}
	return nil
}

// Start は睡眠開始時に Sleeping マーカーを付ける。寝具 Quality を足元・隣接から解決して写す
func (sb *SleepBehavior) Start(_ *gc.Activity, actor ecs.Entity, world w.World) error {
	quality := beddingQualityAt(actor, world)
	if !world.Components.Sleeping.Has(actor) {
		world.Components.Sleeping.Add(actor, &gc.Sleeping{Quality: quality})
	}
	if world.Components.Player.Has(actor) {
		gamelog.New(query.GetGameLog(world)).
			Markup(query.T(world, "Went to sleep.")).
			Log()
	}
	return nil
}

// DoTurn は毎ターン中断条件を検査し、満たさなければ眠り続ける。疲労の減少はターン終了の
// progressTurnFatigue が Sleeping を見て行うので、ここでは起床・中断の判定に徹する
func (sb *SleepBehavior) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	if !query.IsAreaSafe(world, actor) {
		Cancel(comp, "sleep interrupted because enemies are nearby")
		return nil
	}
	if hasActiveHypothermia(actor, world) {
		Cancel(comp, "woke up from the cold")
		return nil
	}
	if isStarving(actor, world) {
		Cancel(comp, "woke up from hunger")
		return nil
	}
	if world.Components.Fatigue.Has(actor) && world.Components.Fatigue.Get(actor).Current <= 0 {
		Complete(comp)
	}
	return nil
}

// Finish は起床時に Sleeping を外す
func (sb *SleepBehavior) Finish(_ *gc.Activity, actor ecs.Entity, world w.World) error {
	removeSleeping(actor, world)
	if world.Components.Player.Has(actor) {
		gamelog.New(query.GetGameLog(world)).
			Markup(query.T(world, "Woke up refreshed.")).
			Log()
	}
	return nil
}

// Canceled は中断時に Sleeping を外す
func (sb *SleepBehavior) Canceled(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	// Sleeping の除去はアーキタイプを変えて comp ポインタを無効化する。理由は除去より前に退避する
	reason := comp.CancelReason
	removeSleeping(actor, world)
	if world.Components.Player.Has(actor) {
		gamelog.New(query.GetGameLog(world)).
			Markup(query.T(world, "Sleep interrupted: %s", query.T(world, reason))).
			Log()
	}
	return nil
}

// removeSleeping は Sleeping マーカーを外す。二重除去でも安全なよう Has でガードする
func removeSleeping(actor ecs.Entity, world w.World) {
	if world.Components.Sleeping.Has(actor) {
		world.Components.Sleeping.Remove(actor)
	}
}

// hasActiveHypothermia は全身の低体温が体感に響く水準に達しているかを返す。
// わずかな残留では起こさず、IsActive すなわちペナルティを及ぼす段階で初めて起床させる。
// 入眠可能な適温帯の下端で寝ると低体温が浅く残るが、火のそばで回復中なら閾値に達さず眠り続けられる
func hasActiveHypothermia(actor ecs.Entity, world w.World) bool {
	if !world.Components.HealthStatus.Has(actor) {
		return false
	}
	hs := world.Components.HealthStatus.Get(actor)
	cond := hs.Parts[gc.BodyPartWholeBody].GetCondition(gc.ConditionHypothermia)
	return cond != nil && cond.IsActive()
}

// isStarving は飢餓段階かを返す
func isStarving(actor ecs.Entity, world w.World) bool {
	return world.Components.Hunger.Has(actor) &&
		world.Components.Hunger.Get(actor).GetLevel() == gc.HungerStarving
}

// beddingQualityAt は足元か隣接の寝具から最大の Quality を返す。無ければ地べたの基準
func beddingQualityAt(actor ecs.Entity, world w.World) consts.Percent {
	quality := consts.PercentBase
	if !world.Components.GridElement.Has(actor) {
		return quality
	}
	center := world.Components.GridElement.Get(actor).Coord
	for dy := consts.Tile(-1); dy <= 1; dy++ {
		for dx := consts.Tile(-1); dx <= 1; dx++ {
			for _, e := range query.GetEntitiesAt(world, center.X+dx, center.Y+dy) {
				if world.Components.Bedding.Has(e) {
					if q := world.Components.Bedding.Get(e).Quality; q > quality {
						quality = q
					}
				}
			}
		}
	}
	return quality
}
