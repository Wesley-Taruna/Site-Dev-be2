package client

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
}
