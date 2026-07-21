package components

// StageMeta はステージごとのフィールド状態を保持する。
// StageBound で各ステージに束縛され、他のフィールドエンティティと同様に共存・退避・serde される。
// 現ステージのメタは Dungeon.CurrentStage で引く。
//
// オーバーワールドもダンジョン階も同じ「ステージ」で、種別は持たない。違いは保有データだけにする。
// 帯・前線データは SeamlessBand コンポーネントとしてオーバーワールドのメタだけが持ち、
// その有無が「オーバーワールドか」の判別を兼ねる。ダンジョン階のメタは持たない。
type StageMeta struct {
	// Level は現ステージのフィールド寸法。ステージごとに保持するため、往復してもステージ固有の
	// 寸法が resume で自然に戻る。
	Level Level
	// ExploredTiles は探索済みタイルのマップ。ステージごとに保持する。
	// GridElement(struct)キーのためserde不可、入場時リセット方針なのでロード後は空で再構築する
	ExploredTiles map[GridElement]bool `json:"-"`
}

// NewStageMeta は初期化された StageMeta を返す。ExploredTiles を空 map で確保する。
func NewStageMeta() *StageMeta {
	return &StageMeta{
		ExploredTiles: make(map[GridElement]bool),
	}
}
