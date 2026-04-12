// HTML confirm/thanks and unsubscribe pages are a UX layer on top of the frozen swagger.yaml contract.
// Use ?format=json on GET /api/confirm/{token} or GET /api/unsubscribe/{token} for the documented JSON {"ok":true} response.
package handlers

import (
	_ "embed"
	"errors"
	"html/template"
	"net/http"
	"strings"

	"email-subscription-service/internal/domain"
)

//go:embed assets/style.css
var embeddedRadarCSS string

// defaultSubscribeUI is used when WEB_UI_URL is unset (e.g. local dev without env).
const defaultSubscribeUI = "https://genesis-email-subscription.vercel.app"

func wantsJSONResponse(r *http.Request) bool {
	return strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("format")), "json")
}

func (h SubscriptionHandlers) subscribePageURL() string {
	u := strings.TrimRight(strings.TrimSpace(h.WebUIURL), "/")
	if u != "" {
		return u
	}
	return defaultSubscribeUI
}

func (h SubscriptionHandlers) thanksPageURL() string {
	return strings.TrimRight(strings.TrimSpace(h.APIPublicURL), "/") + "/api/confirm/thanks"
}

func (h SubscriptionHandlers) writeConfirmSuccessHTML(w http.ResponseWriter) {
	data := struct {
		Title        string
		CSS          template.CSS
		SubscribeURL string
		ThanksURL    string
	}{
		Title:        "RELEASE RADAR // Confirmed",
		CSS:          template.CSS(embeddedRadarCSS),
		SubscribeURL: h.subscribePageURL(),
		ThanksURL:    h.thanksPageURL(),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = confirmSuccessTpl.Execute(w, data)
}

func (h SubscriptionHandlers) writeConfirmErrorHTML(w http.ResponseWriter, status int, err error) {
	msg := http.StatusText(status)
	switch {
	case errors.Is(err, domain.ErrInvalidToken):
		status = http.StatusBadRequest
		msg = "This confirmation link is invalid or incomplete."
	case errors.Is(err, domain.ErrTokenNotFound):
		status = http.StatusNotFound
		msg = "We could not find this confirmation link. It may have already been used or expired."
	default:
		status = http.StatusInternalServerError
		msg = "Something went wrong while confirming. Please try again later."
	}
	data := struct {
		Title        string
		CSS          template.CSS
		Status       int
		Message      string
		SubscribeURL string
	}{
		Title:        "RELEASE RADAR // Confirmation",
		CSS:          template.CSS(embeddedRadarCSS),
		Status:       status,
		Message:      msg,
		SubscribeURL: h.subscribePageURL(),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = confirmErrorTpl.Execute(w, data)
}

func writeConfirmErrorHTMLFromErr(w http.ResponseWriter, h SubscriptionHandlers, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, domain.ErrInvalidToken):
		status = http.StatusBadRequest
	case errors.Is(err, domain.ErrTokenNotFound):
		status = http.StatusNotFound
	}
	h.writeConfirmErrorHTML(w, status, err)
}

func (h SubscriptionHandlers) writeUnsubscribeSuccessHTML(w http.ResponseWriter) {
	data := struct {
		Title        string
		CSS          template.CSS
		SubscribeURL string
	}{
		Title:        "RELEASE RADAR // Unsubscribed",
		CSS:          template.CSS(embeddedRadarCSS),
		SubscribeURL: h.subscribePageURL(),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = unsubscribeSuccessTpl.Execute(w, data)
}

func (h SubscriptionHandlers) writeUnsubscribeErrorHTML(w http.ResponseWriter, status int, err error) {
	msg := http.StatusText(status)
	switch {
	case errors.Is(err, domain.ErrInvalidToken):
		status = http.StatusBadRequest
		msg = "This unsubscribe link is invalid or incomplete."
	case errors.Is(err, domain.ErrTokenNotFound):
		status = http.StatusNotFound
		msg = "We could not find this unsubscribe link. It may have already been used."
	default:
		status = http.StatusInternalServerError
		msg = "Something went wrong while processing your unsubscribe request. Please try again later."
	}
	data := struct {
		Title        string
		CSS          template.CSS
		Status       int
		Message      string
		SubscribeURL string
	}{
		Title:        "RELEASE RADAR // Unsubscribe",
		CSS:          template.CSS(embeddedRadarCSS),
		Status:       status,
		Message:      msg,
		SubscribeURL: h.subscribePageURL(),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = unsubscribeErrorTpl.Execute(w, data)
}

func writeUnsubscribeErrorHTMLFromErr(w http.ResponseWriter, h SubscriptionHandlers, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, domain.ErrInvalidToken):
		status = http.StatusBadRequest
	case errors.Is(err, domain.ErrTokenNotFound):
		status = http.StatusNotFound
	}
	h.writeUnsubscribeErrorHTML(w, status, err)
}

// ConfirmThanks serves a static thank-you page after the user declines another subscription.
func (h SubscriptionHandlers) ConfirmThanks(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Title string
		CSS   template.CSS
	}{
		Title: "RELEASE RADAR // Thank you",
		CSS:   template.CSS(embeddedRadarCSS),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = confirmThanksTpl.Execute(w, data)
}

var (
	confirmSuccessTpl     = template.Must(template.New("confirm_ok").Parse(confirmSuccessHTML))
	confirmErrorTpl       = template.Must(template.New("confirm_err").Parse(confirmErrorHTML))
	confirmThanksTpl      = template.Must(template.New("thanks").Parse(confirmThanksHTML))
	unsubscribeSuccessTpl = template.Must(template.New("unsub_ok").Parse(unsubscribeSuccessHTML))
	unsubscribeErrorTpl   = template.Must(template.New("unsub_err").Parse(unsubscribeErrorHTML))
)

const radarChrome = `
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>{{.Title}}</title>
  <link rel="preconnect" href="https://fonts.googleapis.com" />
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
  <link href="https://fonts.googleapis.com/css2?family=Orbitron:wght@500;700;900&family=Share+Tech+Mono&display=swap" rel="stylesheet" />
  <style>{{.CSS}}</style>
</head>
<body>
  <div class="noise" aria-hidden="true"></div>
  <div class="grid-bg" aria-hidden="true"></div>
  <header class="top-bar">
    <div class="logo">
      <span class="logo-hex">⬡</span>
      <span>RELEASE_RADAR</span>
    </div>
    <div class="chain-pill">
      <span class="pulse"></span>
      <span>CHAIN_LINK_OK</span>
    </div>
  </header>
  <main class="shell">
`

const radarFooter = `
  </main>
  <footer class="foot">
    <span class="mono">GitHub Release Notification API</span>
  </footer>
</body>
</html>
`

const confirmSuccessHTML = radarChrome + `
    <section class="hero">
      <p class="eyebrow">Signal locked</p>
      <h1>Your subscription is <span class="accent">confirmed</span></h1>
      <p class="sub">You will get an email when this repository publishes a <strong>new release tag</strong>.</p>
    </section>
    <section class="panel">
      <div class="panel-corner tl"></div>
      <div class="panel-corner tr"></div>
      <div class="panel-corner bl"></div>
      <div class="panel-corner br"></div>
      <p class="sub" style="margin:0 0 0.5rem">Would you like to subscribe to another GitHub repository?</p>
      <div class="btn-row">
        <a class="btn-primary btn-link" href="{{.SubscribeURL}}"><span class="btn-glow"></span><span>Yes — another repo</span></a>
        <a class="btn-secondary btn-link" href="{{.ThanksURL}}">No, I&apos;m done</a>
      </div>
    </section>
` + radarFooter

const confirmErrorHTML = radarChrome + `
    <section class="hero">
      <p class="eyebrow">Link status {{.Status}}</p>
      <h1>Confirmation <span class="accent">did not complete</span></h1>
      <p class="sub">{{.Message}}</p>
    </section>
    <section class="panel">
      <div class="panel-corner tl"></div>
      <div class="panel-corner tr"></div>
      <div class="panel-corner bl"></div>
      <div class="panel-corner br"></div>
      <p class="sub" style="margin:0">You can return to the subscribe page and try again if you still need access.</p>
      <div class="btn-row">
        <a class="btn-primary btn-link" href="{{.SubscribeURL}}"><span class="btn-glow"></span><span>Back to subscribe</span></a>
      </div>
    </section>
` + radarFooter

const confirmThanksHTML = radarChrome + `
    <section class="hero">
      <p class="eyebrow">Channel closed</p>
      <h1>Thank you — <span class="accent">have a lovely time</span></h1>
      <p class="sub">You can close this tab. We&apos;re glad you&apos;re set up for release signals.</p>
    </section>
` + radarFooter

const unsubscribeSuccessHTML = radarChrome + `
    <section class="hero">
      <p class="eyebrow">Signal dropped</p>
      <h1>Goodbye — <span class="accent">it was nice to have you</span></h1>
      <p class="sub">You will not receive further release notifications for this subscription. You can subscribe again anytime from the main page.</p>
    </section>
    <section class="panel">
      <div class="panel-corner tl"></div>
      <div class="panel-corner tr"></div>
      <div class="panel-corner bl"></div>
      <div class="panel-corner br"></div>
      <p class="sub" style="margin:0">Want release signals for another repository later?</p>
      <div class="btn-row">
        <a class="btn-primary btn-link" href="{{.SubscribeURL}}"><span class="btn-glow"></span><span>Back to RELEASE_RADAR</span></a>
      </div>
    </section>
` + radarFooter

const unsubscribeErrorHTML = radarChrome + `
    <section class="hero">
      <p class="eyebrow">Link status {{.Status}}</p>
      <h1>Unsubscribe <span class="accent">did not complete</span></h1>
      <p class="sub">{{.Message}}</p>
    </section>
    <section class="panel">
      <div class="panel-corner tl"></div>
      <div class="panel-corner tr"></div>
      <div class="panel-corner bl"></div>
      <div class="panel-corner br"></div>
      <p class="sub" style="margin:0">If you still need to leave a subscription, open the latest unsubscribe link from your email, or return to the subscribe page.</p>
      <div class="btn-row">
        <a class="btn-primary btn-link" href="{{.SubscribeURL}}"><span class="btn-glow"></span><span>Back to subscribe</span></a>
      </div>
    </section>
` + radarFooter
