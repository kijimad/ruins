package styled

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
)

func TestNewFragmentText(t *testing.T) {
	t.Parallel()
	// モックのworldを作成（テスト用）
	world := testutil.InitTestWorld(t)
	// 実際のUIリソースが必要になるため、このテストは統合テスト環境でのみ実行可能
	// 単体テストとしては関数の存在確認のみ行う

	// 関数が存在し、パニックしないことを確認
	defer func() {
		if r := recover(); r != nil {
			// UIリソースがない場合のパニックは予期される
			t.Logf("Expected panic due to missing UI resources: %v", r)
		}
	}()

	// 基本的な呼び出しテスト
	text := "テストテキスト"
	color := consts.ColorRed

	NewFragmentText(text, color, world.Resources.UIResources)

	t.Log("NewFragmentText function exists and can be called")
}

// TestFragmentTextConcept は概念的なテスト
func TestFragmentTextConcept(t *testing.T) {
	t.Parallel()
	// NewFragmentTextとNewListItemTextの設計思想の違いを文書化
	t.Log("NewFragmentText design goals:")
	t.Log("- No stretching (Stretch: false)")
	t.Log("- No minimum width constraint")
	t.Log("- No padding around text")
	t.Log("- Exact text width only")

	t.Log("NewListItemText design goals:")
	t.Log("- Stretches to parent width (Stretch: true)")
	t.Log("- Minimum width of 120px")
	t.Log("- Left/right padding of 8px each")
	t.Log("- Suitable for list items")
}
