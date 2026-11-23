package overworld

import (
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io/fs"
	"sort"

	"github.com/kijimaD/ruins/assets"
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	xdraw "golang.org/x/image/draw"
)

// RegionSpritePx はスプライトの PNG の辺。RenderRegion に渡すとスプライトを縮めずに置く
const RegionSpritePx = 32

// regionBackground は描くタイルの無いマスの色
var regionBackground = color.RGBA{R: 20, G: 18, B: 15, A: 255}

// RenderRegion は帯ローカルのタイルの矩形 area にあるタイルと置物を、1タイル tilePx 四方で1枚の画像に描く。
// 各エンティティの SpriteRender が指すスプライトの PNG を assets から引き、Depth の順、同じ Depth は床、置物の順に重ねる。
// tilePx がスプライトより小さければ最近傍で縮める。ゲーム本体と同じスプライトで、生成の結果そのものを写す
func RenderRegion(world w.World, area image.Rectangle, tilePx int) (*image.RGBA, error) {
	img := image.NewRGBA(image.Rect(0, 0, area.Dx()*tilePx, area.Dy()*tilePx))
	draw.Draw(img, img.Bounds(), image.NewUniform(regionBackground), image.Point{}, draw.Src)

	type ent struct {
		g  gc.GridElement
		sr gc.SpriteRender
	}
	var ents []ent
	q := query.ActiveFilter2[gc.GridElement, gc.SpriteRender](world).Query()
	for q.Next() {
		e := q.Entity()
		ge := *world.Components.GridElement.Get(e)
		if !(image.Point{X: int(ge.X), Y: int(ge.Y)}).In(area) {
			continue
		}
		ents = append(ents, ent{g: ge, sr: *world.Components.SpriteRender.Get(e)})
	}
	sort.SliceStable(ents, func(i, j int) bool {
		if ents[i].sr.Depth.Rank() != ents[j].sr.Depth.Rank() {
			return ents[i].sr.Depth.Rank() < ents[j].sr.Depth.Rank()
		}
		if isTile := ents[i].sr.SpriteSheetName == "tile"; isTile != (ents[j].sr.SpriteSheetName == "tile") {
			return isTile
		}
		if ents[i].g.Y != ents[j].g.Y {
			return ents[i].g.Y < ents[j].g.Y
		}
		return ents[i].g.X < ents[j].g.X
	})

	sprites := regionSprites{px: tilePx, cache: map[string]image.Image{}}
	for i := range ents {
		spr, ok, err := sprites.load(ents[i].sr)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		dx := (int(ents[i].g.X) - area.Min.X) * tilePx
		dy := (int(ents[i].g.Y) - area.Min.Y) * tilePx
		draw.Draw(img, image.Rect(dx, dy, dx+tilePx, dy+tilePx), spr, image.Point{}, draw.Over)
	}
	return img, nil
}

// regionSprites は RenderRegion が引いたスプライトを、px 四方に縮めて覚えておく
type regionSprites struct {
	px    int
	cache map[string]image.Image
}

// load は SpriteRender の指すスプライトを返す。タイルは tiles、置物は single の画像から引く。
// 画像の無いスプライトは ok が偽で、描かない
func (s regionSprites) load(sr gc.SpriteRender) (img image.Image, ok bool, err error) {
	var dir string
	switch sr.SpriteSheetName {
	case "tile":
		dir = "file/textures/tiles/"
	case "field":
		dir = "file/textures/single/"
	default:
		return nil, false, nil
	}
	path := dir + sr.SpriteKey + "_.png"
	if im, cached := s.cache[path]; cached {
		return im, im != nil, nil
	}
	f, err := assets.FS.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		s.cache[path] = nil
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = f.Close() }()
	im, err := png.Decode(f)
	if err != nil {
		return nil, false, err
	}
	if s.px != im.Bounds().Dx() {
		small := image.NewRGBA(image.Rect(0, 0, s.px, s.px))
		xdraw.NearestNeighbor.Scale(small, small.Bounds(), im, im.Bounds(), draw.Src, nil)
		im = small
	}
	s.cache[path] = im
	return im, true, nil
}
