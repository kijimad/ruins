package gamelog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPresetFunctions(t *testing.T) {
	t.Parallel()

	// ローカルテストストアを作成
	testGameLog := NewSafeSlice(GameLogMaxSize)

	// プリセット関数のテスト
	New(testGameLog).
		Success("勝利しました！").
		Log()

	New(testGameLog).
		Warning("注意が必要です").
		Log()

	New(testGameLog).
		Error("エラーが発生しました").
		Log()

	New(testGameLog).
		PlayerName("Hero").
		Append("が").
		Location("洞窟").
		Append("で").
		ItemName("宝箱").
		Append("を発見した。").
		Log()

	New(testGameLog).
		Action("攻撃").
		Append("で").
		NPCName("ゴブリン").
		Append("に").
		Damage(25).
		Append("ダメージ！").
		Log()

	New(testGameLog).
		Money("500").
		Append("G獲得した。").
		Log()

	// ログ数の確認
	assert.Equal(t, 6, testGameLog.Count(), "Expected 6 log entries")

	// 色付きエントリの確認
	entries := testGameLog.GetRecentEntries(6)
	require.Len(t, entries, 6, "Expected 6 colored entries")

	// 各エントリの色の確認
	testCases := []struct {
		entryIndex    int
		fragmentIndex int
		expectedColor string
		expectedText  string
	}{
		{0, 0, "green", "勝利しました！"},
		{1, 0, "yellow", "注意が必要です"},
		{2, 0, "red", "エラーが発生しました"},
		{3, 0, "green", "Hero"},  // PlayerName
		{3, 2, "orange", "洞窟"},   // Location
		{3, 4, "cyan", "宝箱"},     // ItemName
		{4, 0, "purple", "攻撃"},   // Action
		{4, 2, "yellow", "ゴブリン"}, // NPCName
		{4, 4, "red", "25"},      // Damage
		{5, 0, "yellow", "500"},  // Money
	}

	for _, tc := range testCases {
		if tc.entryIndex >= len(entries) {
			continue
		}
		entry := entries[tc.entryIndex]
		if tc.fragmentIndex >= len(entry.Fragments) {
			continue
		}
		fragment := entry.Fragments[tc.fragmentIndex]

		assert.Equal(t, tc.expectedText, fragment.Text,
			"Entry %d, Fragment %d text", tc.entryIndex, tc.fragmentIndex)

		// 色の確認（簡単なチェック）
		switch tc.expectedColor {
		case "green":
			assert.Equal(t, ColorGreen, fragment.Color, "Expected green color for '%s'", tc.expectedText)
		case "yellow":
			assert.Equal(t, ColorYellow, fragment.Color, "Expected yellow color for '%s'", tc.expectedText)
		case "red":
			assert.Equal(t, ColorRed, fragment.Color, "Expected red color for '%s'", tc.expectedText)
		case "orange":
			assert.Equal(t, ColorOrange, fragment.Color, "Expected orange color for '%s'", tc.expectedText)
		case "cyan":
			assert.Equal(t, ColorCyan, fragment.Color, "Expected cyan color for '%s'", tc.expectedText)
		case "purple":
			assert.Equal(t, ColorPurple, fragment.Color, "Expected purple color for '%s'", tc.expectedText)
		}
	}
}

func TestSystemPresets(t *testing.T) {
	t.Parallel()

	// ローカルテストストアを作成
	testGameLog := NewSafeSlice(GameLogMaxSize)

	// システム関連のプリセット
	New(testGameLog).
		System("システムが初期化されました").
		Log()

	entries := testGameLog.GetRecentEntries(1)
	require.Len(t, entries, 1, "Expected 1 entry")

	// System は水色（シアン）
	assert.Equal(t, ColorCyan, entries[0].Fragments[0].Color, "Expected cyan color for System")
}
