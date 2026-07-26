package interior

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestClassifyRoom_施設が役割どおりに分類される は role-detector の QA を固定する。生成した内装を配置から
// 逆推定した役割が、意図した役割と一致すること。ズレたら店が店に見えない生成失敗を機械的に検出できる。
func TestClassifyRoom_施設が役割どおりに分類される(t *testing.T) {
	t.Parallel()

	byRole := houseRoomContents()
	cases := []struct {
		name string
		role string
		got  []Placed
	}{
		{"店", "store", FillRoom(42, storeRoom(), storeContent())},
		{"診療所", "clinic", FillRoom(7, clinicRoom(), clinicContent())},
		{"寝室", "bedroom", FillRoom(1, houseSmallRoom(), byRole["bedroom"])},
		{"浴室", "bath", FillRoom(1, houseSmallRoom(), byRole["bath"])},
		{"台所", "kitchen", FillRoom(1, houseSmallRoom(), byRole["kitchen"])},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, c.role, classifyRoom(c.got), "%s は %s に分類される", c.name, c.role)
		})
	}
}

// TestClassifyRoom_多seedで店と診療所は役割どおりに見える は生成の頑健性を QA する。どの seed でも、
// また経年で略奪・撤去を受けても、店は店・診療所は診療所に分類できること。到達性修復が署名家具を根こそぎ
// 撤去するような退行があれば、目視の golden より前にここで落ちる。
func TestClassifyRoom_多seedで店と診療所は役割どおりに見える(t *testing.T) {
	t.Parallel()

	for seed := range uint64(50) {
		store := Age(seed, storeRoom(), FillRoom(seed, storeRoom(), storeContent()))
		assert.Equalf(t, "store", classifyRoom(store), "seed=%d の店は店に見える", seed)

		clinic := FillRoom(seed, clinicRoom(), clinicContent())
		assert.Equalf(t, "clinic", classifyRoom(clinic), "seed=%d の診療所は診療所に見える", seed)
	}
}

// TestClassifyRoom_特徴のない配置はunknown は、署名家具を持たない部屋を役割不明と返すことを固定する。
// 廊下や玄関のような特徴の薄い部屋を無理に施設へ寄せず、reject 判定の土台にする。
func TestClassifyRoom_特徴のない配置はunknown(t *testing.T) {
	t.Parallel()

	placed := []Placed{
		{Kind: KindDecor, Ref: "plant", Pos: Vec{X: 2, Y: 2}},
		{Kind: KindDecor, Ref: "litter", Pos: Vec{X: 3, Y: 3}},
	}
	assert.Equal(t, "unknown", classifyRoom(placed))
	assert.Equal(t, "unknown", classifyRoom(nil))
}

// houseSmallRoom は水回りなど狭い部屋を模した検証用の器。入口は下辺中央。
func houseSmallRoom() Room {
	return Room{Rect: Rect{X: 0, Y: 0, W: 10, H: 8}, Doorways: []Doorway{{X: 5, Y: 7}}}
}
