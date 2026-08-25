package components

// TemperatureTrend は体温の1ターンあたりの変化を表す一時状態。HUD のトレンド表示が読む。
// Delta は符号付きで、正なら温まる、負なら冷える、0付近なら一定。値の大きさが変化の速さ。
// 毎ターン再計算されるので保存しない。
type TemperatureTrend struct {
	Delta float64
}
