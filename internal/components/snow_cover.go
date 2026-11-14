package components

// SnowCover は屋外の土の地面に付き、その日の平均気温が雪の閾値以下のときに代わりに描く絵を持つ。
// 絵はエンティティを書き換えずに描く瞬間に差し替えるので、季節が進めば次の描画で地面が変わる
type SnowCover struct {
	SpriteKey string // 雪のタイルの絵のキー。autotile の添字を除いた基
}
