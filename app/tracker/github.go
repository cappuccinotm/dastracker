package tracker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/cappuccinotm/dastracker/app/errs"
	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/lib"
	"github.com/cappuccinotm/dastracker/pkg/httpx"
	"github.com/cappuccinotm/dastracker/pkg/logx"
)

const githubAPIURL = "https://api.github.com"

var ghSupportedActions = map[string]func(*Github, context.Context, Request) (Response, error){
	"UpdateOrCreateIssue": (*Github).updateOrCreateIssue,
}

// Github implements Interface over the github issues tracker.
type Github struct {
	baseURL string
	name    string
	l       logx.Logger
	repo    struct{ Owner, Name string }
	cl      *http.Client
	handler Handler
}

// GithubParams describes parameters to initialize Github.
type GithubParams struct {
	Name   string
	Vars   lib.Vars
	Client *http.Client
	Logger logx.Logger
}

// NewGithub makes new instance of Github tracker.
func NewGithub(params GithubParams) (*Github, error) {
	svc := &Github{
		baseURL: githubAPIURL,
		name:    params.Name,
		l:       params.Logger,
		cl:      params.Client,
	}

	svc.repo.Owner = params.Vars.Get("owner")
	svc.repo.Name = params.Vars.Get("name")

	svc.cl = &http.Client{
		Transport: httpx.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			r.SetBasicAuth(params.Vars.Get("user"), params.Vars.Get("access_token"))
			return http.DefaultTransport.RoundTrip(r)
		}),
		Timeout: 5 * time.Second,
	}

	return svc, nil
}

// Name returns the name of the tracker.
func (g *Github) Name() string { return fmt.Sprintf("Github[%s]", g.name) }

// Call handles the incoming request.
func (g *Github) Call(ctx context.Context, req Request) (Response, error) {
	fn, supported := ghSupportedActions[req.Method]
	if !supported {
		return Response{}, errs.ErrActionNotSupported(req.Method)
	}

	return fn(g, ctx, req)
}

func (g *Github) updateOrCreateIssue(ctx context.Context, req Request) (Response, error) {
	if req.TaskID == "" {
		id, err := g.issue(ctx, http.MethodPost, "", req.Vars)
		if err != nil {
			return Response{}, fmt.Errorf("create task: %w", err)
		}

		return Response{TaskID: id}, nil
	}

	if _, err := g.issue(ctx, http.MethodPatch, req.TaskID, req.Vars); err != nil {
		return Response{}, fmt.Errorf("update task: %w", err)
	}

	return Response{TaskID: req.TaskID}, nil
}

func (g *Github) issue(ctx context.Context, method, id string, vars lib.Vars) (respID string, err error) {
	bts, err := json.Marshal(map[string]interface{}{
		"title":     vars.Get("title"),
		"body":      vars.Get("body"),
		"assignees": vars.List("assignees"),
		"milestone": vars.Get("milestone"),
	})
	if err != nil {
		return "", fmt.Errorf("marshal request body: %w", err)
	}

	url := fmt.Sprintf("%s/repos/%s/%s/issues", g.baseURL, g.repo.Owner, g.repo.Name)

	if id != "" {
		url += "/" + id
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(bts))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	resp, err := g.cl.Do(req)
	if err != nil {
		return "", fmt.Errorf("do request: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", g.handleUnexpectedStatus(resp)
	}

	var respBody struct {
		ID int64 `json:"id"`
	}

	if err = json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return "", fmt.Errorf("unmarshal created issue's id: %w", err)
	}

	return strconv.FormatInt(respBody.ID, 10), nil
}

// Subscribe sends a request to github for webhook and sets a handler for that webhook.
func (g *Github) Subscribe(ctx context.Context, req SubscribeReq) (SubscribeResp, error) {
	var hookReq struct {
		Config struct {
			URL         string `json:"url"`
			ContentType string `json:"content_type"`
			InsecureSSL string `json:"insecure_ssl"`
		} `json:"config"`
		Events []string `json:"events"`
		Active bool     `json:"active"`
	}

	hookReq.Events = req.Vars.List("events")
	hookReq.Active = true
	hookReq.Config.URL = req.WebhookURL
	hookReq.Config.ContentType = "json"

	bts, err := json.Marshal(hookReq)
	if err != nil {
		return SubscribeResp{}, fmt.Errorf("marshal webhook request: %w", err)
	}

	url := fmt.Sprintf("%s/repos/%s/%s/hooks", g.baseURL, g.repo.Owner, g.repo.Name)

	httpreq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bts))
	if err != nil {
		return SubscribeResp{}, fmt.Errorf("build http request: %w", err)
	}

	resp, err := g.cl.Do(httpreq)
	if err != nil {
		return SubscribeResp{}, fmt.Errorf("do request: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return SubscribeResp{}, g.handleUnexpectedStatus(resp)
	}

	var respBody struct {
		ID int64 `json:"id"`
	}

	if err = json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return SubscribeResp{}, fmt.Errorf("unmarshal created issue's id: %w", err)
	}

	return SubscribeResp{TrackerRef: strconv.FormatInt(respBody.ID, 10)}, nil
}

func (g *Github) handleUnexpectedStatus(resp *http.Response) error {
	rerr := errs.ErrGithubAPI{ResponseStatus: resp.StatusCode}

	if err := json.NewDecoder(resp.Body).Decode(&rerr); err != nil {
		g.l.Printf("[WARN] github API responded with status %d, failed to decode response body: %v", resp.StatusCode, err)
		return rerr
	}

	return rerr
}

// Unsubscribe removes the webhook from github and removes the handler for that webhook.
func (g *Github) Unsubscribe(ctx context.Context, req UnsubscribeReq) error {
	url := fmt.Sprintf("%s/repos/%s/%s/hooks/%s", g.baseURL, g.repo.Owner, g.repo.Name, req.TrackerRef)
	httpreq, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("build http request: %w", err)
	}

	resp, err := g.cl.Do(httpreq)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return errs.ErrNotFound
	}

	if resp.StatusCode != http.StatusNoContent {
		return g.handleUnexpectedStatus(resp)
	}

	return nil
}

// HandleWebhook handles webhooks from github.
func (g *Github) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var ghUpdate struct {
		Action string `json:"action"`
		Issue  struct {
			ID    int64  `json:"number"`
			Title string `json:"title"`
			Body  string `json:"body"`
			URL   string `json:"url"`
		} `json:"issue"`
	}

	if err := json.NewDecoder(r.Body).Decode(&ghUpdate); err != nil {
		g.l.Printf("[WARN] failed to parse github request webhook: %v", err)
		return
	}

	upd := store.Update{
		URL: ghUpdate.Issue.URL,
		ReceivedFrom: store.Locator{
			Tracker: g.name,
			ID:      strconv.FormatInt(ghUpdate.Issue.ID, 10),
		},
		Content: store.Content{
			Body:   ghUpdate.Issue.Body,
			Title:  ghUpdate.Issue.Title,
			Fields: nil, // todo
		},
	}

	g.l.Printf("[DEBUG] received webhook update: %+v", upd)

	if g.handler == nil {
		g.l.Printf("[WARN] no handler is set, but update received")
		return
	}
	g.handler.Handle(ctx, upd)
}

// Listen does nothing and waits until the context is dead.
func (g *Github) Listen(ctx context.Context, h Handler) error {
	g.handler = h
	<-ctx.Done()
	return ctx.Err()
}
