package messagedata

import (
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// DialogueTable は会話データのテーブル。表示文字列は query.T で現在言語へ引く
var DialogueTable = map[string]func(world w.World, speakerName string) *MessageData{
	"tavern_keeper_greeting": func(world w.World, speakerName string) *MessageData {
		return NewDialogMessage("", speakerName).
			AddText(query.T(world, "We've got some capable folks here.\n\nWant to hire a squad member?"))
	},
	"old_soldier_greeting": func(world w.World, speakerName string) *MessageData {
		// 1ページ目
		msg1 := NewDialogMessage("", speakerName).
			AddText(query.T(world, "\"You, ")).
			AddKeyword(query.T(world, "ruins")).
			AddText(query.T(world, "of the ")).
			AddKeyword(query.T(world, "delver")).
			AddText(query.T(world, "..., right?\n\nEveryone strangely young who comes to this town from outside is like that.\nReckless and self-destructive,...\n\nthey carry some hopeless burden.\""))

		// 2ページ目
		msg2 := NewDialogMessage("", speakerName).
			AddText(query.T(world, "\"You... I see, so your mother was ")).
			AddKeyword(query.T(world, "Hollow")).
			AddText(query.T(world, "..., huh.\n\nWhat an irredeemable world.\""))

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
