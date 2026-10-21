package client

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"gitlab.com/fami-app-v2/familib/familabs/env"
)

// ErrMailNotSetUp is returned until the real e-mail sender exists.
var ErrMailNotSetUp = errors.New("mail: e-mail sender is not set up yet (TODO, plan Q4)")

// ApprovalPendingEmail is the content of the "approval waiting for you"
// e-mail (site-dev-api mailer.ApprovalPendingEmail).
type ApprovalPendingEmail struct {
	TaskID       string
	SiteName     string
	SubmittedBy  string
	ApproverRole string
	Notes        string
	// Code is ApprovalDET.SecureApprovalCode, added to the app link as
	// ?approval_code=
	Code string
}

// IMailClient sends e-mail. Today only the approval-pending e-mail.
type IMailClient interface {
	SendApprovalPending(ctx context.Context, to string, toName string, data ApprovalPendingEmail) (err error)
}

// ApprovalPendingMessage builds subject and HTML body exactly as
// site-dev-api (template in mail_template.go); frontendURL is config
// frontend.url.
func ApprovalPendingMessage(toName string, frontendURL string, data ApprovalPendingEmail) (subject string, html string) {
	subject = fmt.Sprintf("Approval Site Baru - %s", data.SiteName)
	appURL := frontendURL
	if data.Code != "" && appURL != "" {
		appURL = fmt.Sprintf("%s?approval_code=%s", appURL, url.QueryEscape(data.Code))
	}
	return subject, renderApprovalPendingHTML(toName, appURL, data)
}

// NewMailClient returns the e-mail sender.
//
// TODO(plan Q4, needs Wes's OK): site-dev-api sends over SMTP (net/smtp,
// PlainAuth) with [smtp] host, port (default 587), username, password,
// from, from_name; headers From "<from_name> <from>", To, Subject,
// MIME-Version 1.0, Content-Type text/html; charset="UTF-8"; body from
// ApprovalPendingMessage. Which mail service the new backend uses is not
// decided; until then every call answers ErrMailNotSetUp (after the same
// empty-recipient check as site-dev-api).
func NewMailClient(env env.Provider) IMailClient {
	c := pendingMailClient{}
	if env != nil {
		c.frontendURL = env.GetString("frontend.url")
	}
	return c
}

type pendingMailClient struct{ frontendURL string }

func (c pendingMailClient) SendApprovalPending(_ context.Context, to string, toName string, data ApprovalPendingEmail) error {
	if to == "" {
		return fmt.Errorf("mailer: recipient email is empty (task %s)", data.TaskID)
	}
	_, _ = ApprovalPendingMessage(toName, c.frontendURL, data)
	return ErrMailNotSetUp
}
