package tracker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/hashicorp/go-multierror"
	"golang.org/x/oauth2"
)

// Github implements Interface over Github issues.
type Github struct {
	webhooks map[string]string // map[internalID]externalID

	baseURL string

	GithubProps
	githubConn
}

// GithubProps describes parameters
// needed to initialize Github driver.
type GithubProps struct {
	Log     *log.Logger
	Client  *http.Client
	Webhook WebhookProps
	Tracker Props
}

// NewGithub makes new instance of Github.
func NewGithub(props GithubProps) (*Github, error) {
	res := &Github{webhooks: map[string]string{}, GithubProps: props, baseURL: "https://api.github.com"}
	if err := res.githubConn.parse(props.Tracker.Variables); err != nil {
		return nil, fmt.Errorf("parse configuration: %w", err)
	}

	ts := oauth2.ReuseTokenSource(nil,
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: res.AccessToken}),
	)
	tr := &oauth2.Transport{Source: ts, Base: props.Client.Transport}
	res.Client.Transport = tr

	return res, nil
}

// Call multiplexes requests via the provided Request.Method.
func (g *Github) Call(ctx context.Context, call Request) (Response, error) {
	switch call.Method {
	case "update_task":
		if call.TicketID == "" {
			id, err := g.createIssue(ctx, call.Vars)
			if err != nil {
				return Response{}, fmt.Errorf("create task: %w", err)
			}

			return Response{ID: id}, nil
		}

		if err := g.updateIssue(ctx, call.TicketID, call.Vars); err != nil {
			return Response{}, fmt.Errorf("update task: %w", err)
		}
	default:
		return Response{}, ErrUnsupportedMethod(call.Method)
	}
	return Response{}, nil
}

// Close stops all webhooks for github tracker.
func (g *Github) Close(ctx context.Context) error {
	merr := &multierror.Error{}
	for _, externalID := range g.webhooks {
		req, err := http.NewRequestWithContext(
			ctx,
			http.MethodDelete,
			fmt.Sprintf("%s/repos/%s/%s/hooks/%s", g.baseURL, g.Owner, g.Name, externalID),
			nil,
		)
		if err != nil {
			merr = multierror.Append(merr, fmt.Errorf("create request to delete webhook %s: %w", externalID, err))
			continue
		}

		resp, err := g.Client.Do(req)
		if err != nil {
			merr = multierror.Append(merr, fmt.Errorf("do request: %w", err))
			continue
		}

		if resp.StatusCode != http.StatusNoContent {
			rerr := ErrUnexpectedStatus{ResponseStatus: resp.StatusCode}
			if rerr.ResponseBody, err = io.ReadAll(resp.Body); err != nil {
				g.Log.Printf("[WARN] failed to read github delete webhook response body for status %d",
					resp.StatusCode)
				continue
			}
		}
	}
	if err := merr.ErrorOrNil(); err != nil {
		return fmt.Errorf("close github tracker %s: %w", g.Tracker.Name, err)
	}
	return nil
}

// SetUpTrigger sends a request to github for webhook and sets a handler for that webhook.
func (g *Github) SetUpTrigger(ctx context.Context, vars Vars, cb Callback) error {
	whURL := g.Webhook.newWebHook(g.whHandler(cb))

	bts, err := json.Marshal(g.parseHookReq(whURL, vars))
	if err != nil {
		return fmt.Errorf("marshal hook request: %w", err)
	}

	url := fmt.Sprintf("%s/repos/%s/%s/hooks", g.baseURL, g.Owner, g.Name)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bts))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := g.Client.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		rerr := ErrUnexpectedStatus{RequestBody: bts, ResponseStatus: resp.StatusCode}
		if rerr.ResponseBody, err = io.ReadAll(resp.Body); err != nil {
			g.Log.Printf("[WARN] failed to read github create webhook response body for status %d",
				resp.StatusCode)
			return rerr
		}
		return rerr
	}
	var respBody struct {
		ID string `json:"id"`
	}

	if err = json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return fmt.Errorf("unmarshal created issue's id: %w", err)
	}

	g.webhooks[whURL] = respBody.ID
	return nil
}

func (g *Github) createIssue(ctx context.Context, vars Vars) (id string, err error) {
	return g.issue(ctx, http.MethodPost, "", vars)
}

func (g *Github) updateIssue(ctx context.Context, id string, vars Vars) error {
	_, err := g.issue(ctx, http.MethodPatch, id, vars)
	return err
}

func (g *Github) issue(ctx context.Context, method, id string, vars Vars) (respID string, err error) {
	bts, err := json.Marshal(g.parseIssueReq(vars))
	if err != nil {
		return "", fmt.Errorf("marshal request body: %w", err)
	}

	url := fmt.Sprintf("%s/repos/%s/%s/issues", g.baseURL, g.Owner, g.Name)

	if id != "" {
		url += "/" + id
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(bts))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	resp, err := g.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("do request: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		rerr := ErrUnexpectedStatus{RequestBody: bts, ResponseStatus: resp.StatusCode}
		if rerr.ResponseBody, err = io.ReadAll(resp.Body); err != nil {
			g.Log.Printf("[WARN] failed to read github create issue response body for status %d",
				resp.StatusCode)
			return "", rerr
		}
		return "", rerr
	}

	var respBody struct {
		ID string `json:"id"`
	}

	if err = json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return "", fmt.Errorf("unmarshal created issue's id: %w", err)
	}

	return "", nil
}

type ghHookReq struct {
	Config struct {
		URL         string `json:"url"`
		ContentType string `json:"content_type"`
		Secret      string `json:"secret"` // not used yet
		InsecureSSL string `json:"insecure_ssl"`
		Token       string `json:"token"`  // not used yet
		Digest      string `json:"digest"` // not used yet
	} `json:"config"`
	Events []string `json:"events"`
	Active bool     `json:"active"`
}

func (g *Github) parseHookReq(url string, vars Vars) ghHookReq {
	r := ghHookReq{}
	r.Events = vars.List("events")
	r.Active = true
	r.Config.URL = url
	r.Config.ContentType = "json"
	return r
}

type ghIssueReq struct {
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	Assignees []string `json:"assignees"`
	Milestone string   `json:"milestone"`
}

func (g *Github) parseIssueReq(vars Vars) ghIssueReq {
	r := ghIssueReq{}
	r.Title = vars.Get("title")
	r.Body = vars.Get("body")
	r.Milestone = vars.Get("milestone")
	r.Assignees = vars.List("assignees")
	return r
}

func (g *Github) whHandler(cb Callback) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var resp struct {
			Action string `json:"action"`
			Issue  struct {
				ID          string `json:"number"`
				Title       string `json:"title"`
				Description string `json:"description"`
				URL         string `json:"url"`
			} `json:"issue"`
		}

		if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
			g.Log.Printf("[WARN] failed to parse github request on %s webhook: %v", g.Tracker.Name, err)
			return
		}

		upd := store.Update{
			TrackerTaskID: resp.Issue.ID,
			Body:          resp.Issue.Description,
			Title:         resp.Issue.Title,
			URL:           resp.Issue.URL,
		}

		if err := cb.Do(r.Context(), upd); err != nil {
			g.Log.Printf("[WARN] callback on github/%s (update %+v) returned error: %v", g.Tracker.Name, upd, err)
			return
		}
	}
}

type githubConn struct {
	Owner       string
	Name        string
	AccessToken string
}

func (r *githubConn) parse(vars Vars) error {
	var ok bool

	if r.Name, ok = vars["owner"]; !ok {
		return ErrInvalidConf("repository owner is not present")
	}

	if r.Name, ok = vars["name"]; !ok {
		return ErrInvalidConf("repository name is not present")
	}

	if r.AccessToken, ok = vars["access_token"]; !ok {
		return ErrInvalidConf("client id is not present")
	}

	return nil
}
