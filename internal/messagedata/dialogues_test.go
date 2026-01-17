package messagedata

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDialogue(t *testing.T) {
	t.Parallel()

	t.Run("登録済みの会話データを取得", func(t *testing.T) {
		t.Parallel()

		msg := GetDialogue("old_soldier_greeting", "老兵テスト")
		assert.Equal(t, "老兵テスト", msg.Speaker)
		// 複数ページの会話なので、TextSegmentLinesが存在することを確認
		assert.NotEmpty(t, msg.TextSegmentLines, "会話にテキストが含まれているべき")
		// 2ページ目が存在することを確認
		assert.Len(t, msg.NextMessages, 1, "2ページ目が存在するべき")
		// 1ページ目の2番目のテキストセグメントを確認（1番目は空文字列）
		if len(msg.TextSegmentLines) > 0 && len(msg.TextSegmentLines[0]) > 1 {
			assert.Equal(t, "「あんた、", msg.TextSegmentLines[0][1].Text)
		}
	})

	t.Run("未登録のキーはデフォルトメッセージを返す", func(t *testing.T) {
		t.Parallel()

		msg := GetDialogue("存在しないNPC", "テストNPC")
		assert.Equal(t, "テストNPC", msg.Speaker)
		assert.Equal(t, "...", getMessageText(msg))
	})

	t.Run("DialogueTableから直接会話を生成", func(t *testing.T) {
		t.Parallel()

		dialogueFunc, ok := DialogueTable["old_soldier_greeting"]
		assert.True(t, ok, "old_soldier_greetingの会話データが存在するべき")

		msg := dialogueFunc("老兵A")
		assert.Equal(t, "老兵A", msg.Speaker)
		// 複数ページの会話なので、TextSegmentLinesが存在することを確認
		assert.NotEmpty(t, msg.TextSegmentLines, "会話にテキストが含まれているべき")
		// 2ページ目が存在することを確認
		assert.Len(t, msg.NextMessages, 1, "2ページ目が存在するべき")
	})
}
