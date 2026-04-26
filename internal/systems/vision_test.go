package systems

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetermineTileDarkness(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		inVisionRange  bool
		visible        bool
		explored       bool
		hasLightSource bool
		expected       TileDarknessLevel
	}{
		{
			name:           "視界内+可視+光源あり → 光源で照らされた明るさ",
			inVisionRange:  true,
			visible:        true,
			explored:       true,
			hasLightSource: true,
			expected:       TileDarknessLit,
		},
		{
			name:           "視界内+可視+光源なし → 視界内の暗さ",
			inVisionRange:  true,
			visible:        true,
			explored:       true,
			hasLightSource: false,
			expected:       TileDarknessVisible,
		},
		{
			name:          "視界内+遮蔽+探索済み → 探索済みの暗さ",
			inVisionRange: true,
			visible:       false,
			explored:      true,
			expected:      TileDarknessExplored,
		},
		{
			name:          "視界内+遮蔽+未探索 → 完全に黒",
			inVisionRange: true,
			visible:       false,
			explored:      false,
			expected:      TileDarknessFull,
		},
		{
			name:     "視界外+探索済み → 探索済みの暗さ",
			explored: true,
			expected: TileDarknessExplored,
		},
		{
			name:     "視界外+未探索 → スキップ",
			expected: TileDarknessSkip,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := determineTileDarkness(tt.inVisionRange, tt.visible, tt.explored, tt.hasLightSource)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTileDarknessLevelOrdering(t *testing.T) {
	t.Parallel()

	// 暗さの段階が明るい順に並んでいることを保証する
	assert.Less(t, TileDarknessLit.DarknessValue(), TileDarknessVisible.DarknessValue())
	assert.Less(t, TileDarknessVisible.DarknessValue(), TileDarknessExplored.DarknessValue())
	assert.Less(t, TileDarknessExplored.DarknessValue(), TileDarknessFull.DarknessValue())
}

func TestTileDarknessExploredNotFullyBlack(t *testing.T) {
	t.Parallel()

	// 探索済みタイルの暗さが完全な黒と区別できることを保証する
	assert.Less(t, TileDarknessExplored.DarknessValue(), TileDarknessFull.DarknessValue(),
		"探索済みタイルは完全な黒より明るくなければならない")

	// ceil量子化でも完全な黒（level DarknessLevels）にならないことを保証する
	darknessLevel := int(TileDarknessExplored.DarknessValue() * float64(DarknessLevels))
	assert.Less(t, darknessLevel, DarknessLevels,
		"探索済みタイルの暗さが量子化で完全な黒(level %d)になってはいけない", DarknessLevels)
}
