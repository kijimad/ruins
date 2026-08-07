package balance

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateReport_プレイヤーと武器と敵テーブルの結果を含むレポートを生成する(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)

	report, err := GenerateReport(master, "Ash", "素手", 3, 5, 42)
	require.NoError(t, err)

	assert.Equal(t, "simple", report.Mode)

	require.NotNil(t, report.Player)
	assert.Equal(t, "Ash", report.Player.Name)
	assert.Equal(t, 80, report.Player.HP)

	require.NotNil(t, report.Weapon)
	assert.Equal(t, "素手", report.Weapon.Name)

	require.Len(t, report.EnemyTables, 3, "raw.tomlのenemyTables数と一致する")
	for _, run := range report.EnemyTables {
		assert.Equal(t, 3, run.MaxDepth)
		assert.Equal(t, 5, run.Trials)
		assert.NotEmpty(t, run.Depths, "深度1には必ず到達する")
		assert.Len(t, run.TrialData, 5, "試行回数分のトライアルデータが記録される")
	}

	assert.NotEmpty(t, report.BattleMetrics, "武器×敵の組み合わせのメトリクスが生成される")
}

func TestGenerateReport_存在しないプレイヤーはエラー(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)

	_, err := GenerateReport(master, "存在しないプレイヤー", "素手", 1, 1, 1)
	require.Error(t, err)
	assert.ErrorContains(t, err, "プレイヤーのロードに失敗")
}

func TestGenerateReport_存在しない武器はエラー(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)

	_, err := GenerateReport(master, "Ash", "存在しない武器", 1, 1, 1)
	require.Error(t, err)
	assert.ErrorContains(t, err, "武器のロードに失敗")
}

func TestGenerateBattleMetrics_存在しないプレイヤーはnilを返す(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)

	metrics := generateBattleMetrics(master, "存在しないプレイヤー", 1)
	assert.Nil(t, metrics)
}

func TestGenerateBattleMetrics_武器と敵の組み合わせでDPSを算出する(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)

	metrics := generateBattleMetrics(master, "Ash", 42)
	require.NotEmpty(t, metrics)
	for _, m := range metrics {
		assert.Equal(t, "Ash", m.Player)
		assert.GreaterOrEqual(t, m.DPS, 0.0)
	}
}

func TestReport_MarshalJSON_omitemptyでnilフィールドを省略する(t *testing.T) {
	t.Parallel()

	r := &Report{
		Mode:   "simple",
		Player: &PlayerInfo{Name: "Ash", HP: 80},
	}

	data, err := json.Marshal(r)
	require.NoError(t, err)
	assert.NotContains(t, string(data), `"weapon"`, "Weaponがnilならomitemptyで省略される")
	assert.NotContains(t, string(data), `"enemyTables"`, "EnemyTablesがnilならomitemptyで省略される")

	var got Report
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, r, &got)
}
