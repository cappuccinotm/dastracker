package service

import (
	"context"
	"fmt"
	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/app/store/engine"
	"github.com/cappuccinotm/dastracker/pkg/logx"
	"github.com/gorilla/mux"
	"net/http"
)

// SubscriptionsManager manages subscriptions for task updates in trackers,
// including creation of subscriptions and recognizing particular subscription.
type SubscriptionsManager struct {
	BaseURL string
	Router  *mux.Router
	Logger  logx.Logger
	Addr    string
	Store   engine.Subscriptions
}

// Create registers a new subscription for a specific trigger.
func (m *SubscriptionsManager) Create(ctx context.Context, tracker, trigger string) (store.Subscription, error) {
	var err error
	wh := store.Subscription{TrackerName: tracker, TriggerName: trigger, BaseURL: m.BaseURL}

	if wh.ID, err = m.Store.Create(ctx, wh); err != nil {
		return store.Subscription{}, fmt.Errorf("create subscription details entry: %w", err)
	}

	return wh, nil
}

// SetTrackerRef updates the subscription by setting the remote tracker ID to the
// desired subscription.
func (m *SubscriptionsManager) SetTrackerRef(ctx context.Context, subscriptionID, trackerRef string) error {
	wh, err := m.Store.Get(ctx, subscriptionID)
	if err != nil {
		return fmt.Errorf("get subscription: %w", err)
	}

	wh.TrackerRef = trackerRef

	if err = m.Store.Update(ctx, wh); err != nil {
		return fmt.Errorf("update subscription: %w", err)
	}
	return nil
}

// Delete proxies the call to the Store.
func (m *SubscriptionsManager) Delete(ctx context.Context, subscriptionID string) error {
	return m.Store.Delete(ctx, subscriptionID)
}

// List proxies the call to the wrapped Store.
func (m *SubscriptionsManager) List(ctx context.Context, tracker string) ([]store.Subscription, error) {
	return m.Store.List(ctx, tracker)
}

// Listen starts the subscription server.
func (m *SubscriptionsManager) Listen(ctx context.Context, handler http.Handler) error {
	m.Router.Use(m.whLoaderMiddleware)
	m.Router.Handle("/{whID}", handler)

	srv := &http.Server{Addr: m.Addr, Handler: m.Router}
	go func() {
		<-ctx.Done()
		if err := srv.Shutdown(context.Background()); err != nil {
			m.Logger.Printf("[WARN] failed to shutdown subscription server: %v", err)
		}
	}()
	if err := srv.ListenAndServe(); err != nil {
		return err
	}
	return nil
}

func (m *SubscriptionsManager) whLoaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var v map[string]string

		if v = mux.Vars(r); v == nil || v["whID"] == "" {
			m.Logger.Printf("[WARN] didn't find tracker name or subscription ID among the request variables in URL %q",
				r.URL)
			return
		}

		wh, err := m.Store.Get(ctx, v["whID"])
		if err != nil {
			m.Logger.Printf("[WARN] failed to get subscription with ID %s, requested at URL %q: %v",
				v["whID"], r.URL, err)
			return
		}

		next.ServeHTTP(w, r.WithContext(store.PutSubscription(ctx, wh)))
	})
}
