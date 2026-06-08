package save

import (
	"encoding/json"
	"strings"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateSaveJSON_ValidData(t *testing.T) {
	t.Parallel()

	world := createStandardTestWorld(t)
	sm := createTestSerializationManager(t)

	jsonStr, err := sm.GenerateWorldJSON(world)
	require.NoError(t, err)

	err = ValidateSaveJSON(jsonStr)
	assert.NoError(t, err)
}

func TestValidateSaveJSON_InvalidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		json    string
		wantErr string
	}{
		{
			name:    "不正なJSON",
			json:    `{invalid`,
			wantErr: "JSONパースに失敗",
		},
		{
			name:    "versionフィールドが欠落",
			json:    `{"timestamp":"2026-01-01T00:00:00Z","world":{"entities":[]},"checksum":"abc"}`,
			wantErr: `property "version" is missing`,
		},
		{
			name:    "不正なversion値",
			json:    `{"version":"2.0.0","timestamp":"2026-01-01T00:00:00Z","world":{"entities":[]},"checksum":"abc"}`,
			wantErr: `value is not one of the allowed values`,
		},
		{
			name:    "worldフィールドが欠落",
			json:    `{"version":"1.0.0","timestamp":"2026-01-01T00:00:00Z","checksum":"abc"}`,
			wantErr: `property "world" is missing`,
		},
		{
			name:    "checksumフィールドが欠落",
			json:    `{"version":"1.0.0","timestamp":"2026-01-01T00:00:00Z","world":{"entities":[]}}`,
			wantErr: `property "checksum" is missing`,
		},
		{
			name:    "entitiesが配列でない",
			json:    `{"version":"1.0.0","timestamp":"2026-01-01T00:00:00Z","world":{"entities":"invalid"},"checksum":"abc"}`,
			wantErr: `value must be an array`,
		},
		{
			name:    "stable_idのindexが文字列",
			json:    `{"version":"1.0.0","timestamp":"2026-01-01T00:00:00Z","world":{"entities":[{"stable_id":{"index":"bad","generation":0},"components":{}}]},"checksum":"abc"}`,
			wantErr: `value must be an integer`,
		},
		{
			name:    "timestampが不正な形式",
			json:    `{"version":"1.0.0","timestamp":"not-a-date","world":{"entities":[]},"checksum":"abc"}`,
			wantErr: `string doesn't match the format "date-time"`,
		},
		{
			name:    "不正なenum値",
			json:    `{"version":"1.0.0","timestamp":"2026-01-01T00:00:00Z","world":{"entities":[{"stable_id":{"index":0,"generation":0},"components":{"Consumable":{"UsableScene":"INVALID","TargetType":{"TargetGroup":"ENEMY","TargetNum":"SINGLE"}}}}]},"checksum":"abc"}`,
			wantErr: `value is not one of the allowed values ["BATTLE","FIELD","ANY"]`,
		},
		{
			name:    "コンポーネントの必須フィールド欠落",
			json:    `{"version":"1.0.0","timestamp":"2026-01-01T00:00:00Z","world":{"entities":[{"stable_id":{"index":0,"generation":0},"components":{"Pools":{"HP":{"Max":100}}}}]},"checksum":"abc"}`,
			wantErr: `property "Current" is missing`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateSaveJSON(tt.json)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// TestValidateSaveJSON_GeneratedData は実際にシリアライズしたデータのバリデーションを検証する
func TestValidateSaveJSON_GeneratedData(t *testing.T) {
	t.Parallel()

	t.Run("プレイヤーのみ", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.Manager.NewEntity()
		entity.AddComponent(world.Components.Player, &gc.Player{})
		entity.AddComponent(world.Components.Name, &gc.Name{Name: "テスト"})
		entity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 1, Y: 2})

		sm := createTestSerializationManager(t)
		jsonStr, err := sm.GenerateWorldJSON(world)
		require.NoError(t, err)

		err = ValidateSaveJSON(jsonStr)
		assert.NoError(t, err)
	})

	t.Run("装備アイテム付きプレイヤー", func(t *testing.T) {
		t.Parallel()
		world := createStandardTestWorld(t)

		sm := createTestSerializationManager(t)
		jsonStr, err := sm.GenerateWorldJSON(world)
		require.NoError(t, err)

		err = ValidateSaveJSON(jsonStr)
		assert.NoError(t, err)
	})

	t.Run("改ざんしたJSONはバリデーションで弾かれる", func(t *testing.T) {
		t.Parallel()
		world := createStandardTestWorld(t)

		sm := createTestSerializationManager(t)
		jsonStr, err := sm.GenerateWorldJSON(world)
		require.NoError(t, err)

		// versionを改ざん
		tampered := strings.Replace(jsonStr, `"1.0.0"`, `"9.9.9"`, 1)

		err = ValidateSaveJSON(tampered)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "value is not one of the allowed values")
	})
}

// TestValidateSaveJSON_ComponentEnumValues はコンポーネント内のenum値を検証する
func TestValidateSaveJSON_ComponentEnumValues(t *testing.T) {
	t.Parallel()

	// 正常なセーブデータのテンプレート。コンポーネントだけ差し替える
	makeJSON := func(components map[string]any) string {
		data := map[string]any{
			"version":   "1.0.0",
			"timestamp": "2026-01-01T00:00:00Z",
			"checksum":  "dummy",
			"world": map[string]any{
				"entities": []any{
					map[string]any{
						"stable_id":  map[string]any{"index": 0, "generation": 0},
						"components": components,
					},
				},
			},
		}
		b, _ := json.Marshal(data)
		return string(b)
	}

	tests := []struct {
		name       string
		components map[string]any
		wantErr    string
	}{
		{
			name: "不正なTargetGroupType",
			components: map[string]any{
				"Consumable": map[string]any{
					"UsableScene": "BATTLE",
					"TargetType":  map[string]any{"TargetGroup": "INVALID", "TargetNum": "SINGLE"},
				},
			},
			wantErr: `value is not one of the allowed values ["ENEMY","ALLY","WEAPON","NONE"]`,
		},
		{
			name: "不正なTargetNumType",
			components: map[string]any{
				"Consumable": map[string]any{
					"UsableScene": "FIELD",
					"TargetType":  map[string]any{"TargetGroup": "ALLY", "TargetNum": "TRIPLE"},
				},
			},
			wantErr: `value is not one of the allowed values ["SINGLE","ALL"]`,
		},
		{
			name: "不正なElementType",
			components: map[string]any{
				"Melee": map[string]any{
					"Accuracy": 80, "Damage": 10, "AttackCount": 1,
					"Element":        "INVALID_ELEMENT",
					"AttackCategory": map[string]any{"Type": "SWORD", "Range": "MELEE", "Label": "剣"},
					"Cost":           5,
					"TargetType":     map[string]any{"TargetGroup": "ENEMY", "TargetNum": "SINGLE"},
				},
			},
			wantErr: `value is not one of the allowed values ["NONE","FIRE","THUNDER","CHILL","PHOTON"]`,
		},
		{
			name: "不正なAttackRangeType",
			components: map[string]any{
				"Melee": map[string]any{
					"Accuracy": 80, "Damage": 10, "AttackCount": 1,
					"Element":        "NONE",
					"AttackCategory": map[string]any{"Type": "SWORD", "Range": "INVALID", "Label": "剣"},
					"Cost":           5,
					"TargetType":     map[string]any{"TargetGroup": "ENEMY", "TargetNum": "SINGLE"},
				},
			},
			wantErr: `value is not one of the allowed values ["MELEE","RANGED"]`,
		},
		{
			name: "不正なEquipmentCategoryType",
			components: map[string]any{
				"Wearable": map[string]any{
					"Defense":           10,
					"EquipmentCategory": "INVALID",
					"EquipBonus":        map[string]any{"Vitality": 0, "Strength": 0, "Sensation": 0, "Dexterity": 0, "Agility": 0},
					"InsulationCold":    0,
					"InsulationHeat":    0,
				},
			},
			wantErr: `value is not one of the allowed values ["HEAD","TORSO","ARMS","HANDS","LEGS","FEET","JEWELRY"]`,
		},
		{
			name: "不正なHealingAmountType",
			components: map[string]any{
				"ProvidesHealing": map[string]any{
					"amount": map[string]any{"type": "invalid_type"},
				},
			},
			wantErr: `value is not one of the allowed values ["ratio","numeral"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			jsonStr := makeJSON(tt.components)
			err := ValidateSaveJSON(jsonStr)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
