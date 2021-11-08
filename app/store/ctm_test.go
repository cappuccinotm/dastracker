package store

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAction_Path(t *testing.T) {
	t.Run("valid name", func(t *testing.T) {
		trk, mtd := Action{Name: "tracker/method"}.Path()
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
				trk, mtd := Action{Name: tt.actionName}.Path()
				assert.Equal(t, "", trk)
				assert.Equal(t, "", mtd)
			})
		}
	})
}
