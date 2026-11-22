package components

import (
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/i18n"
	"github.com/kijimaD/ruins/internal/rawschema"
)

// Severity は状態の重症度
type Severity int

// 重症度定数
const (
	SeverityNone   Severity = iota // なし
	SeverityMinor                  // 軽度
	SeverityMedium                 // 中度
	SeveritySevere                 // 重度
)

// Name は重症度の表示名を返す
func (s Severity) Name() i18n.Msgid {
	switch s {
	case SeverityNone:
		return ""
	case SeverityMinor:
		return "Minor"
	case SeverityMedium:
		return "Medium"
	case SeveritySevere:
		return "Severe"
	default:
		panic("invalid Severity value")
	}
}

// StatType は影響を受けるステータスの種類
type StatType string

// ステータス種類定数
const (
	StatVitality  StatType = "Vitality"
	StatStrength  StatType = "Strength"
	StatSensation StatType = "Sensation"
	StatDexterity StatType = "Dexterity"
	StatAgility   StatType = "Agility"
	StatDefense   StatType = "Defense"
)

// String はステータス種類の表示名を返す
func (st StatType) String() string {
	switch st {
	case StatVitality:
		return "Vitality"
	case StatStrength:
		return "Strength"
	case StatSensation:
		return "Sensation"
	case StatDexterity:
		return "Dexterity"
	case StatAgility:
		return "Agility"
	case StatDefense:
		return "Defense"
	default:
		return string(st)
	}
}

// StatEffect はステータスへの1つの影響
type StatEffect struct {
	Stat  StatType // 影響を受けるステータス
	Value int      // 修正値
}

// ConditionType は状態の種類を識別する
type ConditionType string

// BodyFuncs は身体機能の一式。すべて基準 100 の consts.Percent で、100 が正常、
// 低いほど機能が落ちる。痛み Pain だけは 0 が無痛で、大きいほど痛い。
// 不調から読み取り時に導出し、保存はしない
type BodyFuncs struct {
	Pain          consts.Percent // 0 が無痛。大きいほど痛い
	Blood         consts.Percent // 100 が正常。失血や重い不調で下がる。危険域を下回ると HP が減る
	Consciousness consts.Percent // 100 が正常。痛みと全身性の不調で下がる。速度・命中へは消費側が掛ける
	Manipulation  consts.Percent // 腕・手の局所低下に意識を全体乗数で掛けた値。近接命中・製作
	Moving        consts.Percent // 脚・足の局所低下に意識を全体乗数で掛けた値。速度
	Sight         consts.Percent // 頭の局所低下に意識を全体乗数で掛けた値。遠隔命中
	Metabolism    consts.Percent // 100 が正常。空腹・疲労で下がる。回復の身体機能側。VIT・睡眠は熟練側で別に積む
}

// of は身体機能の種別 k の値を返す
func (b BodyFuncs) of(k BodyFuncKind) consts.Percent {
	switch k {
	case BodyFuncConsciousness:
		return b.Consciousness
	case BodyFuncManipulation:
		return b.Manipulation
	case BodyFuncMoving:
		return b.Moving
	case BodyFuncSight:
		return b.Sight
	case BodyFuncMetabolism:
		return b.Metabolism
	}
	panic("unknown body function: " + string(k))
}

// painConsciousnessDivisor は痛みが意識を下げる割合。痛みをこれで割ったぶん意識が下がる。
// 症状に依らない集約側の係数。値は実プレイで調整する
const painConsciousnessDivisor = 2

// BodyFuncKind は部位が下げる身体機能の区分。値は表示名を兼ねる
type BodyFuncKind string

// Name は身体機能の表示名を返す
func (b BodyFuncKind) Name() i18n.Msgid { return i18n.Msgid(b) }

// 身体機能の区分
const (
	BodyFuncConsciousness BodyFuncKind = "Consciousness" // 胴・全身の全身性
	BodyFuncManipulation  BodyFuncKind = "Manipulation"  // 腕・手
	BodyFuncMoving        BodyFuncKind = "Moving"        // 脚・足
	BodyFuncSight         BodyFuncKind = "Sight"         // 頭
	BodyFuncMetabolism    BodyFuncKind = "Metabolism"    // 回復の身体機能側
)

// bodyPartMeta は部位ごとの静的情報
type bodyPartMeta struct {
	id       string       // raw の bodyParts の id
	bodyFunc BodyFuncKind // この部位の不調が下げる身体機能
}

// bodyPartMetas は部位ごとの静的情報の唯一の表。部位階層は持たず平坦に持つ。
var bodyPartMetas = [BodyPartCount]bodyPartMeta{
	BodyPartHead:      {id: "head", bodyFunc: BodyFuncSight},
	BodyPartTorso:     {id: "torso", bodyFunc: BodyFuncConsciousness},
	BodyPartArms:      {id: "arm", bodyFunc: BodyFuncManipulation},
	BodyPartHands:     {id: "hand", bodyFunc: BodyFuncManipulation},
	BodyPartLegs:      {id: "leg", bodyFunc: BodyFuncMoving},
	BodyPartFeet:      {id: "foot", bodyFunc: BodyFuncMoving},
	BodyPartWholeBody: {id: "whole_body", bodyFunc: BodyFuncConsciousness},
}

// bodyPartFunction は部位が下げる身体機能を返す
func bodyPartFunction(part BodyPart) BodyFuncKind {
	return bodyPartMetas[part].bodyFunc
}

// HealthyBodyFuncs は不調が無いときの身体機能を返す。HealthStatus を持たない対象の既定
func HealthyBodyFuncs() BodyFuncs {
	return (&HealthStatus{}).BodyFuncs(ConditionDefs{})
}

// treatedPenaltyRemain は応急処置後に残る痛みと機能低下の割合。全治まで軽減して残す。
// 出血や HP 減少は治療で止まるが、機能低下はゼロにはならず全治まで引きずる
const treatedPenaltyRemain consts.Percent = 50

// BodyFuncImpact は不調1件が身体機能へ与える影響を返す。
// 身体機能は部位で定まり重症度に依らない。drop が0なら影響なし
func (d ConditionDefs) BodyFuncImpact(cond *HealthCondition, part BodyPart) (pain int, bodyFunc BodyFuncKind, drop int) {
	bodyFunc = bodyPartFunction(part)
	pain, drop = d.severityImpact(cond)
	return pain, bodyFunc, drop
}

// severityImpact は不調1件の痛みと機能低下を返す。症状ごとの反応率に重症度を掛ける。
// 応急処置済みなら軽減して返す。表示と適用がこの1関数を共有するので両者がずれない
func (d ConditionDefs) severityImpact(cond *HealthCondition) (pain, drop int) {
	m := conditionSeverityMultiplier(cond.Severity)
	if m == 0 {
		return 0, 0
	}
	def, ok := d.Def(cond.Type)
	if !ok {
		return 0, 0
	}
	pain, drop = def.PainPerSeverity*m, def.BodyFuncDropPerSeverity*m
	if cond.TendQuality > 0 {
		pain = treatedPenaltyRemain.ApplyInt(pain)
		drop = treatedPenaltyRemain.ApplyInt(drop)
	}
	return pain, drop
}

// conditionSeverityMultiplier は重症度から効果倍率を返す
func conditionSeverityMultiplier(sev Severity) int {
	switch sev {
	case SeveritySevere:
		return 3
	case SeverityMedium:
		return 2
	case SeverityMinor:
		return 1
	default:
		return 0
	}
}

// HealthCondition は部位に付与される1つの状態
type HealthCondition struct {
	Type     ConditionType // 状態の種類
	Severity Severity      // 重症度
	Timer    float64       // 進行度タイマー (0-100)
	// TendQuality は治療の質。0 なら未治療、正なら治療済みで 100 が標準、150 なら回復1.5倍。
	// 適した薬の Potency が入り、回復速度を左右する
	TendQuality consts.Percent
}

// UpdateTimer はタイマーを更新し、Severityを再計算する
// delta が正なら悪化、負なら回復
// 戻り値: (前のSeverity, 新しいSeverity)
func (hc *HealthCondition) UpdateTimer(delta float64) (Severity, Severity) {
	prevSeverity := hc.Severity
	hc.Timer = clamp(hc.Timer+delta, 0, 100)
	hc.Severity = TimerToSeverity(hc.Timer)
	return prevSeverity, hc.Severity
}

// IsActive はこの状態が発症しているかを返す
func (hc *HealthCondition) IsActive() bool {
	return hc.Timer >= 25
}

// TimerToSeverity はタイマー値からSeverityを導出する
func TimerToSeverity(timer float64) Severity {
	switch {
	case timer < 25:
		return SeverityNone
	case timer < 50:
		return SeverityMinor
	case timer < 75:
		return SeverityMedium
	default:
		return SeveritySevere
	}
}

// clamp は値を範囲内に収める
func clamp[T ~int | ~float64](val, minVal, maxVal T) T {
	if val < minVal {
		return minVal
	}
	if val > maxVal {
		return maxVal
	}
	return val
}

// BodyPartHealth は1つの部位の健康状態
type BodyPartHealth struct {
	Conditions []HealthCondition
}

// SetCondition は状態を設定する。既存の同種の状態は上書きする
func (bph *BodyPartHealth) SetCondition(cond HealthCondition) {
	for i := range bph.Conditions {
		if bph.Conditions[i].Type == cond.Type {
			bph.Conditions[i] = cond
			return
		}
	}
	bph.Conditions = append(bph.Conditions, cond)
}

// AddCondition は状態を1つ積む。SetCondition と違い同種でも上書きせず別の傷として足す。
// 戦闘の怪我のように独立した傷を重ねるのに使う
func (bph *BodyPartHealth) AddCondition(cond HealthCondition) {
	bph.Conditions = append(bph.Conditions, cond)
}

// CountConditions は指定した種類の状態がこの部位にいくつあるかを返す。独立傷のソフト上限判定に使う
func (bph *BodyPartHealth) CountConditions(condType ConditionType) int {
	n := 0
	for i := range bph.Conditions {
		if bph.Conditions[i].Type == condType {
			n++
		}
	}
	return n
}

// RemoveCondition は状態を削除する
func (bph *BodyPartHealth) RemoveCondition(condType ConditionType) {
	for i := range bph.Conditions {
		if bph.Conditions[i].Type == condType {
			bph.Conditions = append(bph.Conditions[:i], bph.Conditions[i+1:]...)
			return
		}
	}
}

// GetCondition は指定した種類の状態を取得する。存在しない場合はnil
func (bph *BodyPartHealth) GetCondition(condType ConditionType) *HealthCondition {
	for i := range bph.Conditions {
		if bph.Conditions[i].Type == condType {
			return &bph.Conditions[i]
		}
	}
	return nil
}

// GetOrCreateCondition は指定した種類の状態を取得または作成する
func (bph *BodyPartHealth) GetOrCreateCondition(condType ConditionType) *HealthCondition {
	if cond := bph.GetCondition(condType); cond != nil {
		return cond
	}
	bph.Conditions = append(bph.Conditions, HealthCondition{
		Type:     condType,
		Severity: SeverityNone,
		Timer:    0,
	})
	return &bph.Conditions[len(bph.Conditions)-1]
}

// SeverityChange はSeverityの変化を表す
type SeverityChange struct {
	CondType ConditionType
	Prev     Severity
	Current  Severity
}

// UpdateConditionTimer は指定した状態のタイマーを更新する
// 状態が存在しない場合は作成する
// タイマーが0になった状態は削除する
// 戻り値: Severityの変化情報
func (bph *BodyPartHealth) UpdateConditionTimer(condType ConditionType, delta float64) SeverityChange {
	cond := bph.GetOrCreateCondition(condType)
	prev, current := cond.UpdateTimer(delta)

	// タイマーが0になったら状態を削除
	if cond.Timer == 0 {
		bph.RemoveCondition(condType)
	}

	return SeverityChange{CondType: condType, Prev: prev, Current: current}
}

// HealthStatus は部位ごとの健康状態を管理するコンポーネント
type HealthStatus struct {
	Parts [BodyPartCount]BodyPartHealth
	// BodyTempOffset は平熱からの体温のずれ。摂氏。0 が平熱かつ上限で、寒さで負へ動く
	BodyTempOffset float64
}

// bloodCriticalThreshold は血液量がこれを下回ると失血で HP が減り始める割合。
// bloodDrainStep は血液量の不足をこれで割った数だけ毎ターン削る。深く落ちるほど速く減る。
// 外傷は出血で、重症の低体温や病気は循環低下で血液量を下げる。値は実プレイで調整する
const (
	bloodCriticalThreshold = 40
	bloodDrainStep         = 10
)

// BloodDrop は不調1件がいま下げている血液量を返す。0 なら下げない。
// 外傷は発症中ずっと出血し、重症の全身性の不調は循環を落とす。治療済みは止まる
func (d ConditionDefs) BloodDrop(cond *HealthCondition) int {
	def, ok := d.Def(cond.Type)
	if !ok || def.BloodDropPerSeverity == 0 || cond.TendQuality > 0 {
		return 0
	}
	m := conditionSeverityMultiplier(cond.Severity)
	if m == 0 {
		return 0
	}
	// 外傷は発症中ずっと血を失う。全身性の不調は重症でだけ循環が落ちる
	if def.Recovery == rawschema.RecoverAfterTend {
		if !cond.IsActive() {
			return 0
		}
	} else if cond.Severity != SeveritySevere {
		return 0
	}
	return def.BloodDropPerSeverity * m
}

// BloodLossHPDrain は血液量の低下による毎ターンの HP 減少量と死因を返す。
// 血液量が危険域を下回ると、不足に応じてじわじわ HP を削る。失血・凍死・衰弱死はすべてここを通る
func (hs *HealthStatus) BloodLossHPDrain(defs ConditionDefs) (int, DeathCause) {
	drain := BloodLossHPDrainRate(int(hs.BodyFuncs(defs).Blood))
	if drain == 0 {
		return 0, ""
	}
	return drain, hs.bloodLossCause(defs)
}

// BloodLossHPDrainRate は血液量から失血による1ターンのHP減少量を返す純関数。危険域 bloodCriticalThreshold
// 以上なら0、下回るほど段階的に増える。バランス導出が血液→HPの連鎖を単一出典で評価できるよう公開する。
func BloodLossHPDrainRate(blood int) int {
	if blood >= bloodCriticalThreshold {
		return 0
	}
	deficit := bloodCriticalThreshold - blood
	return (deficit + bloodDrainStep - 1) / bloodDrainStep
}

// bloodLossCause は血液量を最も下げている不調の死因を返す。該当が無ければ失血に落とす
func (hs *HealthStatus) bloodLossCause(defs ConditionDefs) DeathCause {
	cause := CauseBloodLoss
	best := 0
	for i := range hs.Parts {
		for j := range hs.Parts[i].Conditions {
			c := &hs.Parts[i].Conditions[j]
			if d := defs.BloodDrop(c); d > best {
				best = d
				if def := defs.conds[c.Type]; def.Cause != "" {
					cause = def.Cause
				}
			}
		}
	}
	return cause
}

// IsHPDraining はいま HP を削る要因があるかを返す。血液量が危険域なら削っている。
// 削っているあいだは自然回復を止め、減っているのを読み取れるようにする
func (hs *HealthStatus) IsHPDraining(defs ConditionDefs) bool {
	drain, _ := hs.BloodLossHPDrain(defs)
	return drain > 0
}

// BodyFuncs は不調から身体機能の一式を導出する。保存済みの値でなく Timer と Severity から計算する。
// 部位ごとの不調が対応機能を下げ、痛みと全身性の不調が意識を下げ、意識を全体乗数として局所機能へ掛ける。
// needConds に全身性の不調を渡すと保存された不調と一緒に集約する。渡さなければ保存された不調だけで導出する
func (hs *HealthStatus) BodyFuncs(defs ConditionDefs, needConds ...HealthCondition) BodyFuncs {
	pain := 0
	bloodDrop := 0
	var manip, moving, sight, systemic int // 各機能の低下量
	for i := range hs.Parts {
		for j := range hs.Parts[i].Conditions {
			cond := &hs.Parts[i].Conditions[j]
			p, drop := defs.severityImpact(cond)
			pain += p
			bloodDrop += defs.BloodDrop(cond)
			switch bodyPartFunction(BodyPart(i)) {
			case BodyFuncManipulation:
				manip += drop
			case BodyFuncMoving:
				moving += drop
			case BodyFuncSight:
				sight += drop
			case BodyFuncConsciousness:
				systemic += drop
			case BodyFuncMetabolism:
				// 代謝は部位に割り当てない。疲労・空腹の不調から下記で導く
			}
		}
	}

	// 疲労・空腹の不調は全身性。意識へ集約し、代謝を下げる量 needDrop としても集計する
	needDrop := 0
	for i := range needConds {
		p, drop := defs.severityImpact(&needConds[i])
		pain += p
		systemic += drop
		needDrop += drop
	}

	pain = clamp(pain, 0, 100)
	// 意識は全身性の低下と痛みで下がる。怪我・病気・過労・栄養失調がすべて systemic へ集約されている
	consciousness := clamp(100-systemic-pain/painConsciousnessDivisor, 0, 100)
	// 局所機能は低下を引いたうえで意識を全体乗数として掛ける
	withConsciousness := func(local int) consts.Percent {
		return consts.Percent(clamp(local, 0, 100) * consciousness / 100)
	}
	return BodyFuncs{
		Pain:          consts.Percent(pain),
		Blood:         consts.Percent(clamp(100-bloodDrop, 0, 100)),
		Consciousness: consts.Percent(consciousness),
		Manipulation:  withConsciousness(100 - manip),
		Moving:        withConsciousness(100 - moving),
		Sight:         withConsciousness(100 - sight),
		// 代謝は回復の身体機能側。過労・栄養失調でだけ下がり、痛み・全身病は含めない
		Metabolism: consts.Percent(clamp(100-needDrop, 0, 100)),
	}
}

// ConsciousnessDrops は意識を下げる不調由来の内訳を返す。全身性の低下ぶんと痛みぶんに分けて返す。
// 疲労・空腹は量由来で保存しないため含めない。呼び出し側が量から足す。内訳表示に使う
func (hs *HealthStatus) ConsciousnessDrops(defs ConditionDefs) (systemic, pain int) {
	rawPain := 0
	for i := range hs.Parts {
		for j := range hs.Parts[i].Conditions {
			cond := &hs.Parts[i].Conditions[j]
			p, drop := defs.severityImpact(cond)
			rawPain += p
			if bodyPartFunction(BodyPart(i)) == BodyFuncConsciousness {
				systemic += drop
			}
		}
	}
	return systemic, clamp(rawPain, 0, 100) / painConsciousnessDivisor
}

// LocalDrop は指定した局所身体機能を下げる部位の不調ぶんを返す。意識の 全体乗数を掛ける前の局所低下で、
// 局所機能の内訳表示に使う。全身性の Consciousness を渡すと 0 を返す
func (hs *HealthStatus) LocalDrop(defs ConditionDefs, bf BodyFuncKind) int {
	drop := 0
	for i := range hs.Parts {
		if bodyPartFunction(BodyPart(i)) != bf {
			continue
		}
		for j := range hs.Parts[i].Conditions {
			_, d := defs.severityImpact(&hs.Parts[i].Conditions[j])
			drop += d
		}
	}
	return drop
}
