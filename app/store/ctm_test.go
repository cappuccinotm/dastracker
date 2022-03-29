package store

import (
	"testing"

	"github.com/cappuccinotm/dastracker/app/errs"
	"github.com/cappuccinotm/dastracker/lib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestAction_Path(t *testing.T) {
	t.Run("valid name", func(t *testing.T) {
		trk, mtd, err := Action{Name: "tracker/method"}.Path()
		require.NoError(t, err)
		assert.Equal(t, "tracker", trk)
		assert.Equal(t, "method", mtd)
	})

	t.Run("invalid name", func(t *testing.T) {
		tests := []struct{ name, actionName string }{
			{name: "tracker only with slash", actionName: "tracker/"},
			{name: "method only with slash", actionName: "/method"},
			{name: "slash absent", actionName: "tracker"},
			{name: "empty string", actionName: ""},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				trk, mtd, err := Action{Name: tt.actionName}.Path()
				require.ErrorIs(t, err, errs.ErrMethodParseFailed(tt.actionName))
				assert.Equal(t, "", trk)
				assert.Equal(t, "", mtd)
			})
		}
	})
}

func TestSequence_UnmarshalYAML(t *testing.T) {
	t.Run("mixed if/action in the same sequence", func(t *testing.T) {
		const data = "\n" +
			"- action: someActionName         \n" +
			"  detached: true                 \n" +
			"  with:                          \n" +
			"    key: value                   \n" +
			"- if: someCondition              \n" +
			"  do:                            \n" +
			"    - action: someNestedAction   \n" +
			"      detached: false            \n"

		var seq Sequence
		err := yaml.Unmarshal([]byte(data), &seq)
		require.NoError(t, err)
		assert.Equal(t, Sequence{
			Action{
				Name:     "someActionName",
				Detached: true,
				With:     lib.Vars{"key": "value"},
			},
			If{
				Condition: "someCondition",
				Actions: Sequence{
					Action{
						Name:     "someNestedAction",
						Detached: false,
					},
				},
			},
		}, seq)
	})
}
