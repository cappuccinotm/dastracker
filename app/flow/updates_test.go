package flow

import (
	"context"
	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/app/tracker"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestUpdatesDispatcher_Run(t *testing.T) {
	runCtx, cancelRunCtx := context.WithCancel(context.Background())
	updCh := make(chan store.Update)
	expectedUpd := store.Update{
		URL: "update-url",
		ReceivedFrom: store.Locator{
			Tracker: "tracker1",
			TaskID:  "task-id",
		},
		Content: store.Content{
			Body:   "task-body",
			Title:  "task-title",
			Fields: map[string]string{"field1": "fieldval1"},
		},
	}

	trk := &tracker.InterfaceMock{
		SubscribeFunc: func(_ context.Context, vars store.Vars) (<-chan store.Update, error) {
			assert.Equal(t, store.VarsFromMap(map[string]string{
				"tracker1var1": "tracker1val1",
				"tracker1var2": "tracker1val2",
			}), vars)
			return updCh, nil
		},
		CloseFunc: func(_ context.Context) error { return nil },
	}

	actor := &updatesHandlerMock{
		HandleUpdateFunc: func(_ context.Context, jobName string, upd store.Update) error {
			defer cancelRunCtx()
			assert.Equal(t, "action1", jobName)
			assert.Equal(t, expectedUpd, upd)
			return nil
		}}

	ul := &UpdatesDispatcher{
		Trackers:          map[string]tracker.Interface{"tracker1": trk},
		Handler:           actor,
		ProcessingTimeout: 5 * time.Second,
	}

	updCh <- expectedUpd
	close(updCh)

	err := ul.Run(runCtx, []store.Trigger{{
		Name:    "trigger",
		Tracker: "tracker1",
		With: store.VarsFromMap(map[string]string{
			"tracker1var1": "tracker1val1",
			"tracker1var2": "tracker1val2",
		}),
	}}, []store.Job{{
		Name:        "job",
		TriggerName: "trigger",
		Actions: store.Sequence{{
			Name:     "action1",
			With:     store.VarsFromMap(map[string]string{"a1v1": "a1v1"}),
			Detached: false,
		}},
	}})
	assert.ErrorIs(t, err, context.Canceled)

	err = ul.Close(context.Background())
	assert.NoError(t, err)

	assert.Len(t, trk.CloseCalls(), 1)
	assert.Len(t, trk.SubscribeCalls(), 1)
	assert.Len(t, actor.HandleUpdateCalls(), 1)
}
