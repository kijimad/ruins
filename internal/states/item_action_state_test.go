package states

import (
	"testing"

	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/stretchr/testify/assert"
)

func TestVerbByAction_直達アクションを動詞へ対応づける(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		action inputmapper.ActionID
		want   verbID
		ok     bool
	}{
		{"調べる", inputmapper.ActionVerbExamine, verbExamine, true},
		{"置く", inputmapper.ActionVerbPlace, verbPlace, true},
		{"食べる", inputmapper.ActionVerbConsume, verbConsume, true},
		{"読む", inputmapper.ActionVerbRead, verbRead, true},
		{"使う", inputmapper.ActionVerbUse, verbUse, true},
		{"投げるは未実装なので対応なし", inputmapper.ActionVerbThrow, verbID(""), false},
		{"動詞でないアクションは対応なし", inputmapper.ActionMenuSelect, verbID(""), false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := verbByAction(tt.action)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestVerbTabIndex_動詞の表示順を返す(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 0, verbTabIndex(verbExamine))
	assert.Equal(t, 1, verbTabIndex(verbPlace))
	assert.Equal(t, 2, verbTabIndex(verbConsume))
	assert.Equal(t, 3, verbTabIndex(verbRead))
	assert.Equal(t, 4, verbTabIndex(verbUse))
	// 未知の動詞は先頭タブへ寄せる
	assert.Equal(t, 0, verbTabIndex(verbID("unknown")))
}

func TestVerbs_調べるは実行を持たず詳細モーダルを開く(t *testing.T) {
	t.Parallel()
	for _, v := range verbs() {
		if v.ID == verbExamine {
			assert.Nil(t, v.Exec, "調べるは Exec を持たず Enter で詳細モーダルを開く")
			return
		}
	}
	assert.Fail(t, "調べるタブが verbs() に存在しない")
}
