package webhook

import (
	"context"
	"fmt"
	"net/http"

	"github.com/cappuccinotm/dastracker/app/errs"
	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/app/store/engine"
	"github.com/cappuccinotm/dastracker/pkg/logx"
	"github.com/gorilla/mux"
)

//go:generate rm -f interface_mock.go
//go:generate moq -out interface_mock.go -fmt goimports . Interface

// Interface defines methods to create and update webhooks.
type Interface interface {
	Create(ctx context.Context, tracker, trigger string) (store.Webhook, error)
	SetTrackerID(ctx context.Context, webhookID, trackerID string) error
	Register(name string, handler http.Handler) error
	// List returns all webhooks if tracker = ""
	List(ctx context.Context, tracker string) ([]store.Webhook, error)
	Delete(ctx context.Context, webhookID string) error
	Listen(ctx context.Context) error
}

// Manager creates a webhook with specified handler on it and
// returns the exact URL of the created webhook.
type Manager struct {
	baseURL string
	r       *mux.Router
	l       logx.Logger
	addr    string

	registeredHandlers []string
	store              engine.Webhooks
}

// NewManager makes new instance of Manager.
func NewManager(baseURL, addr string, store engine.Webhooks, l logx.Logger) *Manager {
	svc := &Manager{baseURL: baseURL, r: mux.NewRouter(), store: store, l: l, addr: addr}

	// todo load webhooks from storage

	svc.r.Use(svc.whLoaderMiddleware)
	return svc
}

// Register registers a new tracker handler.
func (m *Manager) Register(name string, handler http.Handler) error {
	if m.registered(name) {
		return errs.ErrTrackerRegistered(name)
	}
	m.registeredHandlers = append(m.registeredHandlers, name)
	m.r.Handle(fmt.Sprintf("/%s/{whID}", name), handler)
	return nil
}

// Create registers a new webhook for a specific trigger.
func (m *Manager) Create(ctx context.Context, tracker, trigger string) (store.Webhook, error) {
	if !m.registered(tracker) {
		return store.Webhook{}, errs.ErrTrackerNotRegistered(tracker)
	}

	var err error
	wh := store.Webhook{TrackerName: tracker, TriggerName: trigger, BaseURL: m.baseURL}

	if wh.ID, err = m.store.Create(ctx, wh); err != nil {
		return store.Webhook{}, fmt.Errorf("create webhook details entry: %w", err)
	}

	return wh, nil
}

// SetTrackerID updates the webhook by setting the remote tracker ID to the
// desired webhook.
func (m *Manager) SetTrackerID(ctx context.Context, webhookID, trackerID string) error {
	wh, err := m.store.Get(ctx, webhookID)
	if err != nil {
		return fmt.Errorf("get webhook: %w", err)
	}

	wh.TrackerRef = trackerID

	if err = m.store.Update(ctx, wh); err != nil {
		return fmt.Errorf("update webhook: %w", err)
	}
	return nil
}

func (m *Manager) whLoaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var v map[string]string

		if v = mux.Vars(r); v == nil || v["whID"] == "" {
			m.l.Printf("[WARN] didn't find webhook ID among the request variables in URL %q",
				r.URL)
			return
		}

		wh, err := m.store.Get(ctx, v["whID"])
		if err != nil {
			m.l.Printf("[WARN] failed to get webhook with ID %s, requested at URL %q: %v",
				v["whID"], r.URL, err)
			return
		}

		next.ServeHTTP(w, r.WithContext(PutWebhook(ctx, wh)))
	})
}

// Delete proxies the call to the store.
func (m *Manager) Delete(ctx context.Context, webhookID string) error {
	return m.store.Delete(ctx, webhookID)
}

// List proxies the call to the wrapped store.
func (m *Manager) List(ctx context.Context, tracker string) ([]store.Webhook, error) {
	return m.store.List(ctx, tracker)
}

// Listen starts the webhook server.
func (m *Manager) Listen(ctx context.Context) error {
	srv := &http.Server{Addr: m.addr, Handler: m.r}
	if err := srv.ListenAndServe(); err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		if err := srv.Shutdown(context.Background()); err != nil {
			m.l.Printf("[WARN] failed to shutdown webhook server: %v", err)
		}
	}()
	return nil
}

func (m *Manager) registered(name string) bool {
	for _, h := range m.registeredHandlers {
		if h == name {
			return true
		}
	}
	return false
}
