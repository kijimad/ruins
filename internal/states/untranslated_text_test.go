package states_test

import (
	"slices"
	"strings"
	"sync"
	"testing"

	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/i18n"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/vrt/replay"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"

	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// textCollector は1つの world のフェイスで描かれた文字列を集める。
type textCollector struct {
	mu    sync.Mutex
	faces map[text.Face][]string
}

func (c *textCollector) observe(face text.Face, s string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if drawn, ok := c.faces[face]; ok {
		c.faces[face] = append(drawn, s)
	}
}

func (c *textCollector) register(world w.World) {
	c.mu.Lock()
	defer c.mu.Unlock()
	tr := world.Resources.UIResources.Text
	for _, f := range []text.Face{tr.SmallFace, tr.BodyFace, tr.KeycapFace, tr.TitleFontFace, tr.SplashFontFace} {
		c.faces[f] = nil
	}
}

func (c *textCollector) drawn(world w.World) []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	tr := world.Resources.UIResources.Text
	var out []string
	for _, f := range []text.Face{tr.SmallFace, tr.BodyFace, tr.KeycapFace, tr.TitleFontFace, tr.SplashFontFace} {
		out = append(out, c.faces[f]...)
	}
	return out
}

// untranslatedExemptCases は開発者向けで訳さない場面。
var untranslatedExemptCases = []string{"DebugMenu", "ComponentDebug"}

// languageNeutralWords は訳さずに描いてよい英字の語。1文字の語はキーや記号や単位なので常に許す。
// Ash はプレイヤーの名前。
var languageNeutralWords = []string{"HP", "Esc", "Enter", "Ash"}

// TestUntranslatedText は golden の全場面を疑似訳で描き、訳を通らずに描かれた文字列を探す。
func TestUntranslatedText(t *testing.T) {
	t.Parallel()

	collector := &textCollector{faces: map[text.Face][]string{}}
	uicore.ObserveText(collector.observe)
	t.Cleanup(func() { uicore.ObserveText(nil) })

	for _, tc := range goldenCases {
		if slices.Contains(untranslatedExemptCases, tc.name) {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actions := make([]inputmapper.ActionID, len(tc.steps))
			for i, s := range tc.steps {
				actions[i] = s.action
			}
			var world w.World
			replay.PlayScenario(t,
				func(wd w.World) []es.State[w.World] {
					world = wd
					query.GetUserSettings(world).Language = i18n.PseudoLang
					collector.register(world)
					built, err := tc.build(world)
					require.NoError(t, err)
					return built
				},
				actions,
				func(int, w.World, *ebiten.Image) {},
			)

			var leaks []string
			for _, s := range collector.drawn(world) {
				if hasUntranslatedWord(s) && !slices.Contains(leaks, s) {
					leaks = append(leaks, s)
				}
			}
			assert.Empty(t, leaks, "訳を通らずに描かれた文字列がある")
		})
	}
}

// hasUntranslatedWord は s に languageNeutralWords 以外の2文字以上の英字の語があるかを返す。
func hasUntranslatedWord(s string) bool {
	words := strings.FieldsFunc(s, func(r rune) bool { return !isASCIILetter(r) })
	return slices.ContainsFunc(words, func(word string) bool {
		return len(word) > 1 && !slices.Contains(languageNeutralWords, word)
	})
}

func isASCIILetter(r rune) bool {
	return 'a' <= r && r <= 'z' || 'A' <= r && r <= 'Z'
}
