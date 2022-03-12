package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/cappuccinotm/dastracker/app/flow"
	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/app/store/engine"
	boltEngs "github.com/cappuccinotm/dastracker/app/store/engine/bolt"
	"github.com/cappuccinotm/dastracker/app/store/service"
	"github.com/cappuccinotm/dastracker/app/tracker"
	"github.com/cappuccinotm/dastracker/app/webhook"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/sync/errgroup"
)

// Run starts a tracker listener.
type Run struct {
	ConfLocation string `short:"c" long:"config_location" env:"CONFIG_LOCATION" description:"location of the configuration file"`
	Store        struct {
		Type string `long:"type" env:"TYPE" choice:"bolt" description:"type of storage"`
		Bolt struct {
			Path    string        `long:"path" env:"PATH" default:"./var" description:"parent dir for bolt files"`
			Timeout time.Duration `long:"timeout" env:"TIMEOUT" default:"30s" description:"bolt timeout"`
		} `group:"bolt" namespace:"bolt" env-namespace:"BOLT"`
	} `group:"store" namespace:"store" env-namespace:"STORE"`
	Webhook struct {
		BaseURL string `long:"base_url" env:"BASE_URL" description:"base url for webhooks"`
		Addr    string `long:"addr" env:"ADDR" description:"local address to listen"`
	} `group:"webhook" namespace:"webhook" env-namespace:"WEBHOOK"`
	UpdateTimeout time.Duration `long:"update_timeout" env:"UPDATE_TIMEOUT" description:"amount of time per processing single update"`
	CommonOpts
}

// Execute runs the command
func (r Run) Execute(_ []string) error {
	flowStore, err := r.prepareFlowStore()
	if err != nil {
		return fmt.Errorf("prepare flow storage: %w", err)
	}

	ticketsStore, err := r.prepareTicketsStore()
	if err != nil {
		return fmt.Errorf("initialize tickets store: %w", err)
	}

	webhooksManager, err := r.prepareWebhookManager()
	if err != nil {
		return fmt.Errorf("initialize webhooks manager: %w", err)
	}

	trackers, err := r.prepareTrackers(flowStore, webhooksManager)
	if err != nil {
		return fmt.Errorf("prepare trackers: %w", err)
	}

	actor := &service.Actor{
		Trackers:      trackers,
		TicketsStore:  ticketsStore,
		Flow:          flowStore,
		Log:           r.Logger.Sub("[actor]: "),
		UpdateTimeout: r.UpdateTimeout,
	}

	eg, ctx := errgroup.WithContext(context.Background())
	eg.Go(func() error {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
		<-stop
		r.Logger.Printf("[WARN] interrupt signal")
		return fmt.Errorf("interrupted")
	})
	eg.Go(func() error {
		if err := actor.Listen(ctx); err != nil {
			return fmt.Errorf("actor stopped listening, reason: %w", err)
		}
		return nil
	})
	eg.Go(func() error {
		if err := webhooksManager.Listen(ctx); err != nil {
			return fmt.Errorf("webhook server stopped running, reason: %w", err)
		}
		return nil
	})

	if err = eg.Wait(); err != nil {
		return err
	}

	return nil
}

func (r Run) prepareFlowStore() (flow.Interface, error) {
	return flow.NewStatic(r.ConfLocation)
}

func (r Run) prepareWebhookManager() (webhook.Interface, error) {
	webhooksStore, err := r.prepareWebhooksStore()
	if err != nil {
		return nil, fmt.Errorf("initialize webhooks store: %w", err)
	}

	return webhook.NewManager(r.Webhook.BaseURL, r.Webhook.Addr, webhooksStore, r.Logger.Sub("[webhook_manager]: ")), nil
}

func (r Run) prepareTrackers(flowStore flow.Interface, whm webhook.Interface) (map[string]tracker.Interface, error) {
	trackers, err := flowStore.ListTrackers(context.Background())
	if err != nil {
		return nil, fmt.Errorf("get trackers configs: %w", err)
	}

	res := map[string]tracker.Interface{}
	for _, trk := range trackers {
		if trk.With, err = store.Evaluate(trk.With, store.Update{}); err != nil {
			return nil, fmt.Errorf("evaluate variables for tracker %q: %w", trk.Name, err)
		}

		sublogger := r.Logger.Sub(fmt.Sprintf("[tracker|%s]: ", trk.Name))

		switch trk.Driver {
		case "rpc":
			if res[trk.Name], err = tracker.NewJSONRPC(trk.Name, sublogger, whm, trk.With); err != nil {
				return nil, fmt.Errorf("initialize jsonrpc tracker %s: %w", trk.Name, err)
			}
		case "github":
			if res[trk.Name], err = tracker.NewGithub(tracker.GithubParams{
				Name:           trk.Name,
				WebhookManager: whm,
				Vars:           trk.With,
				Client:         &http.Client{Timeout: 5 * time.Second},
				Logger:         sublogger,
			}); err != nil {
				return nil, fmt.Errorf("initialize github tracker %s: %w", trk.Name, err)
			}
		default:
			return nil, fmt.Errorf("unsupported driver: %s", trk.Driver)
		}
	}

	return res, nil
}

func (r Run) prepareWebhooksStore() (engine.Webhooks, error) {
	switch r.Store.Type {
	case "bolt":
		webhooks, err := boltEngs.NewWebhook(path.Join(r.Store.Bolt.Path, "webhooks.db"), bolt.Options{Timeout: r.Store.Bolt.Timeout})
		if err != nil {
			return nil, fmt.Errorf("initialize bolt store: %w", err)
		}
		return webhooks, nil
	default:
		return nil, fmt.Errorf("unsupported store type: %s", r.Store.Type)
	}
}

func (r Run) prepareTicketsStore() (engine.Tickets, error) {
	switch r.Store.Type {
	case "bolt":
		tickets, err := boltEngs.NewTickets(path.Join(r.Store.Bolt.Path, "tickets.db"), bolt.Options{Timeout: r.Store.Bolt.Timeout})
		if err != nil {
			return nil, fmt.Errorf("initialize bolt store: %w", err)
		}
		return tickets, nil
	default:
		return nil, fmt.Errorf("unsupported store type: %s", r.Store.Type)
	}
}
