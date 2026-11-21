package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// ---------- Text messages ----------
//
// Sending an SMS needs an account with a gateway; there is no way around that.
// Two are supported, and both are configured with environment variables:
//
//	Termii  (Nigeria, cheapest for local numbers)
//	    SMS_PROVIDER=termii
//	    TERMII_API_KEY=...
//	    TERMII_FROM=PathFinder          (the sender ID approved on your Termii account)
//
//	Twilio  (works almost everywhere)
//	    SMS_PROVIDER=twilio
//	    TWILIO_ACCOUNT_SID=...
//	    TWILIO_AUTH_TOKEN=...
//	    TWILIO_FROM=+1...
//
// With neither configured, PathFinder simply sends the email and logs that the
// text was skipped. Nothing breaks.

var errSMSNotConfigured = errors.New("no SMS gateway configured")

func smsConfigured() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("SMS_PROVIDER"))) {
	case "termii":
		return os.Getenv("TERMII_API_KEY") != ""
	case "twilio":
		return os.Getenv("TWILIO_ACCOUNT_SID") != "" && os.Getenv("TWILIO_AUTH_TOKEN") != ""
	}
	return false
}

// sendSMS texts one number. The number is normalised to +234... form for Termii
// and Twilio, which both expect the country code.
func sendSMS(phone, message string) error {
	if !smsConfigured() {
		return errSMSNotConfigured
	}
	to := internationalPhone(phone)
	client := &http.Client{Timeout: 15 * time.Second}

	switch strings.ToLower(strings.TrimSpace(os.Getenv("SMS_PROVIDER"))) {
	case "termii":
		body, _ := json.Marshal(map[string]string{
			"to":      to,
			"from":    getenv("TERMII_FROM", "PathFinder"),
			"sms":     message,
			"type":    "plain",
			"channel": "generic",
			"api_key": os.Getenv("TERMII_API_KEY"),
		})
		resp, err := client.Post("https://api.ng.termii.com/api/sms/send", "application/json", strings.NewReader(string(body)))
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			return errors.New("Termii refused the message: " + resp.Status)
		}
		return nil

	case "twilio":
		sid := os.Getenv("TWILIO_ACCOUNT_SID")
		form := url.Values{}
		form.Set("To", to)
		form.Set("From", os.Getenv("TWILIO_FROM"))
		form.Set("Body", message)
		req, err := http.NewRequest("POST", "https://api.twilio.com/2010-04-01/Accounts/"+sid+"/Messages.json", strings.NewReader(form.Encode()))
		if err != nil {
			return err
		}
		req.SetBasicAuth(sid, os.Getenv("TWILIO_AUTH_TOKEN"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			return errors.New("Twilio refused the message: " + resp.Status)
		}
		return nil
	}
	return errSMSNotConfigured
}

// internationalPhone turns 08012345678 into +2348012345678.
func internationalPhone(phone string) string {
	p := normalisePhone(phone)
	if p == "" {
		return ""
	}
	if strings.HasPrefix(p, "0") {
		return "+234" + p[1:]
	}
	if strings.HasPrefix(p, "234") {
		return "+" + p
	}
	return "+" + p
}

// welcomeMessages sends the welcome email, and the text if a gateway is set up.
func welcomeMessages(fullName, email, phone string) {
	sendWelcomeEmail(email, fullName)

	if phone == "" {
		return
	}
	first := fullName
	if parts := strings.Fields(fullName); len(parts) > 0 {
		first = parts[0]
	}
	msg := "Welcome to PathFinder, " + first + "! PathFinder finds your way to the right resource: " +
		"pick a section, and we point you straight to the best places to learn, watch, read or play, " +
		"and keep every visit in your history so you never lose your place. Log in and press Browse to start."
	if err := sendSMS(phone, msg); err != nil {
		if errors.Is(err, errSMSNotConfigured) {
			log.Println("No SMS gateway configured - welcome text to", phone, "skipped (email still sent)")
		} else {
			log.Println("Could not send the welcome text to", phone, ":", err)
		}
		return
	}
	log.Println("Welcome text sent to", phone)
}
