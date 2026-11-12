package balance

import (
	"testing"

	"github.com/kijimaD/ruins/internal/raw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateReport_プレイヤーと武器と敵テーブルの結果を含むレポートを生成する(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)

	report, err := GenerateReport(master, "ash", "bare_hands", 3, 5, 42)
	require.NoError(t, err)

	assert.Equal(t, "ash", report.Player.Name)
	assert.Equal(t, 80, report.Player.HP)

	assert.Equal(t, "bare_hands", report.Weapon.Name)

	require.Len(t, report.EnemyTables, len(raw.PtrSlice(master.EnemyTables)), "raw.tomlのenemyTables数と一致する")
	for _, run := range report.EnemyTables {
		assert.Equal(t, 3, run.MaxDepth)
		assert.Equal(t, 5, run.Trials)
		assert.NotEmpty(t, run.Depths, "深度1には必ず到達する")
	}

	assert.NotEmpty(t, report.BattleMetrics, "武器×敵の組み合わせのメトリクスが生成される")
}

func TestGenerateReport_存在しないプレイヤーはエラー(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)

	_, err := GenerateReport(master, "存在しないプレイヤー", "bare_hands", 1, 1, 1)
	var notFound raw.KeyNotFoundError
	require.ErrorAs(t, err, &notFound)
	assert.Equal(t, raw.KeyNotFoundError{Key: "存在しないプレイヤー", Collection: "Members"}, notFound)
}

func TestGenerateReport_存在しない武器はエラー(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)

	_, err := GenerateReport(master, "ash", "存在しない武器", 1, 1, 1)
	var notFound raw.KeyNotFoundError
	require.ErrorAs(t, err, &notFound)
	assert.Equal(t, raw.KeyNotFoundError{Key: "存在しない武器", Collection: "Items"}, notFound)
}

func TestGenerateBattleMetrics_存在しないプレイヤーはエラー(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)

	metrics, err := generateBattleMetrics(master, "存在しないプレイヤー", 1)
	require.Error(t, err)
	assert.Nil(t, metrics)
}

func TestGenerateBattleMetrics_武器と敵の組み合わせでDPSを算出する(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)

	metrics, err := generateBattleMetrics(master, "ash", 42)
	require.NoError(t, err)
	require.NotEmpty(t, metrics)
	for _, m := range metrics {
		assert.Equal(t, "ash", m.Player)
		assert.GreaterOrEqual(t, m.DPS, 0.0)
	}
}

func TestGenerateBattleMetrics_同じseedなら同じ結果になる(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)

	first, err := generateBattleMetrics(master, "ash", 42)
	require.NoError(t, err)
	second, err := generateBattleMetrics(master, "ash", 42)
	require.NoError(t, err)
	assert.Equal(t, first, second)
}

func TestReport_Markdown(t *testing.T) {
	t.Parallel()

	r := &Report{
		Player: PlayerInfo{Name: "ash", HP: 80, Strength: 5, Sensation: 4, Dexterity: 3, Agility: 2, Defense: 1},
		Weapon: WeaponInfo{Name: "bare_hands", Damage: 3, Accuracy: 90},
		EnemyTables: []EnemyTableRun{{
			Name: "廃墟", Trials: 10, MaxDepth: 3, MedianDepth: 2, DeathRate: 0.25,
			Depths: []DepthStat{{Depth: 1, MedianHP: 70, P5HP: 50, P95HP: 80, MedianHPBeforeHeal: 60, MedianKillTurns: 4, MedianWeaponDamage: 3, MedianHunger: 10, MedianDamage: 12, MedianHealing: 5, SuddenDeathRate: 0.1}},
		}},
		BattleMetrics: []BattleMetric{{Player: "ash", Weapon: "bare_hands", Enemy: "slime", DPS: 1.5}},
		RoomLoot:      []FacilityLoot{{Facility: "house", Trials: 20, Rooms: []RoomLoot{{Role: "kitchen", Items: []LootItemStat{{Name: "bread", Prob: 0.5, ExpectedCount: 0.75, Value: 10}}}}}},
	}

	assert.Equal(t, `# バランスシミュレーション

プレイヤー ash。HP 80、筋力 5、感覚 4、器用 3、敏捷 2、防御 1。

武器 bare_hands。ダメージ 3、命中 90。

## 敵テーブル

### 廃墟

試行 10、最大深度 3、到達深度の中央値 2、死亡率 25.0%。

| 深度 | HP 中央値 | HP p5 | HP p95 | 回復前 HP 中央値 | 撃破ターン中央値 | 武器ダメージ中央値 | 空腹中央値 | 被ダメージ中央値 | 回復中央値 | 急死率 |
|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 70 | 50 | 80 | 60 | 4 | 3 | 10 | 12 | 5 | 10.0% |

## 武器×敵の戦闘

| 武器 | 敵 | DPS | 遠距離 |
|---|---|---|---|
| bare_hands | slime | 1.50 | false |

## 施設の戦利品

### house

試行 20。

| 部屋 | アイテム | 1棟で出る確率 | 1棟あたりの期待個数 | 売買価値 |
|---|---|---|---|---|
| kitchen | bread | 50.0% | 0.75 | 10 |

`, r.Markdown())
}
