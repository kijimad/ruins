package messagedata

import (
	"testing"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestGetDialogue(t *testing.T) {
	t.Parallel()

	t.Run("登録済みの会話データを取得", func(t *testing.T) {
		t.Parallel()

		msg := GetDialogue(testutil.InitTestWorld(t), "old_soldier_greeting", "老兵テスト")
		assert.Equal(t, "老兵テスト", msg.Speaker)
		// 複数ページの会話なので、TextSegmentLinesが存在することを確認
		assert.NotEmpty(t, msg.TextSegmentLines, "会話にテキストが含まれているべき")
		// 2ページ目が存在することを確認
		assert.Len(t, msg.nextMessages, 1, "2ページ目が存在するべき")
		// 1ページ目の2番目のテキストセグメントを確認（1番目は空文字列）
		if len(msg.TextSegmentLines) > 0 && len(msg.TextSegmentLines[0]) > 1 {
			assert.Equal(t, "\"You, ", msg.TextSegmentLines[0][1].Text)
		}
	})

	t.Run("未登録のキーはデフォルトメッセージを返す", func(t *testing.T) {
		t.Parallel()

		msg := GetDialogue(testutil.InitTestWorld(t), "存在しないNPC", "テストNPC")
		assert.Equal(t, "テストNPC", msg.Speaker)
		assert.Equal(t, "...", getMessageText(msg))
	})

	t.Run("ページの文を訳してマークアップとして読む", func(t *testing.T) {
		t.Parallel()

		msg := GetDialogue(testutil.InitTestWorld(t), "merchant_greeting", "商人")
		assert.Equal(t, [][]TextSegment{
			{{Text: ""}, {Text: "Want to make a deal?"}},
			{{Text: ""}},
			{{Text: "I've got good stuff."}},
		}, msg.TextSegmentLines)
		assert.Empty(t, msg.nextMessages, "1ページの会話は次のページを持たない")
	})
}
