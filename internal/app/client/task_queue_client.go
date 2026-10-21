package client

import (
	"context"
	"errors"

	"gitlab.com/fami-app-v2/familib/familabs/env"
)

// ErrTaskQueueNotSetUp is returned until the real background-job client
// exists.
var ErrTaskQueueNotSetUp = errors.New("task queue: Cloud Tasks client is not set up yet (TODO, plan Q4)")

// ITaskQueueClient runs a job later: it POSTs body to
// <app.base_url><path> (one of the /v1/queue/* callbacks) with Basic auth
// [public_api]. When it is not enabled the services do the job right away
// instead (site-dev-api's local-dev behaviour).
type ITaskQueueClient interface {
	// Enabled is config cloud_task.enabled.
	Enabled() bool
	// Enqueue adds one job; name is the job id prefix (e.g.
	// "approval-email-TS1"), the client adds a unique suffix.
	Enqueue(ctx context.Context, name string, path string, body []byte) (err error)
}

// NewTaskQueueClient returns the background-job client.
//
// TODO(plan Q4, needs Wes's OK): site-dev-api uses Google Cloud Tasks
// (familabs v1.0.149 cloudtaskx + cloud.google.com/go/cloudtasks): task
// "projects/<cloud_task.id>/locations/<cloud_task.location>/queues/
// <cloud_task.queue>/tasks/<name>-<unix nanos>", HTTP POST to
// <app.base_url><path>, headers Content-Type application/json and
// Authorization Basic base64(public_api.username:public_api.password),
// scheduled now. Neither library is in the boilerplate's versions. With
// cloud_task.enabled=false (the default) nothing is queued and the jobs run
// at once, so only enabled=true needs this.
func NewTaskQueueClient(env env.Provider) ITaskQueueClient {
	return pendingTaskQueueClient{enabled: env != nil && env.GetBool("cloud_task.enabled")}
}

type pendingTaskQueueClient struct{ enabled bool }

func (c pendingTaskQueueClient) Enabled() bool { return c.enabled }

func (pendingTaskQueueClient) Enqueue(context.Context, string, string, []byte) error {
	return ErrTaskQueueNotSetUp
}
