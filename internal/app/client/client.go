package client

import "gitlab.com/fami-app-v2/familib/familabs/env"

// "git.edenfarm.id/project-version3/erp-pkg/erp-client-grpc/client"

// Clients all client object injected here
type Clients struct {
	// AccountServiceGrpc       client.IAccountServiceGrpc
	// AuditServiceGrpc         client.IAuditServiceGrpc
	// BridgeServiceGrpc        client.IBridgeServiceGrpc
	// ConfigurationServiceGrpc client.IConfigurationServiceGrpc

	// Outside services Site Dev calls (one interface each; services get
	// them through the New...Client constructors):
	ApprovalJWT IApprovalJWTClient // fami-module token for approval-link login
	Storage     IStorageClient     // site photos (TODO: Google Cloud Storage, plan Q4)
	Gemini      IGeminiClient      // AI insight (TODO: the HTTP call, plan Q4)
	TaskQueue   ITaskQueueClient   // background jobs (TODO: Cloud Tasks, plan Q4)
	Mail        IMailClient        // approval e-mail (TODO: the sender, plan Q4)
	Geocoding   IGeocodingClient   // Google Geocoding API (ported, stdlib only)
}

// NewClients builds every client from the config; called once at start
// (cmd) and kept in global.Setup.Clients.
func NewClients(env env.Provider) Clients {
	return Clients{
		ApprovalJWT: NewApprovalJWTClient(env),
		Storage:     NewStorageClient(env),
		Gemini:      NewGeminiClient(env),
		TaskQueue:   NewTaskQueueClient(env),
		Mail:        NewMailClient(env),
		Geocoding:   NewGeocodingClient(env),
	}
}
