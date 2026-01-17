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
