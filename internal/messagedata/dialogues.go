package messagedata

import (
	"github.com/kijimaD/ruins/internal/i18n"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// GetDialogue は raw の dialogues の行 key から会話データを組み立てる。1ページを1メッセージにして順につなぐ。
// ページの文は英語の msgid で、query.T で現在言語へ引いてからマークアップとして読む。行が無ければ既定の文を返す
func GetDialogue(world w.World, key string, speakerName string) *MessageData {
	for _, d := range raw.PtrSlice(world.Resources.RawMaster.Dialogues) {
		if d.Id == key {
			return dialoguePages(world, d, speakerName)
		}
	}
	return NewDialogMessage("...", speakerName)
}

// dialoguePages は会話の行のページを1メッセージずつ組んでつなぐ
func dialoguePages(world w.World, d rawschema.Dialogue, speakerName string) *MessageData {
	pages := make([]*MessageData, 0, len(d.Pages))
	for _, p := range d.Pages {
		pages = append(pages, NewDialogMessage("", speakerName).AddMarkup(query.T(world, i18n.Msgid(p))))
	}
	return ChainMessages(pages...)
}
