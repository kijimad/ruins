package components

// StageField はステージごとのフィールド状態を保持する。
// StageBound で各ステージに束縛され、他のフィールドエンティティと同様に共存・退避・serde される。
// 現ステージの StageField は Dungeon.CurrentStage で引く。
//
// オーバーワールドもダンジョン階も同じ「ステージ」で、種別は持たない。違いは保有データだけにする。
// 帯データは SeamlessBand コンポーネントとしてオーバーワールドの StageField だけが持ち、
// その有無が「オーバーワールドか」の判別を兼ねる。ダンジョン階の StageField は持たない。
type StageField struct {
	// Level は現ステージのフィールド寸法。ステージごとに保持するため、往復してもステージ固有の
	// 寸法が resume で自然に戻る。
	Level Level
	// BaseTemp はステージの基本気温。摂氏。ステージ生成時に dungeon 定義から確定して写す。
	// 気温計算を query 層に置くため、dungeon 登録表への依存をここで断つ。登録表は別層なので
	// query からは読めず、Level と同じくステージ生成時に定まる設定として保持する。
	BaseTemp int
	// ExploredTiles は探索済みタイルのマップ。ステージごとに保持する。
	// GridElement(struct)キーのためserde不可、入場時リセット方針なのでロード後は空で再構築する
	ExploredTiles map[GridElement]bool `json:"-"`
}

// NewStageField は初期化された StageField を返す。ExploredTiles を空 map で確保する。
func NewStageField() *StageField {
	return &StageField{
		ExploredTiles: make(map[GridElement]bool),
	}
}
