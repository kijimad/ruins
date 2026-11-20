package raw

import (
	"testing"

	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/stretchr/testify/require"
)

func TestCommandEntryMelee_引けない行はエラー(t *testing.T) {
	t.Parallel()

	raws := rawschema.Raws{Items: &[]rawschema.Item{{Id: "rock", Name: "Rock"}}}
	tests := []struct {
		name    string
		entry   rawschema.CommandTableEntry
		wantErr error
	}{
		{name: "近接攻撃を持たないアイテム", entry: rawschema.CommandTableEntry{Item: new("rock"), Weight: 1}, wantErr: errItemNoMelee},
		{name: "どちらも持たない行", entry: rawschema.CommandTableEntry{Weight: 1}, wantErr: errCommandEntryTarget},
		{name: "無い体の攻撃", entry: rawschema.CommandTableEntry{BodyAttack: new("typo"), Weight: 1}, wantErr: KeyNotFoundError{Key: "typo", Collection: "BodyAttacks"}},
		{name: "無いアイテム", entry: rawschema.CommandTableEntry{Item: new("typo"), Weight: 1}, wantErr: KeyNotFoundError{Key: "typo", Collection: "Items"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, _, err := CommandEntryMelee(raws, tt.entry)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}
