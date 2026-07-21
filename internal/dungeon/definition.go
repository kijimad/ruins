package dungeon

import (
	"fmt"
	"math/rand/v2"

	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/raw"
)

// StageKind はステージ種別の静的マスタ。セーブに含めず StageKey.Name で引く。
// マスタ、すなわち不変の設定と、プレイ固有データ、すなわち StageKey・StageMeta・SeamlessBand などの
// 可変でセーブ対象のデータを分ける境界。種別はフラグでなく実装する型で表す。
type StageKind interface {
	// Name は種別の識別名。StageKey.Name と一致し、登録表を引くキーになる
	Name() string
	// BaseTemperature は基本気温。摂氏
	BaseTemperature() int
}

// PlannerWeight はマップ種類と出現重みのペアを表す
type PlannerWeight struct {
	PlannerType mapplanner.PlannerType
	Weight      float64
}

// DungeonKind はフロアを生成して潜る通常ダンジョンのマスタ。
// フィールドは不変なので非公開にし、アクセサ経由でのみ読む。
//
// 型名は OverworldKind と対になる Kind 系で、両者が同じ StageKind を実装することを表す。
// パッケージ名との stutter より、対称な命名で種別の並びを読みやすくする方を優先する。
//
//nolint:revive // DungeonKind は OverworldKind と対称にするため意図的にこの名前にする
type DungeonKind struct {
	name        string
	description string
	imageKey    string
	totalFloors int
	enemyTable  string
	itemTable   string
	plannerPool []PlannerWeight
	baseTemp    int
	// bossPlanner は最終階で使うボスフロアプランナー。nil ならボスフロアなし
	bossPlanner *mapplanner.PlannerType
}

// Name はダンジョン名を返す
func (d *DungeonKind) Name() string { return d.name }

// BaseTemperature は基本気温を返す
func (d *DungeonKind) BaseTemperature() int { return d.baseTemp }

// Description はダンジョン説明文を返す
func (d *DungeonKind) Description() string { return d.description }

// ImageKey は背景画像のスプライトキーを返す
func (d *DungeonKind) ImageKey() string { return d.imageKey }

// TotalFloors は総階層数を返す
func (d *DungeonKind) TotalFloors() int { return d.totalFloors }

// EnemyTableName は敵テーブル名を返す
func (d *DungeonKind) EnemyTableName() string { return d.enemyTable }

// ItemTableName はアイテムテーブル名を返す
func (d *DungeonKind) ItemTableName() string { return d.itemTable }

// PlannerPool は使用するマップ種類と重みの一覧を返す。表示や検証での読み取り用。
func (d *DungeonKind) PlannerPool() []PlannerWeight { return d.plannerPool }

// BossPlanner は depth が最終階のときボスフロアプランナーを返す。
// ボスフロアがない、または最終階でなければ ok=false を返す。
func (d *DungeonKind) BossPlanner(depth int) (mapplanner.PlannerType, bool) {
	if d.bossPlanner != nil && depth == d.totalFloors {
		return *d.bossPlanner, true
	}
	return mapplanner.PlannerType{}, false
}

// SelectPlanner は PlannerPool から重み付き抽選で PlannerType を選ぶ。
// プランナー抽選はフロアを生成するダンジョン固有の振る舞いなのでこの型のメソッドにする。
func (d *DungeonKind) SelectPlanner(rng *rand.Rand) (mapplanner.PlannerType, error) {
	if len(d.plannerPool) == 0 {
		return mapplanner.PlannerType{}, fmt.Errorf("PlannerPoolが空です: %s", d.name)
	}

	result, err := raw.SelectByWeightFunc(
		d.plannerPool,
		func(pw PlannerWeight) float64 { return pw.Weight },
		func(pw PlannerWeight) mapplanner.PlannerType { return pw.PlannerType },
		rng,
	)
	if err != nil {
		return mapplanner.PlannerType{}, err
	}

	if result.Name == "" {
		return mapplanner.PlannerType{}, fmt.Errorf("PlannerPoolの総重みが0です: %s", d.name)
	}

	return result, nil
}

// OverworldKind は帯をスライドし続けるオーバーワールドのマスタ。
// フロアを生成しないので、ダンジョン専用のテーブルやプランナーを持たない。
type OverworldKind struct {
	name     string
	baseTemp int
}

// Name はオーバーワールドの識別名を返す
func (o *OverworldKind) Name() string { return o.name }

// BaseTemperature は基本気温を返す
func (o *OverworldKind) BaseTemperature() int { return o.baseTemp }
