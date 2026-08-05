package states

// BaseState は共通のtransition管理を持つベース構造体
type BaseState[T any] struct {
	trans *Transition[T]
}

// SetTransition は遷移を設定する
func (bs *BaseState[T]) SetTransition(trans Transition[T]) {
	bs.trans = &trans
}

// GetTransition は現在の遷移を取得する
func (bs *BaseState[T]) GetTransition() *Transition[T] {
	return bs.trans
}

// ClearTransition は遷移をクリアする
func (bs *BaseState[T]) ClearTransition() {
	bs.trans = nil
}

// OnPause は既定で何もしない。一時停止で処理が要る state だけが上書きする
func (bs *BaseState[T]) OnPause(_ T) error { return nil }

// OnResume は既定で何もしない。再開で処理が要る state だけが上書きする
func (bs *BaseState[T]) OnResume(_ T) error { return nil }

// OnStop は既定で何もしない。終了で後片付けが要る state だけが上書きする
func (bs *BaseState[T]) OnStop(_ T) error { return nil }

// ConsumeTransition は遷移を消費して返す
func (bs *BaseState[T]) ConsumeTransition() Transition[T] {
	if bs.trans != nil {
		next := *bs.trans
		bs.trans = nil
		return next
	}
	return Transition[T]{Type: TransNone}
}

// StateWithTransition はtransition管理機能を持つstateのインターフェース
type StateWithTransition[T any] interface {
	State[T]
	GetTransition() *Transition[T]
	ClearTransition()
}
