package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStageConstructors はコンストラクタが Kind ごとに正しいフィールドだけを埋め、
// いずれも Validate を通ることを確認する。生成経路を1本に絞る狙いの担保。
func TestStageConstructors(t *testing.T) {
	t.Parallel()

	assert.Equal(t, StageKey{Name: OverworldStageName}, NewOverworldStage())
	assert.Equal(t, StageKey{Depth: 3}, NewDungeonStage(3))
	assert.Equal(t, StageKey{Name: "森の奥", Depth: 2}, NewNamedDungeonStage("森の奥", 2))

	for _, k := range []StageKey{NewOverworldStage(), NewDungeonStage(3), NewNamedDungeonStage("森の奥", 2)} {
		require.NoError(t, k.Validate(), "コンストラクタ生成のキーは Validate を通る: %+v", k)
	}
}

// TestStageKeyValidate は Validate が Kind とフィールドの整合を検査することを確認する。
// ゼロ値は未設定として許容し、Kind に不相応なフィールドが埋まっていれば弾く。
func TestStageKeyValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		key     StageKey
		wantErr bool
	}{
		{"ゼロ値は未設定として許容", StageKey{}, false},
		{"オーバーワールドは深度0で有効", NewOverworldStage(), false},
		{"名前なしダンジョンは深度1以上で有効", NewDungeonStage(1), false},
		{"名前ありダンジョンも有効", NewNamedDungeonStage("森", 1), false},
		{"名前ありダンジョンで深度0は不正", StageKey{Name: "森", Depth: 0}, true},
		{"オーバーワールドで深度1以上は不正", StageKey{Name: OverworldStageName, Depth: 1}, true},
		{"名前なしダンジョンの負深度は不正", StageKey{Depth: -1}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.key.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
