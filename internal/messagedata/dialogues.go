package messagedata

import (
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// DialogueTable は会話データのテーブル。表示文字列は query.T で現在言語へ引く
var DialogueTable = map[string]func(world w.World, speakerName string) *MessageData{
	"old_soldier_greeting": func(world w.World, speakerName string) *MessageData {
		// 会話は1ページ=1文字列。強調語は <keyword> で囲む。断片を連結せず語順ごと訳せる
		msg1 := NewDialogMessage("", speakerName).
			AddMarkup(query.T(world, "\"You, <keyword>ruins</keyword> of the <keyword>delver</keyword>..., right?\n\nEveryone strangely young who comes to this town from outside is like that.\nReckless and self-destructive,...\n\nthey carry some hopeless burden.\""))

		msg2 := NewDialogMessage("", speakerName).
			AddMarkup(query.T(world, "\"You... I see, so your mother was <keyword>Hollow</keyword>..., huh.\n\nWhat an irredeemable world.\""))

		return ChainMessages(msg1, msg2)
	},
}

// GetDialogue は指定されたキーに対応する会話データを取得する
func GetDialogue(world w.World, key string, speakerName string) *MessageData {
	if dialogueFunc, ok := DialogueTable[key]; ok {
		return dialogueFunc(world, speakerName)
	}
	// デフォルトメッセージ
	return NewDialogMessage("...", speakerName)
}
