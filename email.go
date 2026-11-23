package main

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	_ "embed" // lets us bake the banner picture into the program
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/http"
	"net/smtp"
	"net/textproto"
	"os"
	"strings"
	"time"
)

// ---------- Email settings ----------
//
// Put these in a file called .env next to main.go (see .env.example):
//   SMTP_USER  your Gmail address
//   SMTP_PASS  a Gmail "App Password" (NOT your normal password)
// Optional: SMTP_HOST (default smtp.gmail.com), SMTP_PORT (default 587), SMTP_FROM

// The animated dark-blue banner shown at the top of invitation emails.
// It is baked into the program, so it always travels WITH the email
// (no website needed to show it).
//
//go:embed static/email-banner.gif
var bannerGIF []byte

const bannerCID = "pathfinder-banner@pathfinder.local"

var errEmailNotConfigured = errors.New("email is not set up yet (no mail account saved)")

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// loadDotEnv reads a ".env" file (if there is one) and sets each line as an
// environment variable. Real environment variables win over the file.
func loadDotEnv() {
	data, err := os.ReadFile(".env")
	if err != nil {
		wd, _ := os.Getwd()
		log.Println("No .env file found in", wd)
		// A very common mix-up: the file exists but has the wrong name.
		for _, wrong := range []string{"set.env", "env", ".env.txt", "env.txt", ".env.example"} {
			if _, statErr := os.Stat(wrong); statErr == nil {
				log.Printf("HINT: I found %q - rename it to exactly .env (a dot, then env, nothing else)", wrong)
				break
			}
		}
		return
	}
	loaded := 0
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') && value[len(value)-1] == value[0] {
			value = value[1 : len(value)-1] // remove surrounding quotes
		}
		if key == "" || value == "" || os.Getenv(key) != "" {
			continue
		}
		os.Setenv(key, value)
		loaded++
	}
	log.Printf("Loaded %d setting(s) from .env", loaded)
}

// logEmailStatus tells you right at startup whether email is ready.
func logEmailStatus() {
	m := effectiveMail()
	switch {
	case brevoKey() != "":
		log.Println("EMAIL: Brevo API key found (emails go out over https, which Render's free plan allows) - checking it...")
		go checkBrevoAPI() // runs in the background, result appears in a few seconds
	case m.User == "" && m.Pass == "":
		log.Println("EMAIL: not set up yet - log in as the head admin and open Admin > Email Settings")
	case m.User == "" || m.Pass == "":
		log.Println("EMAIL: half set up - it needs both a sending address and an app password (Admin > Email Settings)")
	default:
		log.Println("EMAIL: sending as", m.User, "- testing the login with the mail server...")
		go checkEmailLogin() // runs in the background, result appears in a few seconds
	}
}

// checkEmailLogin connects to the mail server and logs in, WITHOUT sending
// anything, so you find out at startup if the Gmail details are right.
func checkEmailLogin() {
	m := effectiveMail()
	user, pass, host, port := m.User, m.Pass, m.Host, m.Port

	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 10*time.Second)
	if err != nil {
		log.Println("EMAIL CHECK: could not reach", host, "- is your internet working?", err)
		return
	}
	conn.SetDeadline(time.Now().Add(15 * time.Second)) // never wait forever
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		log.Println("EMAIL CHECK: the mail server did not answer properly:", err)
		return
	}
	defer c.Close()

	if err := c.Hello("localhost"); err != nil {
		log.Println("EMAIL CHECK: the mail server did not answer properly:", err)
		return
	}
	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: host}); err != nil {
			log.Println("EMAIL CHECK: could not make a secure connection:", err)
			return
		}
	}
	if err := c.Auth(smtp.PlainAuth("", user, pass, host)); err != nil {
		log.Println("EMAIL CHECK: FAILED - the mail server refused the login for", user)
		log.Println("   reason:", err)
		log.Println("   FIX: check the details on Admin > Email Settings:")
		log.Println("        Gmail -> SMTP_USER = the Gmail that created the App Password, SMTP_PASS = that 16-letter App Password")
		log.Println("        Brevo -> SMTP_USER = the Login on Brevo's SMTP page, SMTP_PASS = the SMTP key (not your Brevo password)")
		return
	}
	c.Quit()
	log.Println("EMAIL CHECK: OK - login accepted, invitations will be emailed from", user)
}

// ---------- Brevo over HTTPS (works where SMTP is blocked, like Render's free plan) ----------
//
// Set these two in Render > Environment (or .env):
//   BREVO_API_KEY   an API key from Brevo
//   EMAIL_FROM      the sender address you verified inside Brevo

func brevoKey() string { return strings.TrimSpace(os.Getenv("BREVO_API_KEY")) }

func brevoBase() string {
	return strings.TrimRight(getenv("BREVO_API_BASE", "https://api.brevo.com/v3"), "/")
}

// brevoSender is the address emails come from. It MUST be a sender verified in Brevo.
func brevoSender() string {
	for _, v := range []string{os.Getenv("EMAIL_FROM"), effectiveMail().From} {
		if v = strings.TrimSpace(v); strings.Contains(v, "@") {
			return v
		}
	}
	return ""
}

func checkBrevoAPI() {
	req, _ := http.NewRequest("GET", brevoBase()+"/account", nil)
	req.Header.Set("api-key", brevoKey())
	req.Header.Set("Accept", "application/json")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		log.Println("EMAIL CHECK: could not reach Brevo -", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		log.Printf("EMAIL CHECK: FAILED - Brevo refused the API key (HTTP %d). FIX: copy a fresh API key from Brevo into BREVO_API_KEY.", resp.StatusCode)
		return
	}
	if brevoSender() == "" {
		log.Println("EMAIL CHECK: API key OK, but there is no sender yet - set EMAIL_FROM to the sender address you verified in Brevo")
		return
	}
	log.Println("EMAIL CHECK: OK - Brevo accepted the API key, emails will be sent from", brevoSender())
}

// sendViaBrevoAPI sends one email through Brevo's HTTPS API (port 443).
func sendViaBrevoAPI(toEmail, subject, textBody, htmlBody string) error {
	from := brevoSender()
	if from == "" {
		return errors.New("no sender address: set EMAIL_FROM to the sender you verified in Brevo")
	}
	payload := map[string]any{
		"sender":      map[string]string{"name": "PathFinder", "email": from},
		"to":          []map[string]string{{"email": toEmail}},
		"subject":     subject,
		"textContent": textBody,
	}
	if htmlBody != "" {
		// Over the API the banner can't travel inside the email, so it loads from the website.
		payload["htmlContent"] = strings.ReplaceAll(htmlBody, "cid:"+bannerCID, baseURL()+"/static/email-banner.gif")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", brevoBase()+"/smtp/email", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("api-key", brevoKey())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 400))
		return fmt.Errorf("Brevo said HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}
	return nil
}

// ---------- Sending ----------

// sendEmail sends an email to ONE person and writes the result in the email diary.
// If htmlBody is empty it is plain text; otherwise it is a "text + pretty HTML" email.
// toEmail must already be validated.
func sendEmail(toEmail, subject, textBody, htmlBody string) (err error) {
	defer func() {
		rec := EmailRecord{Time: time.Now(), To: toEmail, Subject: subject, Status: "sent"}
		switch {
		case errors.Is(err, errEmailNotConfigured):
			rec.Status = "not_configured"
		case err != nil:
			rec.Status = "failed"
			rec.Error = err.Error()
		}
		addEmailRecord(rec)
	}()

	// Option 1: Brevo over https (use this on Render's free plan, which blocks SMTP)
	if brevoKey() != "" {
		return sendViaBrevoAPI(toEmail, subject, textBody, htmlBody)
	}

	// Option 2: normal SMTP
	m := effectiveMail()
	user, pass, host, port, from := m.User, m.Pass, m.Host, m.Port, m.From
	if user == "" || pass == "" {
		return errEmailNotConfigured
	}

	msg, err := buildMessage(from, toEmail, subject, textBody, htmlBody)
	if err != nil {
		return err
	}

	auth := smtp.PlainAuth("", user, pass, host)
	return smtp.SendMail(net.JoinHostPort(host, port), auth, from, []string{toEmail}, msg)
}

func writeQP(w interface{ Write([]byte) (int, error) }, s string) {
	qp := quotedprintable.NewWriter(w)
	qp.Write([]byte(s))
	qp.Close()
}

func buildMessage(from, to, subject, textBody, htmlBody string) ([]byte, error) {
	idBytes := make([]byte, 8)
	if _, err := rand.Read(idBytes); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	header := func(k, v string) { buf.WriteString(k + ": " + v + "\r\n") }
	header("From", "PathFinder <"+from+">")
	header("To", to)
	header("Subject", mime.QEncoding.Encode("utf-8", subject))
	header("Date", time.Now().Format(time.RFC1123Z))
	msgDomain := "redirect.local"
	if at := strings.LastIndex(from, "@"); at >= 0 && at < len(from)-1 {
		msgDomain = from[at+1:]
	}
	header("Message-ID", "<"+hex.EncodeToString(idBytes)+"@"+msgDomain+">")
	header("MIME-Version", "1.0")

	// Plain text email (no HTML)
	if htmlBody == "" {
		header("Content-Type", "text/plain; charset=UTF-8")
		header("Content-Transfer-Encoding", "quoted-printable")
		buf.WriteString("\r\n")
		writeQP(&buf, textBody)
		return buf.Bytes(), nil
	}

	// Does the HTML use the animated banner? Then we attach it ("related").
	withBanner := strings.Contains(htmlBody, "cid:"+bannerCID)

	// Writes the "text version + pretty version" pair into w.
	writeAlternatives := func(w *multipart.Writer) error {
		for _, part := range []struct{ ctype, body string }{
			{"text/plain; charset=UTF-8", textBody}, // simple version first
			{"text/html; charset=UTF-8", htmlBody},  // pretty version last (mail apps prefer it)
		} {
			pw, err := w.CreatePart(textproto.MIMEHeader{
				"Content-Type":              {part.ctype},
				"Content-Transfer-Encoding": {"quoted-printable"},
			})
			if err != nil {
				return err
			}
			writeQP(pw, part.body)
		}
		return nil
	}

	if !withBanner {
		mw := multipart.NewWriter(&buf)
		header("Content-Type", "multipart/alternative; boundary="+mw.Boundary())
		buf.WriteString("\r\n")
		if err := writeAlternatives(mw); err != nil {
			return nil, err
		}
		if err := mw.Close(); err != nil { // write the closing line FIRST...
			return nil, err
		}
		return buf.Bytes(), nil // ...and only then take the finished message
	}

	// related( alternative(text, html), image )
	outer := multipart.NewWriter(&buf)
	header("Content-Type", "multipart/related; type=\"multipart/alternative\"; boundary="+outer.Boundary())
	buf.WriteString("\r\n")

	// the text/html pair
	var altBuf bytes.Buffer
	inner := multipart.NewWriter(&altBuf)
	if err := writeAlternatives(inner); err != nil {
		return nil, err
	}
	if err := inner.Close(); err != nil {
		return nil, err
	}
	pw, err := outer.CreatePart(textproto.MIMEHeader{
		"Content-Type": {"multipart/alternative; boundary=" + inner.Boundary()},
	})
	if err != nil {
		return nil, err
	}
	pw.Write(altBuf.Bytes())

	// the animated picture
	iw, err := outer.CreatePart(textproto.MIMEHeader{
		"Content-Type":              {"image/gif; name=\"pathfinder-banner.gif\""},
		"Content-Transfer-Encoding": {"base64"},
		"Content-ID":                {"<" + bannerCID + ">"},
		"Content-Disposition":       {"inline; filename=\"pathfinder-banner.gif\""},
	})
	if err != nil {
		return nil, err
	}
	encoded := base64.StdEncoding.EncodeToString(bannerGIF)
	for len(encoded) > 0 { // email lines must be short: 76 characters each
		n := 76
		if len(encoded) < n {
			n = len(encoded)
		}
		iw.Write([]byte(encoded[:n] + "\r\n"))
		encoded = encoded[n:]
	}
	if err := outer.Close(); err != nil { // write the closing line FIRST...
		return nil, err
	}
	return buf.Bytes(), nil // ...and only then take the finished message
}

// ---------- The emails themselves ----------

// inviteEmail builds the dark-blue invitation email: an animated banner,
// a heading, ONE short paragraph that ends with the question "would you
// like to join?", and an Accept Invite button.
func inviteEmail(heading, message, buttonText, link string) (text, htmlBody string) {
	text = heading + "\n\n" +
		message + "\n\n" +
		strings.ToUpper(buttonText) + ":\n" + link + "\n\n" +
		"If you weren't expecting this email, you can safely ignore it.\n\n" +
		"The PathFinder Team\n"

	e := html.EscapeString
	var h strings.Builder
	h.WriteString(`<!DOCTYPE html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">` +
		`<meta name="color-scheme" content="dark"><meta name="supported-color-schemes" content="dark"></head>`)
	h.WriteString(`<body style="margin:0;padding:0;background:#030a18;" bgcolor="#030a18">`)
	h.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" bgcolor="#030a18" style="background:#030a18;"><tr><td align="center" style="padding:24px 12px;">`)
	h.WriteString(`<table role="presentation" width="600" cellpadding="0" cellspacing="0" bgcolor="#0a1a3a" style="max-width:600px;width:100%;background:#0a1a3a;border:1px solid #1e3a5c;border-radius:14px;">`)
	// animated banner
	h.WriteString(`<tr><td style="padding:0;line-height:0;font-size:0;border-radius:14px 14px 0 0;">` +
		`<img src="cid:` + bannerCID + `" width="600" alt="PathFinder - find your way to the right resource" ` +
		`style="display:block;width:100%;max-width:600px;height:auto;border:0;border-radius:14px 14px 0 0;color:#6cb2ff;font-size:22px;"></td></tr>`)
	h.WriteString(`<tr><td style="padding:32px 36px 12px;">`)
	h.WriteString(`<h1 style="margin:0 0 16px;color:#eef6ff;font-family:'Segoe UI',Arial,sans-serif;font-size:26px;line-height:1.3;">` + e(heading) + `</h1>`)
	h.WriteString(`<p style="margin:0;color:#c5d6ee;font-family:'Segoe UI',Arial,sans-serif;font-size:16px;line-height:1.7;">` + e(message) + `</p>`)
	h.WriteString(`</td></tr>`)
	// button
	h.WriteString(`<tr><td align="center" style="padding:24px 36px 28px;">` +
		`<table role="presentation" cellpadding="0" cellspacing="0"><tr><td align="center" bgcolor="#2b7de9" style="background:#2b7de9;border-radius:10px;">` +
		`<a href="` + e(link) + `" style="display:inline-block;padding:15px 44px;color:#ffffff;font-family:'Segoe UI',Arial,sans-serif;font-size:17px;font-weight:bold;text-decoration:none;border-radius:10px;">` + e(buttonText) + `</a>` +
		`</td></tr></table></td></tr>`)
	// small print
	h.WriteString(`<tr><td style="padding:0 36px 30px;font-family:'Segoe UI',Arial,sans-serif;font-size:12px;line-height:1.6;color:#7f9bc0;">` +
		`If the button doesn't work, copy this link into your browser:<br>` +
		`<a href="` + e(link) + `" style="color:#6cb2ff;word-break:break-all;">` + e(link) + `</a><br><br>` +
		`If you weren't expecting this email, you can safely ignore it.` +
		`</td></tr>`)
	h.WriteString(`</table></td></tr></table></body></html>`)
	return text, h.String()
}

// sendWelcomeEmail sends the branded welcome: the animated PathFinder banner at
// the top, one short paragraph, and a button into Browse.
func sendWelcomeEmail(toEmail, fullName string) {
	name := fullName
	if parts := strings.Fields(fullName); len(parts) > 0 {
		name = parts[0]
	}

	subject := "Welcome to PathFinder 🎉"
	heading := "Welcome, " + name + "!"
	message := "PathFinder exists to take the searching out of learning. Tell it what you want, whether that is a " +
		"programming language, a film, a manga or a game, and it points you straight to the best places to read, " +
		"watch, listen or play it, then keeps every visit in your history so you never lose your place. " +
		"Press the button, choose a section, and start with one thing today."
	link := baseURL() + "/browse"

	text := "Hi " + name + ",\n\nWelcome to PathFinder! Your account is ready.\n\n" +
		"Pick a section and PathFinder points you straight to the best places to learn, watch, read or play, " +
		"and remembers where you have been.\n\nStart here: " + link + "\n\nThe PathFinder Team\n"
	_, htmlBody := inviteEmail(heading, message, "Start with Browse", link)

	err := sendEmail(toEmail, subject, text, htmlBody)
	switch {
	case errors.Is(err, errEmailNotConfigured):
		log.Println("Email not configured - skipping welcome email to", toEmail)
	case err != nil:
		log.Println("Could not send welcome email to", toEmail, ":", err)
	default:
		log.Println("Welcome email sent to", toEmail)
	}
}
