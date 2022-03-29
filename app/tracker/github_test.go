package tracker

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"bytes"
	"github.com/cappuccinotm/dastracker/app/errs"
	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/lib"
	"github.com/cappuccinotm/dastracker/pkg/logx"
	"github.com/cappuccinotm/dastracker/pkg/sign"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"time"
)

func prepareGithub(t *testing.T, handlerFunc http.HandlerFunc) *Github {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, pwd, ok := r.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "user", u)
		assert.Equal(t, "token", pwd)

		handlerFunc(w, r)
	}))
	t.Cleanup(ts.Close)

	svc, err := NewGithub(GithubParams{
		Name: "name",
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
	return svc
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
		svc := prepareGithub(t, func(w http.ResponseWriter, r *http.Request) {
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
		svc := prepareGithub(t, func(w http.ResponseWriter, r *http.Request) {
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
		svc := prepareGithub(t, func(w http.ResponseWriter, r *http.Request) {
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
			TaskID: "123",
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
		svc := prepareGithub(t, func(w http.ResponseWriter, r *http.Request) {
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
		resp, err := svc.Subscribe(context.Background(), SubscribeReq{
			WebhookURL: "http://localhost/name/webhook-id",
			Vars:       lib.Vars{"events": "issue,pull_request"},
		})
		require.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, resp, SubscribeResp{TrackerRef: "123"})
	})
}

func TestGithub_HandleWebhook(t *testing.T) {
	called := sign.Signal()
	svc := &Github{
		l: logx.Nop(),
		handler: HandlerFunc(func(ctx context.Context, update store.Update) {
			assert.Equal(t, store.Update{
				URL: "update-url",
				ReceivedFrom: store.Locator{
					Tracker: "tracker",
					ID:      "12345",
				},
				Content: store.Content{
					Body:   "body",
					Title:  "title",
					Fields: nil, // todo
				},
			}, update)
			called.Done()
		}),
		name: "tracker",
	}

	ts := httptest.NewServer(http.HandlerFunc(svc.HandleWebhook))
	defer ts.Close()

	b := []byte(`{
		"action": "some action",
		"issue": {
			"number": 12345,
			"title": "title",
			"body": "body",
			"url": "update-url"
		}
	}`)

	resp, err := ts.Client().Post(ts.URL, "application/json", bytes.NewReader(b))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	require.NoError(t, called.WaitTimeout(5*time.Second))
}

func TestGithub_Unsubscribe(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/owner/name/hooks/tracker-ref", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	svc := &Github{
		baseURL: ts.URL,
		repo:    struct{ Owner, Name string }{Owner: "owner", Name: "name"},
		cl:      ts.Client(),
	}

	err := svc.Unsubscribe(context.Background(), UnsubscribeReq{TrackerRef: "tracker-ref"})
	require.NoError(t, err)
}
