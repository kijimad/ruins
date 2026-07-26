package interior

// Archetype は家具型が自分の配置作法を知る定義。content レシピは Ref と個数だけ言えばよく、置き方は
// archetype が既定する。同じ家具型は施設をまたいでも同じ置き方になり、レシピ間の重複と食い違いを防ぐ。
//
// doc の Archetype は footprint / clearance / needs_wall / tags まで持つが、まず消費する Default だけを
// 置き、残りは使う段で足す。将来は raw/TOML の語彙へ移す。
type Archetype struct {
	Default Placement // 既定の置き方。Stuff が Placement を空にしたとき使う
}

// defaultArchetypes は現行デモ什器の既定配置。placement を毎レシピに書かず、家具型の自然な置き方を
// 1箇所へ集約する。レシピは Placement を空にすればここを引き、自然な既定と違う例外だけ明示上書きする。
// これは Go 定義の暫定カタログで、器が固まったら raw/TOML の語彙へ移す。
var defaultArchetypes = map[string]Archetype{
	// 店
	"register":      {Default: PlaceNearDoor},
	"gondola":       {Default: PlaceRow},
	"walkin_cooler": {Default: PlaceFarFromDoor},
	"snacks":        {Default: PlaceCenter},
	"drinks":        {Default: PlaceFarFromDoor},
	"bento":         {Default: PlaceWall},
	"litter":        {Default: PlaceFullArea},
	// 診療所
	"reception":  {Default: PlaceNearDoor},
	"waitchair":  {Default: PlaceNearDoor},
	"exam_bed":   {Default: PlaceFarFromDoor},
	"medcabinet": {Default: PlaceWall},
	"meds":       {Default: PlaceFarFromDoor},
	"bandage":    {Default: PlaceWall},
	// 民家・建物
	"bed":     {Default: PlaceFarFromDoor},
	"table":   {Default: PlaceCenter},
	"chair":   {Default: PlaceCenter},
	"sofa":    {Default: PlaceWall},
	"closet":  {Default: PlaceWall},
	"lantern": {Default: PlaceWall},
	"plant":   {Default: PlaceFullArea},
	"pantry":  {Default: PlaceRow},
	"barrel":  {Default: PlaceRow},
	"bathtub": {Default: PlaceFarFromDoor},
	"toilet":  {Default: PlaceWall},
	"sink":    {Default: PlaceWall},
	"washer":  {Default: PlaceWall},
}

// placementOf は配置指示の置き方を決める。明示 Placement を最優先し、空なら archetype の既定、それも
// 無ければ全域とする。レシピが Ref だけ書けば家具型の自然な置き方に落ち、例外だけ明示上書きできる。
func placementOf(ref string, explicit Placement) Placement {
	if explicit != "" {
		return explicit
	}
	if a, ok := defaultArchetypes[ref]; ok && a.Default != "" {
		return a.Default
	}
	return PlaceFullArea
}
