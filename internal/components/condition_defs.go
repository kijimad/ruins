package components

import "slices"

// ConditionDef は状態種類ごとの定義。raw の conditions の1行から組み立てる。
// Recovery を持つ症状は ConditionSystem が進行と回復を扱い、持たない低体温は TemperatureSystem 管轄になる
type ConditionDef struct {
	Name                    string       // 表示名 msgid
	Description             string       // 概要説明 msgid
	PainPerSeverity         int          // 重症度1段あたりに加える痛み
	BodyFuncDropPerSeverity int          // 重症度1段あたりに下げる該当機能の量
	Recovery                RecoveryMode // 未治療の振る舞いと治し方。空なら ConditionSystem 管轄外
	WorsenPer               int          // ProgressUntilTend で未治療のとき1ターン Timer を増やす量
	RecoverPer              int          // 治療済みで1ターン Timer を減らす基準量。質と代謝で増減する
	BloodDropPerSeverity    int          // 重症度1段あたりに下げる血液量。外傷は出血、重い全身性は循環低下
	Cause                   DeathCause   // 血液量が尽きて倒したときの死因
	Parts                   []BodyPart   // 起こりうる部位。空なら全部位
}

// ConditionDefs は状態種類ごとの定義、部位ごとの命中の重み、攻撃種別ごとの怪我の表。
// 読み込み時に raw から1度だけ組み立て、以後は変えない。ゼロ値は定義を持たない表として振る舞う
type ConditionDefs struct {
	conds      map[ConditionType]ConditionDef
	hitWeights [BodyPartCount]float64
	injuries   map[string]ConditionType
}

// NewConditionDefs は状態種類ごとの定義、部位ごとの命中の重み、攻撃種別の Type ごとの怪我から表を作る。
// 渡した map は表が所有する
func NewConditionDefs(conds map[ConditionType]ConditionDef, hitWeights [BodyPartCount]float64, injuries map[string]ConditionType) ConditionDefs {
	return ConditionDefs{conds: conds, hitWeights: hitWeights, injuries: injuries}
}

// Def は状態種類の定義を返す。未登録なら ok=false
func (d ConditionDefs) Def(ct ConditionType) (ConditionDef, bool) {
	def, ok := d.conds[ct]
	return def, ok
}

// CanAffect は状態種類 ct が部位 part に起こりうるかを返す
func (d ConditionDefs) CanAffect(ct ConditionType, part BodyPart) bool {
	def, ok := d.Def(ct)
	if !ok {
		return false
	}
	return len(def.Parts) == 0 || slices.Contains(def.Parts, part)
}

// HitWeight は戦闘の命中抽選での部位の重みを返す。0 なら命中先にならない
func (d ConditionDefs) HitWeight(part BodyPart) float64 {
	return d.hitWeights[part]
}

// Injury は攻撃種別が命中で負わせる怪我を返す。未登録なら ok=false
func (d ConditionDefs) Injury(at AttackType) (ConditionType, bool) {
	ct, ok := d.injuries[at.Type]
	return ct, ok
}
