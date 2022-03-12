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
	"github.com/cappuccinotm/dastracker/app/webhook"
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
	whm     webhook.Interface
	repo    struct{ Owner, Name string }
	cl      *http.Client
	handler Handler
}

// GithubParams describes parameters to initialize Github.
type GithubParams struct {
	Name           string
	WebhookManager webhook.Interface
	Vars           lib.Vars
	Client         *http.Client
	Logger         logx.Logger
}

// NewGithub makes new instance of Github tracker.
func NewGithub(params GithubParams) (*Github, error) {
	svc := &Github{
		baseURL: githubAPIURL,
		name:    params.Name,
		l:       params.Logger,
		whm:     params.WebhookManager,
		cl:      params.Client,
	}

	if err := svc.whm.Register(svc.name, http.HandlerFunc(svc.whHandler)); err != nil {
		return nil, fmt.Errorf("register webhooks handler: %w", err)
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
	ghID := req.Ticket.TrackerIDs.Get(g.name)
	if ghID == "" {
		id, err := g.issue(ctx, http.MethodPost, "", req.Vars)
		if err != nil {
			return Response{}, fmt.Errorf("create task: %w", err)
		}

		return Response{TaskID: id}, nil
	}

	if _, err := g.issue(ctx, http.MethodPatch, ghID, req.Vars); err != nil {
		return Response{}, fmt.Errorf("update task: %w", err)
	}

	return Response{TaskID: ghID}, nil
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
func (g *Github) Subscribe(ctx context.Context, req SubscribeReq) error {
	wh, err := g.whm.Create(ctx, g.name, req.TriggerName)
	if err != nil {
		return fmt.Errorf("create webhook entry: %w", err)
	}

	whURL, err := wh.URL()
	if err != nil {
		return fmt.Errorf("get url for the webhook: %w", err)
	}

	g.l.Printf("[INFO] setting up a webhook to url %s", whURL)

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
	hookReq.Config.URL = whURL
	hookReq.Config.ContentType = "json"

	bts, err := json.Marshal(hookReq)
	if err != nil {
		return fmt.Errorf("marshal webhook request: %w", err)
	}

	url := fmt.Sprintf("%s/repos/%s/%s/hooks", g.baseURL, g.repo.Owner, g.repo.Name)

	httpreq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(bts))
	if err != nil {
		return fmt.Errorf("build http request: %w", err)
	}

	resp, err := g.cl.Do(httpreq)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return g.handleUnexpectedStatus(resp)
	}

	var respBody struct {
		ID int64 `json:"id"`
	}

	if err = json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return fmt.Errorf("unmarshal created issue's id: %w", err)
	}

	if err = g.whm.SetTrackerID(ctx, wh.ID, strconv.FormatInt(respBody.ID, 10)); err != nil {
		return fmt.Errorf("set github's webhook id %q: %w", respBody.ID, err)
	}

	return nil
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
	panic("implement me")
}

func (g *Github) whHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var ghUpdate struct {
		Action string `json:"action"`
		Issue  struct {
			ID          int    `json:"number"`
			Title       string `json:"title"`
			Description string `json:"description"`
			URL         string `json:"url"`
		} `json:"issue"`
	}

	if err := json.NewDecoder(r.Body).Decode(&ghUpdate); err != nil {
		g.l.Printf("[WARN] failed to parse github request webhook: %v", err)
		return
	}

	wh, err := webhook.GetWebhook(ctx)
	if err != nil {
		g.l.Printf("[WARN] failed to get webhook information from request: %v", err)
		return
	}

	upd := store.Update{
		TriggerName:  wh.TriggerName,
		URL:          ghUpdate.Issue.URL,
		ReceivedFrom: store.Locator{Tracker: wh.TrackerName, ID: strconv.Itoa(ghUpdate.Issue.ID)},
		Content: store.Content{
			Body:   ghUpdate.Issue.Description,
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
