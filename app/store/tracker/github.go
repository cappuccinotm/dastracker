package tracker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"golang.org/x/oauth2"
)

// Github implements Interface over Github issues.
type Github struct {
	cl      *http.Client
	baseURL string
	log     *log.Logger

	githubConn
}

// NewGithub makes new instance of Github.
func NewGithub(cl *http.Client, confVars Vars) (*Github, error) {
	res := &Github{cl: cl, baseURL: "https://api.github.com"}
	if err := res.githubConn.parse(confVars); err != nil {
		return nil, fmt.Errorf("parse configuration: %w", err)
	}

	ts := oauth2.ReuseTokenSource(nil,
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: res.AccessToken}),
	)
	tr := &oauth2.Transport{Source: ts, Base: cl.Transport}
	res.cl.Transport = tr

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

		if err := g.updateTask(ctx, call.TicketID, call.Vars); err != nil {
			return Response{}, fmt.Errorf("update task: %w", err)
		}
	default:
		return Response{}, ErrUnsupportedMethod(call.Method)
	}
	return Response{}, nil
}

func (g *Github) SetUpTrigger(ctx context.Context, vars Vars, cb Callback) error {
	panic("implement me")
}

func (g *Github) createIssue(ctx context.Context, vars Vars) (id string, err error) {
	return g.send(ctx, http.MethodPost, "", vars)
}

func (g *Github) updateTask(ctx context.Context, id string, vars Vars) error {
	_, err := g.send(ctx, http.MethodPatch, id, vars)
	return err
}

type ghTicketRequest struct {
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	Assignees []string `json:"assignees"`
	Milestone string   `json:"milestone"`
}

func (g *Github) send(ctx context.Context, method, id string, vars Vars) (respID string, err error) {
	body := ghTicketRequest{
		Title:     vars.Get("title"),
		Body:      vars.Get("body"),
		Assignees: vars.List("assignees"),
		Milestone: vars.Get("milestone"),
	}

	bts, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request body: %w", err)
	}

	url := fmt.Sprintf("%s/repos/%s/%s/issues", g.baseURL, g.Owner, g.Name)

	if id != "" {
		url += "/" + id
	}

	req, err := http.NewRequestWithContext(
		ctx,
		method,
		url,
		bytes.NewReader(bts),
	)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	resp, err := g.cl.Do(req)
	if err != nil {
		return "", fmt.Errorf("do request: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		rerr := ErrUnexpectedStatus{RequestBody: bts, ResponseStatus: resp.StatusCode}
		if rerr.ResponseBody, err = io.ReadAll(resp.Body); err != nil {
			g.log.Printf("[WARN] failed to read github create issue response body for status %d",
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

type githubConn struct {
	Owner       string
	Name        string
	AccessToken string
}

func (c githubConn) parse(vars Vars) error {
	var ok bool

	if c.Name, ok = vars["owner"]; !ok {
		return ErrInvalidConf("repository owner is not present")
	}

	if c.Name, ok = vars["name"]; !ok {
		return ErrInvalidConf("repository name is not present")
	}

	if c.AccessToken, ok = vars["access_token"]; !ok {
		return ErrInvalidConf("client id is not present")
	}

	return nil
}
