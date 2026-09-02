package states

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/messagedata"
	"github.com/kijimaD/ruins/internal/widgets/messagewindow"
	w "github.com/kijimaD/ruins/internal/world"
)

// MessageState はメッセージを表示する専用ステート
type MessageState struct {
	es.BaseState[w.World]
	// build は world を要するメッセージの構築。翻訳は現在言語を要するので、ファクトリ時ではなく
	// OnStart で world を渡して messageData を組む。構築は必ずこれを通す。組み立て済みのデータは
	// NewMessageState がそれを返すだけの build に包む
	build           func(w.World) *messagedata.MessageData
	messageData     *messagedata.MessageData // build の結果をキャッシュする内部フィールド
	messageWindow   *messagewindow.Window
	backgroundImage *ebiten.Image
	currentBgKey    string
}

var _ es.State[w.World] = &MessageState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *MessageState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *MessageState) OnResume(_ w.World) error { return nil }

// OnStart はステートが開始される際に呼ばれる
func (st *MessageState) OnStart(world w.World) error {
	if st.build == nil {
		return fmt.Errorf("message state has no build function")
	}
	if st.messageData == nil {
		st.messageData = st.build(world)
	}
	if st.messageData.BackgroundKey != "" {
		bgImage, err := loadBackgroundImage(world, st.messageData.BackgroundKey)
		if err != nil {
			return err
		}
		st.backgroundImage = bgImage
		st.currentBgKey = st.messageData.BackgroundKey
	}

	st.messageWindow = messagewindow.NewWindow(world, st.messageData)
	return nil
}

// OnStop はステートが停止される際に呼ばれる
func (st *MessageState) OnStop(_ w.World) error { return nil }

// loadBackgroundImage はスプライトシートから背景画像を読み込んで返す。
// 無効なspriteKeyが指定された場合はエラーを返す。
func loadBackgroundImage(world w.World, spriteKey string) (*ebiten.Image, error) {
	tex, rect, ok := world.Resources.Sprites.Rect(&gc.SpriteRender{SpriteSheetName: "bg", SpriteKey: spriteKey})
	if !ok {
		return nil, fmt.Errorf("invalid BackgroundKey: %q not found in bg sprite sheet", spriteKey)
	}
	return gc.SubImage(tex.Image, rect), nil
}

// Update はゲームステートの更新処理を行う
func (st *MessageState) Update(world w.World) (es.Transition[w.World], error) {
	if st.messageWindow != nil {
		// 現在のメッセージの背景キーが変わっていたら背景を更新する
		if key := st.messageWindow.CurrentMessage().BackgroundKey; key != "" && key != st.currentBgKey {
			bgImage, err := loadBackgroundImage(world, key)
			if err != nil {
				return es.Transition[w.World]{Type: es.TransNone}, err
			}
			st.backgroundImage = bgImage
			st.currentBgKey = key
		}

		if err := st.messageWindow.Update(); err != nil {
			return es.Transition[w.World]{Type: es.TransNone}, err
		}

		if st.messageWindow.IsClosed() {
			// BaseStateで設定された遷移を優先確認
			if transition := st.ConsumeTransition(); transition.Type != es.TransNone {
				return transition, nil
			}
			// デフォルトはステートをポップ
			return es.Transition[w.World]{Type: es.TransPop}, nil
		}
		// MessageWindowがアクティブな間は何もしない
		return es.Transition[w.World]{Type: es.TransNone}, nil
	}

	return st.ConsumeTransition(), nil
}

// Draw はゲームステートの描画処理を行う
func (st *MessageState) Draw(_ w.World, screen *ebiten.Image) error {
	// 背景画像があれば最初に描画
	if st.backgroundImage != nil {
		screen.DrawImage(st.backgroundImage, nil)
	}

	if st.messageWindow != nil {
		st.messageWindow.Draw(screen)
	}
	return nil
}
