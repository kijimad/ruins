package balance

import (
	"encoding/json"
	"testing"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateReport_プレイヤーと武器と敵テーブルの結果を含むレポートを生成する(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)

	report, err := GenerateReport(master, "ash", "bare_hands", 3, 5, 42)
	require.NoError(t, err)

	assert.Equal(t, "simple", report.Mode)

	require.NotNil(t, report.Player)
	assert.Equal(t, "ash", report.Player.Name)
	assert.Equal(t, 80, report.Player.Hp)

	require.NotNil(t, report.Weapon)
	assert.Equal(t, "bare_hands", report.Weapon.Name)

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

	_, err := GenerateReport(master, "存在しないプレイヤー", "bare_hands", 1, 1, 1)
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to load player")
}

func TestGenerateReport_存在しない武器はエラー(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)

	_, err := GenerateReport(master, "ash", "存在しない武器", 1, 1, 1)
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to load weapon")
}

func TestGenerateBattleMetrics_存在しないプレイヤーは空を返す(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)

	metrics := generateBattleMetrics(master, "存在しないプレイヤー", 1)
	assert.Empty(t, metrics)
}

func TestGenerateBattleMetrics_武器と敵の組み合わせでDPSを算出する(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)

	metrics := generateBattleMetrics(master, "ash", 42)
	require.NotEmpty(t, metrics)
	for _, m := range metrics {
		assert.Equal(t, "ash", m.Player)
		assert.GreaterOrEqual(t, m.Dps, 0.0)
	}
}

func TestGenerateReport_OpenAPIスキーマに適合する(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)

	report, err := GenerateReport(master, "ash", "bare_hands", 3, 5, 42)
	require.NoError(t, err)

	// 生成したレポートを tsp 由来の Balance.Report スキーマで検証する。raw の ValidateRaws と同じ手法で、
	// 生成型と埋め込みスペックの同期ズレや、balance.tsp に足した制約への違反を回帰で捕まえる。
	spec, err := oapi.GetSpec()
	require.NoError(t, err)
	schemaRef, ok := spec.Components.Schemas["Balance.Report"]
	require.True(t, ok, "Balance.Report コンポーネントがスキーマにある")

	jsonBytes, err := json.Marshal(report)
	require.NoError(t, err)
	var jsonData any
	require.NoError(t, json.Unmarshal(jsonBytes, &jsonData))

	require.NoError(t, schemaRef.Value.VisitJSON(jsonData), "生成した balance レポートが OpenAPI スキーマに適合する")
}

func TestGenerateReport_スキーマ違反はバリデーションで検出する(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)

	report, err := GenerateReport(master, "ash", "bare_hands", 3, 5, 42)
	require.NoError(t, err)

	spec, err := oapi.GetSpec()
	require.NoError(t, err)
	schemaRef, ok := spec.Components.Schemas["Balance.Report"]
	require.True(t, ok)
	schema := schemaRef.Value

	// 適合するレポートを一度だけ JSON 化し、ケースごとに map へ展開し直して1箇所を壊す。
	jsonBytes, err := json.Marshal(report)
	require.NoError(t, err)
	freshData := func(t *testing.T) map[string]any {
		t.Helper()
		var data map[string]any
		require.NoError(t, json.Unmarshal(jsonBytes, &data))
		return data
	}

	tests := []struct {
		name    string
		corrupt func(t *testing.T, data map[string]any)
	}{
		{
			name: "Rateの範囲を超えるdeathRate",
			corrupt: func(t *testing.T, data map[string]any) {
				t.Helper()
				tables, ok := data["enemyTables"].([]any)
				require.True(t, ok)
				require.NotEmpty(t, tables)
				first, ok := tables[0].(map[string]any)
				require.True(t, ok)
				first["deathRate"] = 1.5 // Rate は 0.0-1.0
			},
		},
		{
			name: "負のmedianDamage",
			corrupt: func(t *testing.T, data map[string]any) {
				t.Helper()
				tables, ok := data["enemyTables"].([]any)
				require.True(t, ok)
				require.NotEmpty(t, tables)
				first, ok := tables[0].(map[string]any)
				require.True(t, ok)
				depths, ok := first["depths"].([]any)
				require.True(t, ok)
				require.NotEmpty(t, depths)
				d0, ok := depths[0].(map[string]any)
				require.True(t, ok)
				d0["medianDamage"] = -1 // Damage は minValue 0
			},
		},
		{
			name: "必須のmodeが欠落",
			corrupt: func(_ *testing.T, data map[string]any) {
				delete(data, "mode")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data := freshData(t)
			tt.corrupt(t, data)
			require.Error(t, schema.VisitJSON(data), "スキーマ違反は VisitJSON で検出される")
		})
	}
}

func TestBalanceReport_nilのweaponはJSONで省略される(t *testing.T) {
	t.Parallel()

	r := &oapi.BalanceReport{
		Mode:   "simple",
		Player: &oapi.BalancePlayerInfo{Name: "ash", Hp: 80},
	}

	data, err := json.Marshal(r)
	require.NoError(t, err)
	assert.NotContains(t, string(data), `"weapon"`, "Weaponがnilならomitemptyで省略される")

	var got oapi.BalanceReport
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, r, &got)
}
