package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/i18n"

	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/stretchr/testify/assert"
)

func TestBodyPart_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		bp   BodyPart
		want string
	}{
		{BodyPartHead, "head"},
		{BodyPartTorso, "torso"},
		{BodyPartArms, "arm"},
		{BodyPartHands, "hand"},
		{BodyPartLegs, "leg"},
		{BodyPartFeet, "foot"},
		{BodyPartWholeBody, "whole_body"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.bp.String())
		})
	}
}

func TestBodyPart_String_InvalidPanic(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() {
		_ = BodyPart(99).String()
	})
}

func TestElementName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		et   rawschema.Element
		want i18n.Msgid
	}{
		{rawschema.ElementNONE, "None"},
		{rawschema.ElementFIRE, "Fire"},
		{rawschema.ElementTHUNDER, "Thunder"},
		{rawschema.ElementCHILL, "Ice"},
		{rawschema.ElementPHOTON, "Light"},
	}
	for _, tt := range tests {
		t.Run(string(tt.want), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, ElementName(tt.et))
		})
	}
}

func TestElementName_InvalidPanic(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() {
		_ = ElementName(rawschema.Element("INVALID"))
	})
}
