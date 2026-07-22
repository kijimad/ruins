package dungeon

import (
	"fmt"
	"math/rand/v2"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/raw"
)

// StageDefinition はステージ種別の静的マスタ。セーブに含めず StageKey.Name で引く。
// マスタ、すなわち不変の設定と、プレイ固有データ、すなわち StageKey・StageField・SeamlessBand などの
// 可変でセーブ対象のデータを分ける境界。種別はフラグでなく実装する型で表す。
type StageDefinition interface {
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

// DungeonDefinition はフロアを生成して潜る通常ダンジョンのマスタ。
// フィールドは不変なので非公開にし、アクセサ経由でのみ読む。
//
// 型名は OverworldDefinition と対になる Definition 系で、両者が同じ StageDefinition を実装することを表す。
// パッケージ名との stutter より、対称な命名で種別の並びを読みやすくする方を優先する。
//
//nolint:revive // DungeonDefinition は OverworldDefinition と対称にするため意図的にこの名前にする
type DungeonDefinition struct {
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
func (d *DungeonDefinition) Name() string { return d.name }

// BaseTemperature は基本気温を返す
func (d *DungeonDefinition) BaseTemperature() int { return d.baseTemp }

// Description はダンジョン説明文を返す
func (d *DungeonDefinition) Description() string { return d.description }

// ImageKey は背景画像のスプライトキーを返す
func (d *DungeonDefinition) ImageKey() string { return d.imageKey }

// TotalFloors は総階層数を返す
func (d *DungeonDefinition) TotalFloors() int { return d.totalFloors }

// EnemyTableName は敵テーブル名を返す
func (d *DungeonDefinition) EnemyTableName() string { return d.enemyTable }

// ItemTableName はアイテムテーブル名を返す
func (d *DungeonDefinition) ItemTableName() string { return d.itemTable }

// PlannerPool は使用するマップ種類と重みの一覧を返す。表示や検証での読み取り用。
func (d *DungeonDefinition) PlannerPool() []PlannerWeight { return d.plannerPool }

// BossPlanner は depth が最終階のときボスフロアプランナーを返す。
// ボスフロアがない、または最終階でなければ ok=false を返す。
func (d *DungeonDefinition) BossPlanner(depth int) (mapplanner.PlannerType, bool) {
	if d.bossPlanner != nil && depth == d.totalFloors {
		return *d.bossPlanner, true
	}
	return mapplanner.PlannerType{}, false
}

// SelectPlanner は PlannerPool から重み付き抽選で PlannerType を選ぶ。
// プランナー抽選はフロアを生成するダンジョン固有の振る舞いなのでこの型のメソッドにする。
func (d *DungeonDefinition) SelectPlanner(rng *rand.Rand) (mapplanner.PlannerType, error) {
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

// OverworldDefinition は帯をスライドし続けるオーバーワールドのマスタ。
// フロアを生成しないので、ダンジョン専用のテーブルやプランナーを持たない。
// 帯形状 chunkW/chunkH/k は静的な設定なのでマスタが持つ。RunSeed はプレイごとに変わるため
// プレイ固有データ SeamlessBand が持ち、ここには含めない。
type OverworldDefinition struct {
	name     string
	baseTemp int
	chunkW   consts.Tile
	chunkH   consts.Tile
	k        consts.Chunk
}

// NewOverworldDefinition はオーバーワールド種別を構成する。帯形状を含む設定を渡す。
// 本番は登録済みの DungeonOverworld を使い、テストは任意形状の種別を組むのに使う。
func NewOverworldDefinition(name string, baseTemp int, chunkW, chunkH consts.Tile, k consts.Chunk) *OverworldDefinition {
	return &OverworldDefinition{name: name, baseTemp: baseTemp, chunkW: chunkW, chunkH: chunkH, k: k}
}

// Name はオーバーワールドの識別名を返す
func (o *OverworldDefinition) Name() string { return o.name }

// BaseTemperature は基本気温を返す
func (o *OverworldDefinition) BaseTemperature() int { return o.baseTemp }

// BandShape は帯の形状、1チャンクの幅と高さ、チャンク数を返す。RunSeed は含まない。
func (o *OverworldDefinition) BandShape() (chunkW, chunkH consts.Tile, k consts.Chunk) {
	return o.chunkW, o.chunkH, o.k
}
