package gamelog

import (
	"image/color"
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
)

func TestLoggerBuildMethod(t *testing.T) {
	t.Parallel()
	store := NewSafeSlice(100)

	// Build メソッドを使ったログ構築
	logger := New(store)
	logger.Append("開始").
		Build(func(l *Logger) {
			l.PlayerName("プレイヤー")
			l.Append(" が ")
			l.NPCName("敵")
		}).
		Append(" を攻撃した。").
		Log()

	// 結果確認
	entries := store.GetHistoryEntries()
	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
		return
	}

	fragments := entries[0].Fragments
	if len(fragments) != 5 {
		t.Errorf("Expected 5 fragments, got %d", len(fragments))
		return
	}

	// フラグメントの内容確認
	expected := []struct {
		text  string
		color color.RGBA
	}{
		{"開始", consts.ColorWhite},
		{"プレイヤー", consts.ColorGreen},
		{" が ", consts.ColorWhite},
		{"敵", consts.ColorYellow},
		{" を攻撃した。", consts.ColorWhite},
	}

	for i, exp := range expected {
		if fragments[i].Text != exp.text {
			t.Errorf("Fragment %d: expected text '%s', got '%s'", i, exp.text, fragments[i].Text)
		}
		if fragments[i].Color != exp.color {
			t.Errorf("Fragment %d: expected color %v, got %v", i, exp.color, fragments[i].Color)
		}
	}
}

func TestLoggerBuildWithCondition(t *testing.T) {
	t.Parallel()
	store := NewSafeSlice(100)

	// Build メソッドを使った条件付きログ構築
	critical := true

	logger := New(store)
	logger.PlayerName("プレイヤー").
		Append(" が ").
		NPCName("敵").
		Build(func(l *Logger) {
			if critical {
				l.Append(" にクリティカル攻撃")
			} else {
				l.Append(" に通常攻撃")
			}
		}).
		Append("した。").
		Log()

	// 結果確認
	entries := store.GetHistoryEntries()
	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
		return
	}

	fragments := entries[0].Fragments
	expectedTexts := []string{"プレイヤー", " が ", "敵", " にクリティカル攻撃", "した。"}

	if len(fragments) != len(expectedTexts) {
		t.Errorf("Expected %d fragments, got %d", len(expectedTexts), len(fragments))
		return
	}

	for i, expectedText := range expectedTexts {
		if fragments[i].Text != expectedText {
			t.Errorf("Fragment %d: expected '%s', got '%s'", i, expectedText, fragments[i].Text)
		}
	}
}

func TestLoggerBuildWithEntityLogic(t *testing.T) {
	t.Parallel()
	store := NewSafeSlice(100)

	// Build メソッドでエンティティロジックを実装
	isPlayer := true
	isNPC := false

	logger := New(store)
	logger.Build(func(l *Logger) {
		if isPlayer {
			l.PlayerName("セレスティン")
		} else if isNPC {
			l.NPCName("スライム")
		} else {
			l.Append("Unknown")
		}
	}).
		Append(" が ").
		Build(func(l *Logger) {
			// 対戦相手は異なる種類にする
			l.NPCName("スライム")
		}).
		Append(" を攻撃した。").
		Log()

	// 結果確認
	entries := store.GetHistoryEntries()
	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
		return
	}

	fragments := entries[0].Fragments
	if len(fragments) != 4 {
		t.Errorf("Expected 4 fragments, got %d", len(fragments))
		return
	}

	// 色の確認
	if fragments[0].Color != consts.ColorGreen { // PlayerName
		t.Errorf("Expected player name to be green, got %v", fragments[0].Color)
	}
	if fragments[2].Color != consts.ColorYellow { // NPCName
		t.Errorf("Expected NPC name to be yellow, got %v", fragments[2].Color)
	}
}

func TestComplexMethodChain(t *testing.T) {
	t.Parallel()
	store := NewSafeSlice(100)

	// 複合的なメソッドチェーンのテスト
	hit := true
	critical := false
	damage := 15

	logger := New(store)
	logger.Build(func(l *Logger) {
		l.PlayerName("プレイヤー")
	}).
		Append(" が ").
		Build(func(l *Logger) {
			l.NPCName("ゴブリン")
		}).
		Build(func(l *Logger) {
			if !hit {
				l.Append(" を攻撃したが外れた。")
			} else if critical {
				l.Append(" にクリティカルヒット。").Damage(damage).Append("ダメージ")
			} else {
				l.Append(" を攻撃した。").Damage(damage).Append("ダメージ")
			}
		}).
		Log()

	// 結果確認
	entries := store.GetHistoryEntries()
	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
		return
	}

	fragments := entries[0].Fragments
	if len(fragments) != 6 {
		t.Errorf("Expected 6 fragments, got %d", len(fragments))
		return
	}

	// 期待される内容: "プレイヤー が ゴブリン を攻撃した。15ダメージ"
	expectedTexts := []string{"プレイヤー", " が ", "ゴブリン", " を攻撃した。", "15", "ダメージ"}
	expectedColors := []color.RGBA{
		consts.ColorGreen,  // プレイヤー名
		consts.ColorWhite,  // " が "
		consts.ColorYellow, // NPC名
		consts.ColorWhite,  // " を攻撃した。"
		consts.ColorRed,    // ダメージ数値
		consts.ColorWhite,  // "ダメージ"
	}

	for i, expectedText := range expectedTexts {
		if fragments[i].Text != expectedText {
			t.Errorf("Fragment %d: expected text '%s', got '%s'", i, expectedText, fragments[i].Text)
		}
		if fragments[i].Color != expectedColors[i] {
			t.Errorf("Fragment %d: expected color %v, got %v", i, expectedColors[i], fragments[i].Color)
		}
	}
}
