package tracker

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cappuccinotm/dastracker/app/errs"
	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/app/webhook"
	"github.com/cappuccinotm/dastracker/lib"
	"github.com/cappuccinotm/dastracker/pkg/logx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func prepareGithubTestEnv(t *testing.T, handlerFunc http.HandlerFunc) (*Github, *webhook.InterfaceMock) {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, pwd, ok := r.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "user", u)
		assert.Equal(t, "token", pwd)

		handlerFunc(w, r)
	}))
	t.Cleanup(ts.Close)

	whm := &webhook.InterfaceMock{
		RegisterFunc: func(name string, handler http.Handler) error {
			assert.Equal(t, "name", name)
			assert.NotNil(t, handler)
			return nil
		},
	}

	svc, err := NewGithub(GithubParams{
		Name:           "name",
		WebhookManager: whm,
		Vars: lib.Vars{
			"owner":        "repo-owner",
			"name":         "repo-name",
			"user":         "user",
			"access_token": "token",
		},
		Client: ts.Client(),
		Logger: logx.Nop(),
	})
	require.NoError(t, err)
	svc.baseURL = ts.URL
	return svc, whm
}

func requireJSONMarshal(t *testing.T, src interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(src)
	require.NoError(t, err)
	return b
}

func TestGithub_Name(t *testing.T) {
	assert.Equal(t, "Github[blah]", (&Github{name: "blah"}).Name())
}

func TestGithub_updateOrCreateIssue(t *testing.T) {
	type issue struct {
		Title     string   `json:"title"`
		Body      string   `json:"body"`
		Assignees []string `json:"assignees"`
		Milestone string   `json:"milestone"`
	}

	t.Run("create", func(t *testing.T) {
		called := false
		svc, _ := prepareGithubTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, r.URL.Path, "/repos/repo-owner/repo-name/issues")
			assert.Equal(t, r.Method, "POST")
			b, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.NoError(t, r.Body.Close())

			var resp issue
			require.NoError(t, json.Unmarshal(b, &resp))

			assert.Equal(t, issue{
				Title:     "title",
				Body:      "body",
				Assignees: []string{"assignee1", "assignee2"},
				Milestone: "milestone",
			}, resp)

			_, err = w.Write([]byte(`{"id": 123}`))
			require.NoError(t, err)
			called = true
		})

		resp, err := svc.Call(context.Background(), Request{
			Method: "UpdateOrCreateIssue",
			Ticket: store.Ticket{},
			Vars: lib.Vars{
				"title":     "title",
				"body":      "body",
				"assignees": "assignee1,assignee2",
				"milestone": "milestone",
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "123", resp.TaskID)
		assert.True(t, called)
	})

	t.Run("unexpected status", func(t *testing.T) {
		called := false
		svc, _ := prepareGithubTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, r.URL.Path, "/repos/repo-owner/repo-name/issues")
			assert.Equal(t, r.Method, "POST")
			b, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.NoError(t, r.Body.Close())

			var resp issue
			require.NoError(t, json.Unmarshal(b, &resp))

			assert.Equal(t, issue{
				Title:     "title",
				Body:      "body",
				Assignees: []string{"assignee1", "assignee2"},
				Milestone: "milestone",
			}, resp)

			w.WriteHeader(http.StatusBadRequest)
			_, err = w.Write([]byte(`{"message": "some-github-error"}`))
			require.NoError(t, err)
			called = true
		})

		resp, err := svc.Call(context.Background(), Request{
			Method: "UpdateOrCreateIssue",
			Ticket: store.Ticket{},
			Vars: lib.Vars{
				"title":     "title",
				"body":      "body",
				"assignees": "assignee1,assignee2",
				"milestone": "milestone",
			},
		})
		assert.Empty(t, resp)
		var rerr errs.ErrGithubAPI
		require.ErrorAs(t, err, &rerr)
		assert.Equal(t, http.StatusBadRequest, rerr.ResponseStatus)
		assert.Equal(t, "some-github-error", rerr.Message)
		assert.True(t, called)
	})

	t.Run("update", func(t *testing.T) {
		called := false
		svc, _ := prepareGithubTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, r.URL.Path, "/repos/repo-owner/repo-name/issues/123")
			assert.Equal(t, r.Method, "PATCH")
			b, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.NoError(t, r.Body.Close())
			var resp issue
			require.NoError(t, json.Unmarshal(b, &resp))

			assert.Equal(t, issue{
				Title:     "title",
				Body:      "body",
				Assignees: []string{"assignee1", "assignee2"},
				Milestone: "milestone",
			}, resp)

			_, err = w.Write([]byte(`{"id": 123}`))
			require.NoError(t, err)
			called = true
		})

		resp, err := svc.Call(context.Background(), Request{
			Method: "UpdateOrCreateIssue",
			Ticket: store.Ticket{TrackerIDs: map[string]string{"name": "123"}},
			Vars: lib.Vars{
				"title":     "title",
				"body":      "body",
				"assignees": "assignee1,assignee2",
				"milestone": "milestone",
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "123", resp.TaskID)
		assert.True(t, called)
	})
}

func TestGithub_Subscribe(t *testing.T) {
	type hookReqCfg struct {
		URL         string `json:"url"`
		ContentType string `json:"content_type"`
		InsecureSSL string `json:"insecure_ssl"`
	}
	type hookReq struct {
		Config hookReqCfg `json:"config"`
		Events []string   `json:"events"`
		Active bool       `json:"active"`
	}

	t.Run("success", func(t *testing.T) {
		called := false
		svc, whm := prepareGithubTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, r.URL.Path, "/repos/repo-owner/repo-name/hooks")
			assert.Equal(t, r.Method, "POST")
			b, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.NoError(t, r.Body.Close())

			var req hookReq
			require.NoError(t, json.Unmarshal(b, &req))
			assert.Equal(t, hookReq{
				Config: hookReqCfg{
					URL:         "http://localhost/name/webhook-id",
					ContentType: "json",
				},
				Events: []string{"issue", "pull_request"},
				Active: true,
			}, req)

			w.WriteHeader(http.StatusCreated)
			_, err = w.Write([]byte(`{"id": 123}`))
			require.NoError(t, err)
			called = true
		})

		whm.CreateFunc = func(ctx context.Context, tracker string, trigger string) (store.Webhook, error) {
			assert.Equal(t, "name", tracker)
			assert.Equal(t, "trigger-name", trigger)
			return store.Webhook{
				ID:          "webhook-id",
				TrackerName: "name",
				TriggerName: "trigger-name",
				BaseURL:     "http://localhost/",
			}, nil
		}

		whm.SetTrackerIDFunc = func(ctx context.Context, webhookID string, trackerID string) error {
			assert.Equal(t, "webhook-id", webhookID)
			assert.Equal(t, "123", trackerID)
			return nil
		}

		err := svc.Subscribe(context.Background(), SubscribeReq{
			TriggerName: "trigger-name",
			Vars:        lib.Vars{"events": "issue,pull_request"},
		})
		require.NoError(t, err)
		assert.True(t, called)
	})
}

func TestGithub_Listen(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &Github{
			l: logx.Nop(),
			handler: HandlerFunc(func(ctx context.Context, update store.Update) {
				assert.Equal(t, store.Update{
					TriggerName:  "trigger-name",
					URL:          "url",
					ReceivedFrom: store.Locator{Tracker: "tracker-name", ID: "12347"},
					Content: store.Content{
						Body:   "description",
						Title:  "title",
						Fields: nil, // todo
					},
				}, update)
			}),
		}

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			svc.whHandler(w, r.WithContext(webhook.PutWebhook(r.Context(), store.Webhook{
				ID:          "webhook-id",
				TrackerRef:  "tracker-ref",
				TrackerName: "tracker-name",
				TriggerName: "trigger-name",
				BaseURL:     "http://localhost/",
			})))
		}))
		defer ts.Close()

		req, err := http.NewRequest(http.MethodPost, ts.URL, bytes.NewReader(
			requireJSONMarshal(t, map[string]interface{}{
				"action": "update",
				"issue": map[string]interface{}{
					"number":      12347,
					"title":       "title",
					"description": "description",
					"url":         "url",
				},
			})))
		require.NoError(t, err)

		_, err = ts.Client().Do(req)
		require.NoError(t, err)

		// todo, do we need a special response for github?
	})
}
