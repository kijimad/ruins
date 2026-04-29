package states

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/messagedata"
	"github.com/kijimaD/ruins/internal/widgets/messagewindow"
	w "github.com/kijimaD/ruins/internal/world"
)

// MessageState はメッセージを表示する専用ステート
type MessageState struct {
	es.BaseState[w.World]
	messageData     *messagedata.MessageData
	messageWindow   *messagewindow.Window
	backgroundImage *ebiten.Image
}

func (st MessageState) String() string {
	return "Message"
}

var _ es.State[w.World] = &MessageState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *MessageState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *MessageState) OnResume(_ w.World) error { return nil }

// OnStart はステートが開始される際に呼ばれる
func (st *MessageState) OnStart(world w.World) error {
	if st.messageData.BackgroundKey != "" {
		st.backgroundImage = loadBackgroundImage(world, st.messageData.BackgroundKey)
	}

	// メッセージデータからキュー対応メッセージウィンドウを構築
	st.messageWindow = messagewindow.NewBuilder(world).
		WithOnMessageChange(func(msg *messagedata.MessageData) {
			if msg.BackgroundKey != "" {
				st.backgroundImage = loadBackgroundImage(world, msg.BackgroundKey)
			}
		}).
		Build(st.messageData)
	return nil
}

// OnStop はステートが停止される際に呼ばれる
func (st *MessageState) OnStop(_ w.World) error { return nil }

// loadBackgroundImage はスプライトシートから背景画像を読み込んで返す
func loadBackgroundImage(world w.World, spriteKey string) *ebiten.Image {
	sheet := (*world.Resources.SpriteSheets)["bg"]
	sprite := sheet.Sprites[spriteKey]
	rect := image.Rect(
		sprite.X,
		sprite.Y,
		sprite.X+sprite.Width,
		sprite.Y+sprite.Height,
	)
	return sheet.Texture.Image.SubImage(rect).(*ebiten.Image)
}

// Update はゲームステートの更新処理を行う
func (st *MessageState) Update(_ w.World) (es.Transition[w.World], error) {
	if st.messageWindow != nil {
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
