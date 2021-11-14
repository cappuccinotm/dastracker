package service

//
//func TestActor_handleUpdate(t *testing.T) {
//	t.Listen("task just initiated", func(t *testing.T) {
//		trk := &tracker.InterfaceMock{}
//		eng := &engine.InterfaceMock{}
//		fl := &flow.InterfaceMock{}
//
//		s := &Actor{
//			Tracker:       trk,
//			Engine:        eng,
//			Flow:          fl,
//			Log:           log.Default(),
//			UpdateTimeout: 5 * time.Second,
//		}
//
//		upd := store.Update{
//			TriggerName: "created-trigger",
//			URL:         "https://example.com",
//			ReceivedFrom: store.Locator{
//				Tracker: "tracker",
//				TaskID:  "task-id",
//			},
//			Content: store.Content{
//				Body:  "foo",
//				Title: "bar",
//				Fields: map[string]string{
//					"field1": "value1",
//					"field2": "value2",
//					"field3": "value3",
//				},
//			},
//		}
//
//		updates := make(chan store.Update)
//
//		trk.UpdatesFunc = func() <-chan store.Update { return updates }
//		fl.GetSubscribedJobsFunc = func(_ context.Context, triggerName string) ([]store.Job, error) {
//			assert.Equal(t, "created-trigger", triggerName)
//			return []store.Job{{
//				Actions: store.Sequence{{
//					Name: "create",
//					With: store.VarsFromMap(map[string]string{"var1": "val1", "var2": "val2"}),
//				}},
//			}}, nil
//		}
//		eng.GetFunc = func(_ context.Context, req engine.GetRequest) (store.Ticket, error) {
//			assert.Equal(t, engine.GetRequest{Locator: store.Locator{
//				Tracker: "tracker",
//				TaskID:  "task-id",
//			}}, req)
//			return store.Ticket{}, engine.ErrNotFound
//		}
//		trk.CallFunc = func(_ context.Context, req tracker.Request) (tracker.Response, error) {
//			assert.Equal(t, tracker.Request{
//				Method: "create",
//				Vars: store.EvaluatedVarsFromMap(map[string]string{
//					"var1": "val1",
//					"var2": "val2",
//				}),
//				Ticket: store.Ticket{TrackerIDs: map[string]string{}, Content: upd.Content},
//			}, req)
//			return tracker.Response{Tracker: "tracker", TaskID: "task-id"}, nil
//		}
//		eng.CreateFunc = func(_ context.Context, ticket store.Ticket) (string, error) {
//			assert.Equal(t, store.Ticket{
//				TrackerIDs: map[string]string{
//					"tracker": "task-id",
//				},
//				Content: store.Content{
//					Body:  "foo",
//					Title: "bar",
//					Fields: map[string]string{
//						"field1": "value1",
//						"field2": "value2",
//						"field3": "value3",
//					},
//				},
//			}, ticket)
//			return "new-ticket-id", nil
//		}
//
//		closed := atomicFalse
//
//		updateRead := atomicFalse
//		go func() {
//			updates <- upd
//			atomic.StoreInt32(&updateRead, atomicTrue)
//		}()
//
//		go func() {
//			assert.ErrorIs(t, s.Listen(context.Background()), context.Canceled)
//			atomic.StoreInt32(&closed, atomicTrue)
//		}()
//
//		waitTimeout(t, &updateRead, defaultTestTimeout, "update haven't been read")
//		assert.Len(t, trk.UpdatesCalls(), 1)
//		assert.Len(t, fl.GetSubscribedJobsCalls(), 1)
//		assert.Len(t, eng.GetCalls(), 1)
//		assert.Len(t, trk.CallCalls(), 1)
//		assert.Len(t, eng.CreateCalls(), 1)
//
//		trk.CloseFunc = func(_ context.Context) error { return nil }
//		assert.NoError(t, s.Close(context.Background()))
//		waitTimeout(t, &closed, defaultTestTimeout, "close didn't finish within timeout")
//	})
//}
