package static

import (
	"io"
	"os"
	"path"
	"testing"

	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/cappuccinotm/dastracker/lib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testCfg = `trackers:
  - name: gh_dastracker
    driver: github
    with:
      owner: cappuccinotm
      name: dastracker
      user: '{{ env "GITHUB_USER" }}'
      access_token: '{{ env "GITHUB_ACCESS_TOKEN" }}'

  - name: customrpc
    driver: rpc
    with:
      address: '{{ env "CUSTOM_RPC_ADDRESS" }}'

# fixme what if there are the same triggers for jobs?

triggers:
  - name: gh_task_updated
    in: gh_dastracker
    with:
      events: "issue"

jobs:
  - name: print task update if task is received
    on: gh_task_updated
    do:
      - action: customrpc/Print
        detached: true
        with:
          message: 'Task "{{.Update.Title}}" has been updated and printed to the terminal.'
`

func TestNewStatic(t *testing.T) {
	t.Run("valid configuration", func(t *testing.T) {
		loc, err := os.MkdirTemp("", "test_dastracker")
		require.NoError(t, err, "failed to make temp dir")

		f, err := os.Create(path.Join(loc, "config.yml"))
		require.NoError(t, err)
		defer f.Close()

		_, err = io.WriteString(f, testCfg)
		require.NoError(t, err)

		svc, err := NewStatic(path.Join(loc, "config.yml"))
		require.NoError(t, err)

		expectedSvc := &Static{
			config: config{
				Trackers: []store.Tracker{
					{
						Name:   "gh_dastracker",
						Driver: "github",
						With: lib.Vars{
							"owner":        "cappuccinotm",
							"name":         "dastracker",
							"user":         `{{ env "GITHUB_USER" }}`,
							"access_token": `{{ env "GITHUB_ACCESS_TOKEN" }}`,
						},
					},
					{
						Name:   "customrpc",
						Driver: "rpc",
						With:   lib.Vars{"address": `{{ env "CUSTOM_RPC_ADDRESS" }}`},
					},
				},
				Triggers: []store.Trigger{{
					Name:    "gh_task_updated",
					Tracker: "gh_dastracker",
					With:    lib.Vars{"events": "issue"},
				}},
				Jobs: []store.Job{{
					Name:        "print task update if task is received",
					TriggerName: "gh_task_updated",
					Actions: []store.Step{store.Action{
						Name:     "customrpc/Print",
						Detached: true,
						With: lib.Vars{
							"message": `Task "{{.Update.Title}}" has been updated and printed to the terminal.`,
						},
					}},
				}},
			},
			subscriptions: map[string][]store.Job{
				"gh_task_updated": {
					{
						Name:        "print task update if task is received",
						TriggerName: "gh_task_updated",
						Actions: []store.Step{store.Action{
							Name:     "customrpc/Print",
							Detached: true,
							With: lib.Vars{
								"message": `Task "{{.Update.Title}}" has been updated and printed to the terminal.`,
							},
						}},
					},
				},
			},
		}
		assert.Equal(t, expectedSvc, svc)
	})
}
