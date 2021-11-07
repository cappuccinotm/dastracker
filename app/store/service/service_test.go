package service

import (
	"context"
	"github.com/cappuccinotm/dastracker/app/flow"
	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/app/store/engine"
	"github.com/cappuccinotm/dastracker/app/tracker"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDataStore_TaskUpdatedCallback(t *testing.T) {
	t.Run("task just initiated", func(t *testing.T) {
		eng := &engine.InterfaceMock{}
		trk := &tracker.InterfaceMock{}
		s := &DataStore{
			eng:      eng,
			trackers: map[string]tracker.Interface{"tracker-to-create": trk},
		}

		eng.GetFunc = func(_ context.Context, req engine.GetRequest) (store.Ticket, error) {
			assert.Equal(t, engine.GetRequest{Locator: store.Locator{
				Tracker: "received-from-tracker",
				TaskID:  "received-task-id",
			}}, req)
			return store.Ticket{}, engine.ErrNotFound
		}

		trk.CallFunc = func(_ context.Context, req tracker.Request) (tracker.Response, error) {
			assert.Equal(t, tracker.Request{
				Method: "create",
				Vars: store.EvaluatedVarsFromMap(map[string]string{
					"var1": "val1",
					"var2": "val2",
				}),
			}, req)
			return tracker.Response{TaskID: "created-task-id"}, nil
		}

		eng.CreateFunc = func(_ context.Context, ticket store.Ticket) (string, error) {
			assert.Equal(t, store.Ticket{
				TrackerIDs: map[string]string{
					"tracker-to-create": "created-task-id",
				},
				Content: store.Content{
					Body:  "foo",
					Title: "bar",
					Fields: map[string]string{
						"field1": "value1",
						"field2": "value2",
						"field3": "value3",
					},
				},
			}, ticket)
			return "new-ticket-id", nil
		}

		err := s.onTaskUpdated(context.Background(), flow.Job{
			Actions: flow.Sequence{{
				Name: "tracker-to-create/create",
				With: store.VarsFromMap(map[string]string{
					"var1": "val1",
					"var2": "val2",
				}),
			}},
		}, store.Update{
			URL: "ticket-updated-url",
			ReceivedFrom: store.Locator{
				Tracker: "received-from-tracker",
				TaskID:  "received-task-id",
			},
			Content: store.Content{
				Body:  "foo",
				Title: "bar",
				Fields: map[string]string{
					"field1": "value1",
					"field2": "value2",
					"field3": "value3",
				},
			},
		})
		assert.NoError(t, err)
	})

	t.Run("task updated", func(t *testing.T) {
		eng := &engine.InterfaceMock{}
		trk := &tracker.InterfaceMock{}
		s := &DataStore{
			eng:      eng,
			trackers: map[string]tracker.Interface{"update-tracker": trk},
		}

		eng.GetFunc = func(_ context.Context, req engine.GetRequest) (store.Ticket, error) {
			assert.Equal(t, engine.GetRequest{Locator: store.Locator{
				Tracker: "received-from-tracker",
				TaskID:  "received-task-id",
			}}, req)
			return store.Ticket{
				ID: "ticket-id",
				TrackerIDs: map[string]string{
					"update-tracker":        "update-task-id",
					"received-from-tracker": "received-task-id",
				},
				Content: store.Content{},
			}, nil
		}

		trk.CallFunc = func(_ context.Context, req tracker.Request) (tracker.Response, error) {
			assert.Equal(t, tracker.Request{
				Method: "update",
				Vars: store.EvaluatedVarsFromMap(map[string]string{
					"var1": "val1",
					"var2": "val2",
				}),
				TaskID: "update-task-id",
			}, req)
			return tracker.Response{TaskID: "update-task-id"}, nil
		}

		eng.UpdateFunc = func(_ context.Context, ticket store.Ticket) error {
			assert.Equal(t, store.Ticket{
				ID: "ticket-id",
				TrackerIDs: map[string]string{
					"update-tracker":        "update-task-id",
					"received-from-tracker": "received-task-id",
				},
				Content: store.Content{
					Body:  "foo",
					Title: "bar",
					Fields: map[string]string{
						"field1": "value1",
						"field2": "value2",
						"field3": "value3",
					},
				},
			}, ticket)
			return nil
		}

		err := s.onTaskUpdated(context.Background(), flow.Job{
			Actions: flow.Sequence{{
				Name: "update-tracker/update",
				With: store.VarsFromMap(map[string]string{
					"var1": "val1",
					"var2": "val2",
				}),
			}},
		}, store.Update{
			URL: "ticket-updated-url",
			ReceivedFrom: store.Locator{
				Tracker: "received-from-tracker",
				TaskID:  "received-task-id",
			},
			Content: store.Content{
				Body:  "foo",
				Title: "bar",
				Fields: map[string]string{
					"field1": "value1",
					"field2": "value2",
					"field3": "value3",
				},
			},
		})
		assert.NoError(t, err)
	})
}
