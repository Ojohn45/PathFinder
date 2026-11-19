package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

// ---------- Mail settings kept INSIDE the app ----------
//
// You do not need a .env file. The head admin can type the mail details on the
// Email Settings page, and they are saved in storage/settings.json.
//
// Order of precedence:
//   1. a real environment variable (useful on Render, Docker, a VPS)
//   2. the value saved on the Email Settings page
//   3. the Gmail default
//
// So a .env file is optional: env vars still win when they exist, which keeps
// deployed servers in control of their own secrets.

type MailSettings struct {
	Host string `json:"host"`
	Port string `json:"port"`
	User string `json:"user"`
	Pass string `json:"pass"`
	From string `json:"from"`
}

const settingsPath = "storage/settings.json"

var (
	mailMu  sync.RWMutex
	mailSet MailSettings
)

// loadMailSettings reads storage/settings.json if it exists. Missing file is
// normal: it just means nothing has been saved on the page yet.
func loadMailSettings() {
	mailMu.Lock()
	defer mailMu.Unlock()
	mailSet = MailSettings{}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return
	}
	var stored MailSettings
	if err := json.Unmarshal(data, &stored); err != nil {
		log.Println("Could not read", settingsPath, "-", err)
		return
	}
	mailSet = stored
	if stored.User != "" {
		log.Println("EMAIL: using the mail account saved on the Email Settings page (" + stored.User + ")")
	}
}

// saveMailSettings writes the details to storage/settings.json.
// The file sits in storage/, which is already excluded from git, so the
// password never reaches GitHub.
func saveMailSettings(in MailSettings) error {
	in.Host = strings.TrimSpace(in.Host)
	in.Port = strings.TrimSpace(in.Port)
	in.User = strings.TrimSpace(in.User)
	in.Pass = strings.TrimSpace(in.Pass)
	in.From = strings.TrimSpace(in.From)

	if in.Host == "" {
		in.Host = "smtp.gmail.com"
	}
	if in.Port == "" {
		in.Port = "587"
	}
	if in.From == "" {
		in.From = in.User
	}
	if in.User == "" || in.Pass == "" {
		return errors.New("the email address and the app password are both needed")
	}
	if !strings.Contains(in.User, "@") {
		return errors.New("that does not look like an email address")
	}

	data, err := json.MarshalIndent(in, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll("storage", 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(settingsPath, data, 0o600); err != nil {
		return err
	}

	mailMu.Lock()
	mailSet = in
	mailMu.Unlock()
	return nil
}

// effectiveMail is what the app actually uses to send.
//
// Environment variables are treated as a PAIR: SMTP_USER and SMTP_PASS are only
// used together. That way a stray variable in the environment (a leftover, or a
// hosting platform's own variable) can never override a good saved password and
// leave the app half configured.
func effectiveMail() MailSettings {
	mailMu.RLock()
	saved := mailSet
	mailMu.RUnlock()

	envUser := strings.TrimSpace(os.Getenv("SMTP_USER"))
	envPass := strings.ReplaceAll(strings.TrimSpace(os.Getenv("SMTP_PASS")), " ", " ")
	envPass = strings.ReplaceAll(envPass, " ", "")

	out := MailSettings{
		Host: firstNonEmpty(os.Getenv("SMTP_HOST"), saved.Host, "smtp.gmail.com"),
		Port: firstNonEmpty(os.Getenv("SMTP_PORT"), saved.Port, "587"),
	}

	switch {
	case envUser != "" && envPass != "": // a complete pair in the environment wins
		out.User, out.Pass = envUser, envPass
	case saved.User != "" && saved.Pass != "": // the complete pair saved on the page
		out.User, out.Pass = saved.User, strings.ReplaceAll(saved.Pass, " ", "")
	case envUser != "" || envPass != "": // half configured in the environment
		out.User, out.Pass = envUser, envPass
	default: // half configured on the page
		out.User, out.Pass = saved.User, strings.ReplaceAll(saved.Pass, " ", "")
	}

	out.From = firstNonEmpty(os.Getenv("SMTP_FROM"), saved.From, out.User)
	return out
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// mailReady says whether there is enough to attempt a send.
func mailReady() bool {
	m := effectiveMail()
	return m.User != "" && m.Pass != ""
}

// ---------- The page ----------

func emailSettingsPageHandler(w http.ResponseWriter, r *http.Request) {
	u := requireHeadAdmin(w, r)
	if u == nil {
		return
	}

	data := PageData{User: u}
	data.Mail = effectiveMail()
	data.MailSaved = mailSet.User != ""
	data.MailFromEnv = os.Getenv("SMTP_USER") != ""
	data.MailReady = mailReady()

	switch r.URL.Query().Get("saved") {
	case "1":
		data.Info = "Saved. PathFinder will use this account from now on, no restart needed."
	case "test-sent":
		data.Info = "Saved, and a test email is on its way to your own address. Check your inbox and Spam."
	case "test-failed":
		data.Error = "Saved, but the test email could not be sent. The reason is on the Email Log page."
	}
	render(w, "email-settings.html", data)
}

func emailSettingsSaveHandler(w http.ResponseWriter, r *http.Request) {
	u := requireHeadAdmin(w, r)
	if u == nil {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	in := MailSettings{
		Host: r.FormValue("host"),
		Port: r.FormValue("port"),
		User: r.FormValue("user"),
		Pass: r.FormValue("pass"),
		From: r.FormValue("from"),
	}
	// An empty password box means "keep the one already saved", so the head
	// admin can change the address without retyping the secret.
	if strings.TrimSpace(in.Pass) == "" && mailSet.Pass != "" {
		in.Pass = mailSet.Pass
	}

	if err := saveMailSettings(in); err != nil {
		data := PageData{User: u, Error: err.Error()}
		data.Mail = effectiveMail()
		data.MailSaved = mailSet.User != ""
		data.MailFromEnv = os.Getenv("SMTP_USER") != ""
		render(w, "email-settings.html", data)
		return
	}

	// Saving "send a test" on the same form fires a test email to the head admin.
	if r.FormValue("send_test") == "1" {
		if err := sendTestEmail(*u); err != nil {
			log.Println("Test email after saving settings failed:", err)
			http.Redirect(w, r, "/admin/email-settings?test-failed=1", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/admin/email-settings?test-sent=1", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin/email-settings?saved=1", http.StatusSeeOther)
}

// sendTestEmail mails the head admin so they can see the whole path works.
func sendTestEmail(u User) error {
	subject := "PathFinder test email ✅"
	heading := "Your email is working"
	message := "This is a test from your PathFinder admin page. If you can read this, invitations and welcome " +
		"emails will reach people too. Nothing else to do."
	text, htmlBody := inviteEmail(heading, message, "Open PathFinder", baseURL())
	return sendEmail(u.Email, subject, text, htmlBody)
}
