package client

// The approval-pending e-mail template of site-dev-api 20eeab4
// internal/app/mailer/template.go, unchanged.

import (
	"fmt"
	"html/template"
	"strings"
)

var approvalPendingTpl = template.Must(template.New("approval_pending").Parse(`
<!DOCTYPE html>
<html lang="id">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Approval Site Development</title>
<style>
  @import url('https://fonts.googleapis.com/css2?family=Poppins:wght@600;700;800&family=Inter:wght@400;500;600;700&family=Roboto+Mono:wght@400;500;600&display=swap');

  * { box-sizing: border-box; }
  body {
    margin: 0;
    padding: 40px 16px;
    background: #EEF1EC;
    background-image:
      radial-gradient(circle at 1px 1px, #dfe4dc 1px, transparent 1px);
    background-size: 18px 18px;
    font-family: 'Inter', Arial, sans-serif;
    color: #1F2A24;
  }

  .wrap { max-width: 460px; margin: 0 auto; width: 100%; }

  /* ---- the receipt/ticket card ---- */
  .ticket {
    background: #FBFAF6;
    padding: 0 0 30px 0;
    border-radius: 10px;
    position: relative;
    box-shadow: 0 24px 48px -20px rgba(15,35,25,0.35), 0 2px 6px rgba(15,35,25,0.06);
  }

  .header {
    padding: 26px 32px 22px;
    border-bottom: 1px dashed #C9CFC4;
    position: relative;
  }

  .eyebrow {
    font-family: 'Roboto Mono', monospace;
    font-size: 11px;
    letter-spacing: 0.16em;
    text-transform: uppercase;
    color: #0088D0;
    font-weight: 600;
    margin: 0 0 6px;
  }

  h1 {
    font-family: 'Poppins', Arial, sans-serif;
    font-weight: 800;
    font-size: 24px;
    line-height: 1.25;
    margin: 0;
    color: #12271E;
    letter-spacing: -0.01em;
  }

  .stamp {
    position: absolute;
    top: 22px;
    right: 26px;
    transform: rotate(9deg);
    border: 2.5px solid #C97A1A;
    color: #C97A1A;
    font-family: 'Poppins', Arial, sans-serif;
    font-weight: 800;
    font-size: 12px;
    letter-spacing: 0.08em;
    padding: 5px 10px;
    border-radius: 6px;
    opacity: 0.9;
  }

  .body { padding: 24px 32px 6px; }

  .greet {
    font-size: 15px;
    margin: 0 0 4px;
    color: #12271E;
  }
  .lede {
    font-size: 14.5px;
    line-height: 1.6;
    color: #3E4A43;
    margin: 0 0 22px;
  }
  .lede b { color: #12271E; }

  /* ---- receipt line items ---- */
  .lines {
    border-bottom: 1px dashed #C9CFC4;
    padding: 16px 0;
    margin-bottom: 20px;
  }
  .line-row {
    display: table;
    width: 100%;
    padding: 7px 0;
  }
  .line-label {
    display: table-cell;
    font-family: 'Roboto Mono', monospace;
    font-size: 11.5px;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: #7C8A81;
    width: 34%;
    vertical-align: top;
    padding-top: 2px;
  }
  .line-value {
    display: table-cell;
    font-size: 14.5px;
    font-weight: 600;
    color: #12271E;
    vertical-align: top;
  }
  .line-note {
    font-weight: 400;
    color: #3E4A43;
  }

  /* ---- status pill ---- */
  .status-row {
    display: table;
    width: 100%;
    margin-bottom: 22px;
  }
  .status-text { display: table-cell; font-size: 14px; color: #3E4A43; vertical-align: middle; }
  .pill {
    display: inline-block;
    font-family: 'Roboto Mono', monospace;
    font-size: 11px;
    font-weight: 600;
    letter-spacing: 0.06em;
    background: #FBEBD3;
    color: #A85E10;
    border: 1px solid #EAC48C;
    padding: 3px 9px;
    border-radius: 20px;
    margin: 0 2px;
  }

  /* ---- CTA ---- */
  .cta-wrap { text-align: center; padding: 6px 0 4px; }
  .cta {
    display: inline-block;
    background-color: #0088D0;
    color: #FFFFFF !important;
    font-family: 'Poppins', Arial, sans-serif;
    font-weight: 700;
    font-size: 14.5px;
    letter-spacing: 0.02em;
    text-decoration: none;
    padding: 14px 40px;
    border-radius: 8px;
  }
  .cta-sub {
    font-family: 'Roboto Mono', monospace;
    font-size: 10.5px;
    color: #8C9890;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    margin-top: 10px;
  }

  /* ---- barcode footer ---- */
  .footer { padding: 26px 32px 4px; text-align: center; }
  .barcode {
    height: 32px;
    margin: 0 auto 10px;
    width: 78%;
    background: repeating-linear-gradient(
      90deg,
      #1F2A24 0px, #1F2A24 2px,
      transparent 2px, transparent 4px,
      #1F2A24 4px, #1F2A24 5px,
      transparent 5px, transparent 9px,
      #1F2A24 9px, #1F2A24 12px,
      transparent 12px, transparent 14px
    );
    opacity: 0.75;
  }
  .footnote {
    font-family: 'Roboto Mono', monospace;
    font-size: 10.5px;
    color: #9AA69C;
    letter-spacing: 0.04em;
  }

  @media only screen and (max-width: 480px) {
    body { padding: 24px 8px; }
    .header { padding: 22px 20px 18px; }
    .body { padding: 20px 20px 4px; }
    .footer { padding: 22px 20px 4px; }
    h1 { font-size: 20px; }
    .stamp { top: 18px; right: 16px; font-size: 10.5px; padding: 4px 8px; }
    .line-label { width: 40%; }
    .cta { padding: 13px 28px; font-size: 14px; }
  }
</style>
</head>
<body>
  <div class="wrap">
    <div class="ticket">

      <div class="header">
        <p class="eyebrow">Site Development &middot; Approval</p>
        <h1>Request Site Baru</h1>
      </div>

      <div class="body">
        <p class="greet">Halo <b>{{.ToName}}</b>,</p>
        <p class="lede">Ada pengajuan yang menunggu approval Anda sebagai <b>{{.ApproverRole}}</b>.</p>

        <div class="lines">
          <div class="line-row">
            <div class="line-label">Site</div>
            <div class="line-value">{{.SiteName}}</div>
          </div>
          <div class="line-row">
            <div class="line-label">Diajukan oleh</div>
            <div class="line-value">{{.SubmittedBy}}</div>
          </div>
          {{if .Notes}}
          <div class="line-row">
            <div class="line-label">Catatan</div>
            <div class="line-value line-note">{{.Notes}}</div>
          </div>
          {{end}}
        </div>

        {{if .AppURL}}
        <div class="cta-wrap">
          <a href="{{.AppURL}}" class="cta" style="display:inline-block;background-color:#0088D0;color:#FFFFFF;font-weight:700;font-size:14.5px;text-decoration:none;padding:14px 40px;border-radius:8px;">Buka &amp; Review</a>
        </div>
        {{end}}
      </div>

      <div class="footer">
        <div class="footnote">EMAIL OTOMATIS &middot; MOHON TIDAK MEMBALAS</div>
      </div>
    </div>
  </div>
</body>
</html>
`))

type approvalPendingView struct {
	ToName       string
	SiteName     string
	SubmittedBy  string
	ApproverRole string
	Notes        string
	AppURL       string
}

func renderApprovalPendingHTML(toName, appURL string, data ApprovalPendingEmail) string {
	var buf strings.Builder
	view := approvalPendingView{
		ToName:       toName,
		SiteName:     data.SiteName,
		SubmittedBy:  data.SubmittedBy,
		ApproverRole: data.ApproverRole,
		Notes:        data.Notes,
		AppURL:       appURL,
	}
	if err := approvalPendingTpl.Execute(&buf, view); err != nil {
		return fmt.Sprintf("<p>Approval diperlukan untuk task %s (%s)</p>", data.TaskID, data.SiteName)
	}
	return buf.String()
}
