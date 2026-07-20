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

	assert.Equal(t, StageKey{Kind: StageKindOverworld}, NewOverworldStage())
	assert.Equal(t, StageKey{Kind: StageKindDungeon, Depth: 3}, NewDungeonStage(3))
	assert.Equal(t, StageKey{Kind: StageKindRuin, Ruin: "遺跡", Depth: 2}, NewRuinStage("遺跡", 2))

	for _, k := range []StageKey{NewOverworldStage(), NewDungeonStage(3), NewRuinStage("遺跡", 2)} {
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
		{"オーバーワールドは深度も遺跡名もなし", NewOverworldStage(), false},
		{"ダンジョンは深度あり遺跡名なし", NewDungeonStage(1), false},
		{"遺跡は遺跡名あり", NewRuinStage("森", 1), false},
		{"ダンジョンに遺跡名は不正", StageKey{Kind: StageKindDungeon, Ruin: "森", Depth: 1}, true},
		{"遺跡に遺跡名なしは不正", StageKey{Kind: StageKindRuin, Depth: 1}, true},
		{"オーバーワールドに深度は不正", StageKey{Kind: StageKindOverworld, Depth: 1}, true},
		{"未知の種別は不正", StageKey{Kind: StageKind("banana")}, true},
		{"空 Kind で深度だけ埋まるのは不正", StageKey{Depth: 5}, true},
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
