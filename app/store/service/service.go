package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"text/template"
	"time"

	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/app/store/engine"
	"github.com/cappuccinotm/dastracker/app/tracker"
)

// Service wraps engine.Interface with methods
// common for each engine implementation.
type Service struct {
	eng      engine.Interface
	trackers map[string]tracker.Interface
	jobs     map[string]Job
}

// NewService makes new instance of Service.
func NewService(eng engine.Interface, trackers map[string]tracker.Interface) *Service {
	return &Service{eng: eng, trackers: trackers}
}

// Trigger describes parameters for trigger.
type Trigger struct {
	Tracker string
	Job     string
	Vars    tracker.Vars
}

// InitTrigger initializes triggers in trackers.
func (s *Service) InitTrigger(ctx context.Context, trigger Trigger) error {
	err := s.trackers[trigger.Tracker].SetUpTrigger(ctx, trigger.Vars,
		tracker.CallbackFunc(func(ctx context.Context, update store.Update) error {
			if err := s.onTrigger(ctx, trigger.Tracker, trigger.Job, update); err != nil {
				return fmt.Errorf("handle error: %w", err)
			}

			return nil
		}))
	if err != nil {
		return fmt.Errorf("set up trigger: %w", err)
	}

	return nil
}

// Job describes a single job.
type Job struct {
	Name    string
	Actions []Action
}

type Action struct {
	Tracker string
	Method  string
	Vars    map[string]*template.Template
}

type tmpl struct {
	Old       store.Ticket
	Update    store.Update
	Timestamp time.Time
}

func (s *Service) onTrigger(ctx context.Context, trackerName, jobName string, update store.Update) error {
	oldTicket, err := s.eng.Get(ctx, trackerName, update.TrackerTaskID)
	switch {
	case errors.Is(err, engine.ErrNotFound):
		oldTicket = store.Ticket{Fields: map[string]string{}}
	case err != nil:
		return fmt.Errorf("get ticket %s/%s: %w", trackerName, update.TrackerTaskID, err)
	}

	vals := tmpl{Old: oldTicket, Update: update, Timestamp: time.Now()}

	// todo case non exist
	job := s.jobs[jobName]

	ticket := oldTicket

	for _, action := range job.Actions {
		varVals := map[string]string{}
		for name, t := range action.Vars {
			buf := &bytes.Buffer{}
			if err := t.Execute(buf, vals); err != nil {
				return fmt.Errorf("execute variable %s: %w", name, err)
			}
			varVals[name] = buf.String()
		}

		req := tracker.Request{Method: action.Method, Vars: varVals}

		if ticketID, ok := oldTicket.TrackerIDs[action.Tracker]; ok {
			req.TicketID = ticketID
		}

		resp, err := s.trackers[action.Tracker].Call(ctx, req)
		if err != nil {
			return fmt.Errorf("call %s/%s: %w", action.Tracker, action.Method, err)
		}

		if resp.ID != "" {
			ticket.TrackerIDs[action.Tracker] = resp.ID
		}
	}

	ticket.Body = update.Body
	ticket.Title = update.Title
	ticket.Fields = update.Fields

	if ticket.ID != "" {
		if err := s.eng.Update(ctx, ticket); err != nil {
			return fmt.Errorf("update ticket: %w", err)
		}

		return nil
	}

	if ticket.ID, err = s.eng.Create(ctx, ticket); err != nil {
		return fmt.Errorf("create ticket: %w", err)
	}

	return nil
}
