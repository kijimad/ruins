package gamelog

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.Len(t, entries, 1, "Expected 1 entry")

	fragments := entries[0].Fragments
	require.Len(t, fragments, 5, "Expected 5 fragments")

	// フラグメントの内容確認
	expected := []struct {
		text  string
		color color.RGBA
	}{
		{"開始", ColorWhite},
		{"プレイヤー", ColorGreen},
		{" が ", ColorWhite},
		{"敵", ColorYellow},
		{" を攻撃した。", ColorWhite},
	}

	for i, exp := range expected {
		assert.Equal(t, exp.text, fragments[i].Text, "Fragment %d text", i)
		assert.Equal(t, exp.color, fragments[i].Color, "Fragment %d color", i)
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
	require.Len(t, entries, 1, "Expected 1 entry")

	fragments := entries[0].Fragments
	expectedTexts := []string{"プレイヤー", " が ", "敵", " にクリティカル攻撃", "した。"}

	require.Len(t, fragments, len(expectedTexts), "Expected %d fragments", len(expectedTexts))

	for i, expectedText := range expectedTexts {
		assert.Equal(t, expectedText, fragments[i].Text, "Fragment %d", i)
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
		switch {
		case isPlayer:
			l.PlayerName("Ash")
		case isNPC:
			l.NPCName("スライム")
		default:
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
	require.Len(t, entries, 1, "Expected 1 entry")

	fragments := entries[0].Fragments
	require.Len(t, fragments, 4, "Expected 4 fragments")

	// 色の確認
	assert.Equal(t, ColorGreen, fragments[0].Color, "Expected player name to be green") // PlayerName
	assert.Equal(t, ColorYellow, fragments[2].Color, "Expected NPC name to be yellow")  // NPCName
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
			switch {
			case !hit:
				l.Append(" を攻撃したが外れた。")
			case critical:
				l.Append(" にクリティカルヒット。").Damage(damage).Append("ダメージ")
			default:
				l.Append(" を攻撃した。").Damage(damage).Append("ダメージ")
			}
		}).
		Log()

	// 結果確認
	entries := store.GetHistoryEntries()
	require.Len(t, entries, 1, "Expected 1 entry")

	fragments := entries[0].Fragments
	require.Len(t, fragments, 6, "Expected 6 fragments")

	// 期待される内容: "プレイヤー が ゴブリン を攻撃した。15ダメージ"
	expectedTexts := []string{"プレイヤー", " が ", "ゴブリン", " を攻撃した。", "15", "ダメージ"}
	expectedColors := []color.RGBA{
		ColorGreen,  // プレイヤー名
		ColorWhite,  // " が "
		ColorYellow, // NPC名
		ColorWhite,  // " を攻撃した。"
		ColorRed,    // ダメージ数値
		ColorWhite,  // "ダメージ"
	}

	for i, expectedText := range expectedTexts {
		assert.Equal(t, expectedText, fragments[i].Text, "Fragment %d text", i)
		assert.Equal(t, expectedColors[i], fragments[i].Color, "Fragment %d color", i)
	}
}
