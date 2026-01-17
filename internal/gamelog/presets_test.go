package gamelog

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
)

func TestPresetFunctions(t *testing.T) {
	t.Parallel()

	// ローカルテストストアを作成
	testFieldLog := NewSafeSlice(FieldLogMaxSize)

	// プリセット関数のテスト
	New(testFieldLog).
		Success("勝利しました！").
		Log()

	New(testFieldLog).
		Warning("注意が必要です").
		Log()

	New(testFieldLog).
		Error("エラーが発生しました").
		Log()

	New(testFieldLog).
		PlayerName("Hero").
		Append("が").
		Location("洞窟").
		Append("で").
		ItemName("宝箱").
		Append("を発見した。").
		Log()

	New(testFieldLog).
		Action("攻撃").
		Append("で").
		NPCName("ゴブリン").
		Append("に").
		Damage(25).
		Append("ダメージ！").
		Log()

	New(testFieldLog).
		Money("500").
		Append("G獲得した。").
		Log()

	// ログ数の確認
	if testFieldLog.Count() != 6 {
		t.Errorf("Expected 6 log entries, got %d", testFieldLog.Count())
	}

	// 色付きエントリの確認
	entries := testFieldLog.GetRecentEntries(6)
	if len(entries) != 6 {
		t.Errorf("Expected 6 colored entries, got %d", len(entries))
	}

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

		if fragment.Text != tc.expectedText {
			t.Errorf("Entry %d, Fragment %d: expected text '%s', got '%s'",
				tc.entryIndex, tc.fragmentIndex, tc.expectedText, fragment.Text)
		}

		// 色の確認（簡単なチェック）
		switch tc.expectedColor {
		case "green":
			if fragment.Color != consts.ColorGreen {
				t.Errorf("Expected green color for '%s'", tc.expectedText)
			}
		case "yellow":
			if fragment.Color != consts.ColorYellow {
				t.Errorf("Expected yellow color for '%s'", tc.expectedText)
			}
		case "red":
			if fragment.Color != consts.ColorRed {
				t.Errorf("Expected red color for '%s'", tc.expectedText)
			}
		case "orange":
			if fragment.Color != consts.ColorOrange {
				t.Errorf("Expected orange color for '%s'", tc.expectedText)
			}
		case "cyan":
			if fragment.Color != consts.ColorCyan {
				t.Errorf("Expected cyan color for '%s'", tc.expectedText)
			}
		case "purple":
			if fragment.Color != consts.ColorPurple {
				t.Errorf("Expected purple color for '%s'", tc.expectedText)
			}
		}
	}
}

func TestSystemPresets(t *testing.T) {
	t.Parallel()

	// ローカルテストストアを作成
	testFieldLog := NewSafeSlice(FieldLogMaxSize)

	// システム関連のプリセット
	New(testFieldLog).
		System("システムが初期化されました").
		Log()

	entries := testFieldLog.GetRecentEntries(1)
	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
	}

	// System は水色（シアン）
	if entries[0].Fragments[0].Color != consts.ColorCyan {
		t.Errorf("Expected cyan color for System")
	}
}
