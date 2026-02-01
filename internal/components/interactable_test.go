package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDoorInteraction_Config はDoorInteractionの設定が正しいことを確認
func TestDoorInteraction_Config(t *testing.T) {
	t.Parallel()

	trigger := DoorInteraction{}
	config := trigger.Config()

	assert.Equal(t, ActivationRangeAdjacent, config.ActivationRange,
		"Doorは隣接タイルで発動する")
	assert.Equal(t, ActivationWayOnCollision, config.ActivationWay,
		"Doorは衝突時に自動発動する")
}

// TestTalkInteraction_Config はTalkInteractionの設定が正しいことを確認
func TestTalkInteraction_Config(t *testing.T) {
	t.Parallel()

	trigger := TalkInteraction{}
	config := trigger.Config()

	assert.Equal(t, ActivationRangeAdjacent, config.ActivationRange,
		"Talkは隣接タイルで発動する")
	assert.Equal(t, ActivationWayOnCollision, config.ActivationWay,
		"Talkは衝突時に自動発動する")
}

// TestItemInteraction_Config はItemInteractionの設定が正しいことを確認
func TestItemInteraction_Config(t *testing.T) {
	t.Parallel()

	trigger := ItemInteraction{}
	config := trigger.Config()

	assert.Equal(t, ActivationRangeSameTile, config.ActivationRange,
		"Itemは直上タイルで発動する")
	assert.Equal(t, ActivationWayManual, config.ActivationWay,
		"Itemは手動発動する")
}

// TestMeleeInteraction_Config はMeleeInteractionの設定が正しいことを確認
func TestMeleeInteraction_Config(t *testing.T) {
	t.Parallel()

	trigger := MeleeInteraction{}
	config := trigger.Config()

	assert.Equal(t, ActivationRangeAdjacent, config.ActivationRange,
		"Meleeは隣接タイルで発動する")
	assert.Equal(t, ActivationWayOnCollision, config.ActivationWay,
		"Meleeは衝突時に自動発動する")
}

// TestActivationRange_Valid は有効なActivationRangeの検証
func TestActivationRange_Valid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		activationRange ActivationRange
		expectValid     bool
	}{
		{
			name:            "SameTile は有効",
			activationRange: ActivationRangeSameTile,
			expectValid:     true,
		},
		{
			name:            "Adjacent は有効",
			activationRange: ActivationRangeAdjacent,
			expectValid:     true,
		},
		{
			name:            "空文字列は無効",
			activationRange: ActivationRange(""),
			expectValid:     false,
		},
		{
			name:            "未定義の値は無効",
			activationRange: ActivationRange("INVALID_RANGE"),
			expectValid:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.activationRange.Valid()
			if tt.expectValid {
				assert.NoError(t, err, "Valid()はエラーを返さないべき")
			} else {
				assert.Error(t, err, "Valid()はエラーを返すべき")
			}
		})
	}
}

// TestActivationWay_Valid は有効なActivationWayの検証
func TestActivationWay_Valid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		activationWay ActivationWay
		expectValid   bool
	}{
		{
			name:          "Auto は有効",
			activationWay: ActivationWayAuto,
			expectValid:   true,
		},
		{
			name:          "Manual は有効",
			activationWay: ActivationWayManual,
			expectValid:   true,
		},
		{
			name:          "OnCollision は有効",
			activationWay: ActivationWayOnCollision,
			expectValid:   true,
		},
		{
			name:          "空文字列は無効",
			activationWay: ActivationWay(""),
			expectValid:   false,
		},
		{
			name:          "未定義の値は無効",
			activationWay: ActivationWay("INVALID_MODE"),
			expectValid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.activationWay.Valid()
			if tt.expectValid {
				assert.NoError(t, err, "Valid()はエラーを返さないべき")
			} else {
				assert.Error(t, err, "Valid()はエラーを返すべき")
			}
		})
	}
}

// TestTriggerInterfaceImplementation は全てのトリガーがInteractionDataインターフェースを実装していることを確認
func TestTriggerInterfaceImplementation(t *testing.T) {
	t.Parallel()

	// 全てのトリガータイプがInteractionDataインターフェースを実装していることを確認
	var _ InteractionData = DoorInteraction{}
	var _ InteractionData = TalkInteraction{}
	var _ InteractionData = ItemInteraction{}
	var _ InteractionData = MeleeInteraction{}
}

// TestTriggerConfigConsistency は全トリガーの設定が一貫していることを確認
func TestTriggerConfigConsistency(t *testing.T) {
	t.Parallel()

	// 全トリガータイプ
	triggers := []struct {
		name    string
		trigger InteractionData
	}{
		{"Door", DoorInteraction{}},
		{"Talk", TalkInteraction{}},
		{"Item", ItemInteraction{}},
		{"Melee", MeleeInteraction{}},
	}

	for _, tt := range triggers {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			config := tt.trigger.Config()

			// ActivationRangeが有効な値であることを確認
			err := config.ActivationRange.Valid()
			require.NoError(t, err, "%s のActivationRangeは有効でなければならない", tt.name)

			// ActivationWayが有効な値であることを確認
			err = config.ActivationWay.Valid()
			require.NoError(t, err, "%s のActivationWayは有効でなければならない", tt.name)
		})
	}
}

// TestTriggerDesignConstraints は設計上の制約をテスト（仕様確認）
func TestTriggerDesignConstraints(t *testing.T) {
	t.Parallel()

	t.Run("SameTileトリガーはManual方式", func(t *testing.T) {
		t.Parallel()
		// 仕様: 直上タイルトリガー（Item）は手動発動
		sameTileTriggers := []InteractionData{
			ItemInteraction{},
		}

		for _, trigger := range sameTileTriggers {
			config := trigger.Config()
			assert.Equal(t, ActivationRangeSameTile, config.ActivationRange)
			assert.Equal(t, ActivationWayManual, config.ActivationWay,
				"直上タイルトリガーは手動発動である")
		}
	})

	t.Run("AdjacentトリガーはOnCollision方式", func(t *testing.T) {
		t.Parallel()
		// 仕様: 隣接タイルトリガー（Door, Talk）は衝突時自動発動
		adjacentTriggers := []InteractionData{
			DoorInteraction{},
			TalkInteraction{},
		}

		for _, trigger := range adjacentTriggers {
			config := trigger.Config()
			assert.Equal(t, ActivationRangeAdjacent, config.ActivationRange)
			assert.Equal(t, ActivationWayOnCollision, config.ActivationWay,
				"隣接タイルトリガーは衝突時自動発動である")
		}
	})

	t.Run("MeleeトリガーはAdjacent+OnCollision方式", func(t *testing.T) {
		t.Parallel()
		// 仕様: 近接攻撃トリガーは隣接タイルで衝突時自動発動
		trigger := MeleeInteraction{}
		config := trigger.Config()
		assert.Equal(t, ActivationRangeAdjacent, config.ActivationRange,
			"Meleeは隣接タイルで発動する")
		assert.Equal(t, ActivationWayOnCollision, config.ActivationWay,
			"Meleeは衝突時自動発動する")
	})
}
