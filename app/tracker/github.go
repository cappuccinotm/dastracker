package tracker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/app/util"
	"github.com/cappuccinotm/dastracker/lib"
	"github.com/hashicorp/go-multierror"
)

// Github implements Interface over Github issues.
type Github struct {
	webhooks map[string]string // map[internalID]externalID

	baseURL string

	GithubParams
	githubConn
}

// GithubParams describes parameters
// needed to initialize Github driver.
type GithubParams struct {
	Log     *log.Logger
	Client  *http.Client
	Webhook WebhookProps
	Tracker Props
}

// NewGithub makes new instance of Github.
func NewGithub(params GithubParams) (Interface, error) {
	res := &Github{webhooks: map[string]string{}, GithubParams: params, baseURL: "https://api.github.com"}
	if err := res.githubConn.parse(params.Tracker.Variables); err != nil {
		return nil, fmt.Errorf("parse configuration: %w", err)
	}

	res.Client.Transport = util.RoundTripperFunc(
		func(r *http.Request) (*http.Response, error) {
			r.SetBasicAuth(res.User, res.AccessToken)
			return http.DefaultTransport.RoundTrip(r)
		})

	log.Printf("[INFO] initialized Github tracker with params %+v", params)

	return res, nil
}

// Call multiplexes requests via the provided Request.Method.
func (g *Github) Call(ctx context.Context, call lib.Request) (lib.Response, error) {
	switch call.Method {
	case "update_task":
		if call.TicketID == "" {
			id, err := g.createIssue(ctx, call.Vars)
			if err != nil {
				return lib.Response{}, fmt.Errorf("create task: %w", err)
			}

			return lib.Response{ID: id}, nil
		}

		if err := g.updateIssue(ctx, call.TicketID, call.Vars); err != nil {
			return lib.Response{}, fmt.Errorf("update task: %w", err)
		}
	default:
		return lib.Response{}, ErrUnsupportedMethod(call.Method)
	}
	return lib.Response{}, nil
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
func (g *Github) SetUpTrigger(ctx context.Context, vars lib.Vars, cb Callback) error {
	whURL := g.Webhook.newWebHook(g.whHandler(cb))

	log.Printf("[INFO] setting up a webhook to url %s for %s github trigger", g.Tracker.Name, whURL)

	return nil

	//bts, err := json.Marshal(g.parseHookReq(whURL, vars))
	//if err != nil {
	//	return fmt.Errorf("marshal hook request: %w", err)
	//}
	//
	//url := fmt.Sprintf("%s/repos/%s/%s/hooks", g.baseURL, g.Owner, g.Name)
	//
	//req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bts))
	//if err != nil {
	//	return fmt.Errorf("create request: %w", err)
	//}
	//
	//resp, err := g.Client.Do(req)
	//if err != nil {
	//	return fmt.Errorf("do request: %w", err)
	//}
	//
	//if resp.StatusCode != http.StatusCreated {
	//	rerr := ErrUnexpectedStatus{RequestBody: bts, ResponseStatus: resp.StatusCode}
	//	if rerr.ResponseBody, err = io.ReadAll(resp.Body); err != nil {
	//		g.Logger.Printf("[WARN] failed to read github create webhook response body for status %d",
	//			resp.StatusCode)
	//		return rerr
	//	}
	//	return rerr
	//}
	//var respBody struct {
	//	ID string `json:"id"`
	//}
	//
	//if err = json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
	//	return fmt.Errorf("unmarshal created issue's id: %w", err)
	//}
	//
	//g.webhooks[whURL] = respBody.ID
	//
	//return nil
}

func (g *Github) createIssue(ctx context.Context, vars lib.Vars) (id string, err error) {
	return g.issue(ctx, http.MethodPost, "", vars)
}

func (g *Github) updateIssue(ctx context.Context, id string, vars lib.Vars) error {
	_, err := g.issue(ctx, http.MethodPatch, id, vars)
	return err
}

func (g *Github) issue(ctx context.Context, method, id string, vars lib.Vars) (respID string, err error) {
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

func (g *Github) parseHookReq(url string, vars lib.Vars) ghHookReq {
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

func (g *Github) parseIssueReq(vars lib.Vars) ghIssueReq {
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
				ID          int    `json:"number"`
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
			TrackerTaskID: strconv.Itoa(resp.Issue.ID),
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
	User        string
	AccessToken string
}

func (r *githubConn) parse(vars lib.Vars) error {
	var ok bool

	if r.Name, ok = vars["owner"]; !ok {
		return ErrInvalidConf("repository owner is not present")
	}

	if r.Name, ok = vars["name"]; !ok {
		return ErrInvalidConf("repository name is not present")
	}

	if r.User, ok = vars["user"]; !ok {
		return ErrInvalidConf("user is not present")
	}

	if r.AccessToken, ok = vars["access_token"]; !ok {
		return ErrInvalidConf("client id is not present")
	}

	return nil
}
