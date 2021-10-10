package cmd

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/rpc"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/cappuccinotm/dastracker/app/store/engine"
	"github.com/cappuccinotm/dastracker/app/store/service"
	"github.com/cappuccinotm/dastracker/app/tracker"
	"github.com/gorilla/mux"
	bolt "go.etcd.io/bbolt"
	"gopkg.in/yaml.v3"
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
}

// Execute runs the command
func (r Run) Execute(_ []string) error {
	f, err := os.Open(r.ConfLocation)
	if err != nil {
		return fmt.Errorf("open config file at location %s: %w", r.ConfLocation, err)
	}

	var conf Config

	if err := yaml.NewDecoder(f).Decode(&conf); err != nil {
		return fmt.Errorf("decode config: %w", err)
	}

	whRouter := mux.NewRouter()

	trackers, err := r.initializeTrackers(conf, whRouter)
	if err != nil {
		return fmt.Errorf("initialize trackers: %w", err)
	}

	eng, err := r.makeDataStore()
	if err != nil {
		return fmt.Errorf("make data engine: %w", err)
	}

	parsedJobs, err := r.parseJobs(conf.Jobs)
	if err != nil {
		return fmt.Errorf("parse jobs: %w", err)
	}

	svc := service.NewService(eng, trackers, parsedJobs)

	if err = r.initTriggers(svc, conf); err != nil {
		return fmt.Errorf("initialzie triggers: %w", err)
	}

	err = whRouter.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		t, err := route.GetPathTemplate()
		if err != nil {
			return err
		}
		log.Printf("[INFO] route %s", t)
		return nil
	})
	if err != nil {
		log.Printf("[WARN] failed to walk through webhook routes: %v", err)
	}

	log.Printf("[INFO] starting listening for webhooks at %s", r.Webhook.Addr)

	if err := http.ListenAndServe(r.Webhook.Addr, whRouter); err != nil {
		log.Printf("[WARN] listener stopped, reason: %v", err)
	}

	return nil
}

func executeVars(varTmpls map[string]string) (map[string]string, error) {
	res := map[string]string{}
	for name, val := range varTmpls {
		tmpl, err := template.New("").Funcs(funcs).Parse(val)
		if err != nil {
			return nil, fmt.Errorf("parse template: %w", err)
		}

		buf := &bytes.Buffer{}
		if err = tmpl.Execute(buf, nil); err != nil {
			return nil, fmt.Errorf("execute template: %w", err)
		}

		res[name] = buf.String()
	}
	return res, nil
}

func (r Run) initializeTrackers(conf Config, whMux *mux.Router) (map[string]tracker.Interface, error) {
	res := map[string]tracker.Interface{}

	for _, trackerConf := range conf.Trackers {
		vars, err := executeVars(trackerConf.Vars)
		if err != nil {
			return nil, fmt.Errorf("tracker %s execute vars: %w", trackerConf.Name, err)
		}

		switch trackerConf.Driver {
		case "github":
			subrouter := whMux.PathPrefix("/" + trackerConf.Name).Subrouter()

			res[trackerConf.Name], err = tracker.NewGithub(tracker.GithubParams{
				Log:     log.Default(),
				Client:  &http.Client{Timeout: 5 * time.Second},
				Webhook: tracker.WebhookProps{Mux: subrouter, BaseURL: r.Webhook.BaseURL + "/" + trackerConf.Name},
				Tracker: tracker.Props{Name: trackerConf.Name, Variables: vars},
			})
			if err != nil {
				return nil, fmt.Errorf("github tracker %s: %w", trackerConf.Name, err)
			}
		case "rpc":
			rcl, err := tracker.NewRPC(tracker.RPCParams{
				Dialer: tracker.RPCDialerFunc(func(network, address string) (tracker.RPCClient, error) {
					return rpc.Dial(network, address)
				}),
				Logger:  log.Default(),
				Tracker: tracker.Props{Name: trackerConf.Name, Variables: vars},
			})
			if err != nil {
				return nil, fmt.Errorf("rpc tracker %s: %w", trackerConf.Name, err)
			}
			res[trackerConf.Name] = rcl
		default:
			return nil, fmt.Errorf("unsupported driver %s for %s", trackerConf.Driver, trackerConf.Name)
		}
	}

	return res, nil
}

func (r Run) makeDataStore() (res engine.Interface, err error) {
	switch r.Store.Type {
	case "bolt":
		boltDB, err := engine.NewBolt(r.Store.Bolt.Path, bolt.Options{Timeout: r.Store.Bolt.Timeout})
		if err != nil {
			return nil, fmt.Errorf("failed to initialize bolt storage at %s: %w", r.Store.Bolt.Path, err)
		}
		res = boltDB
	default:
		return nil, fmt.Errorf("unsupported storage type %s", r.Store.Type)
	}

	return res, nil
}

func (r Run) initTriggers(svc *service.Service, conf Config) error {
	for _, job := range conf.Jobs {
		trigger := service.Trigger{
			Tracker: job.On.TrackerName,
			Job:     job.Name,
			Vars:    job.On.Vars,
		}
		err := svc.InitTrigger(context.Background(), trigger)
		if err != nil {
			return fmt.Errorf("initialize trigger %v: %w", trigger, err)
		}
	}

	return nil
}

func (r Run) parseJobs(jobs []Job) (map[string]service.Job, error) {
	res := map[string]service.Job{}

	for _, jobCfg := range jobs {
		job := service.Job{Name: jobCfg.Name, Actions: make([]service.Action, len(jobCfg.Actions))}
		for actIdx, actCfg := range jobCfg.Actions {
			act := service.Action{Vars: map[string]*template.Template{}}
			act.Tracker, act.Method = parseMethodName(actCfg.Method)

			for vname, vval := range actCfg.Vars {
				tmpl, err := template.New("").Funcs(funcs).Parse(vval)
				if err != nil {
					return nil, fmt.Errorf("parse var %s of action %s in job %s: %w", vname, act.Method, job.Name, err)
				}

				act.Vars[vname] = tmpl
			}
			job.Actions[actIdx] = act
		}
		res[jobCfg.Name] = job
	}

	return res, nil
}

func parseMethodName(method string) (string, string) {
	res := strings.Split(method, "/")
	return res[0], res[1]
}
