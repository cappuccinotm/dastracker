package store

import (
	"os"
	"testing"

	"github.com/cappuccinotm/dastracker/app/errs"
	"github.com/cappuccinotm/dastracker/lib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvaluate(t *testing.T) {
	v := lib.Vars(map[string]string{
		"var1": `{{ env "TESTVAR" }}`,
		"var2": `{{ seq (keys .Update.Fields) }}`,
		"var3": `{{ seq (values .Update.Fields) }}`,
		"var4": `static text`,
		"var5": `{{ .Ticket.ID }}`,
		"var6": `{{ (.Ticket.Variations.Get "jira").Title }}`,
		"var7": `{{ (.Ticket.Variations.Get "gitlab").Title }}`,
	})
	err := os.Setenv("TESTVAR", "blah")
	vs, err := Evaluate(v, EvalData{
		Ticket: Ticket{
			ID: "ticket-id",
			Variations: Variations{"jira": Task{
				ID:      "task-id",
				Content: Content{Body: "task-body", Title: "task-title"},
			}},
		},
		Update: Update{Content: Content{Fields: TicketFields{
			"f1": "f1v",
			"f2": "f2v",
		}}},
	})
	assert.NoError(t, err)

	// checking field-by-field, as mustn't rely on order of walking over map
	assert.Equal(t, "blah", vs["var1"])
	assert.Contains(t, []string{"f1,f2", "f2,f1"}, vs["var2"])
	assert.Contains(t, []string{"f1v,f2v", "f2v,f1v"}, vs["var3"])
	assert.Equal(t, "static text", vs["var4"])
	assert.Equal(t, "ticket-id", vs["var5"])
	assert.Equal(t, "task-title", vs["var6"])
	assert.Equal(t, "", vs["var7"])
}

func TestIf_Eval(t *testing.T) {
	b, err := If{Condition: `string_contains(Update.Title, "[PTT]")`}.
		Eval(Update{Content: Content{
			Title: "[PTT] something",
		}})
	require.NoError(t, err)
	assert.True(t, b)

	b, err = If{Condition: `string_contains(Update.Title, "[PTT]")`}.
		Eval(Update{Content: Content{
			Title: "something",
		}})
	require.NoError(t, err)
	assert.False(t, b)

	b, err = If{Condition: `keys(Update.Fields)`}.
		Eval(Update{Content: Content{
			Fields: map[string]string{
				"k1": "v1",
				"k2": "v2",
				"k3": "v3",
			},
		}})
	assert.ErrorIs(t, err, errs.ErrIfNotBool)
	assert.False(t, b)
}
