package flow

import (
	"context"
	"github.com/cappuccinotm/dastracker/app/provider"
	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/app/store/engine"
	"github.com/cappuccinotm/dastracker/app/tracker"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDataStore_onTaskUpdated(t *testing.T) {
	t.Run("task just initiated", func(t *testing.T) {
		eng := &engine.InterfaceMock{}
		trk := &tracker.InterfaceMock{}
		prv := &provider.InterfaceMock{}
		s := &DataStore{
			Engine:   eng,
			Trackers: map[string]tracker.Interface{"tracker-to-create": trk},
			Provider: prv,
		}

		prv.GetJobFunc = func(_ context.Context, name string) (store.Job, error) {
			assert.Equal(t, "tracker-to-create/create", name)
			return store.Job{
				Actions: store.Sequence{{
					Name: "tracker-to-create/create",
					With: store.VarsFromMap(map[string]string{
						"var1": "val1",
						"var2": "val2",
					}),
				}},
			}, nil
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

		err := s.HandleUpdate(context.Background(), "tracker-to-create/create", store.Update{
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
		prv := &provider.InterfaceMock{}
		s := &DataStore{
			Engine:   eng,
			Trackers: map[string]tracker.Interface{"update-tracker": trk},
			Provider: prv,
		}

		prv.GetJobFunc = func(_ context.Context, name string) (store.Job, error) {
			assert.Equal(t, "update-tracker/update", name)
			return store.Job{
				Actions: store.Sequence{{
					Name: "update-tracker/update",
					With: store.VarsFromMap(map[string]string{
						"var1": "val1",
						"var2": "val2",
					}),
				}},
			}, nil
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

		err := s.HandleUpdate(context.Background(), "update-tracker/update", store.Update{
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
