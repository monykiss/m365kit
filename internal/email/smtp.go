// Package email provides SMTP email sending for the kit send command.
package email

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net"
	"net/smtp"
	"net/textproto"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Config holds SMTP connection settings.
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// Message holds the email content and recipients.
type Message struct {
	To      []string
	CC      []string
	Subject string
	Body    string
	Attach  string // file path
}

// LoadConfig reads SMTP settings from environment variables.
func LoadConfig() (Config, error) {
	host := os.Getenv("KIT_SMTP_HOST")
	port := os.Getenv("KIT_SMTP_PORT")
	username := os.Getenv("KIT_SMTP_USERNAME")
	password := os.Getenv("KIT_SMTP_PASSWORD")
	from := os.Getenv("KIT_SMTP_FROM")

	if host == "" || username == "" || password == "" || from == "" {
		missing := []string{}
		if host == "" {
			missing = append(missing, "KIT_SMTP_HOST")
		}
		if username == "" {
			missing = append(missing, "KIT_SMTP_USERNAME")
		}
		if password == "" {
			missing = append(missing, "KIT_SMTP_PASSWORD")
		}
		if from == "" {
			missing = append(missing, "KIT_SMTP_FROM")
		}
		return Config{}, fmt.Errorf("No SMTP configuration found. Set %s environment variables.", strings.Join(missing, ", "))
	}

	p := 587
	if port != "" {
		var err error
		p, err = strconv.Atoi(port)
		if err != nil {
			return Config{}, fmt.Errorf("invalid KIT_SMTP_PORT %q: must be a number", port)
		}
	}

	if !ValidateEmail(from) {
		return Config{}, fmt.Errorf("invalid KIT_SMTP_FROM address: %q", from)
	}

	return Config{
		Host:     host,
		Port:     p,
		Username: username,
		Password: password,
		From:     from,
	}, nil
}

// Validate returns an error if the message is malformed.
func (m *Message) Validate() error {
	if len(m.To) == 0 {
		return fmt.Errorf("no recipients specified — use --to")
	}
	for _, addr := range m.To {
		if !ValidateEmail(addr) {
			return fmt.Errorf("invalid recipient email address: %q", addr)
		}
	}
	for _, addr := range m.CC {
		if !ValidateEmail(addr) {
			return fmt.Errorf("invalid CC email address: %q", addr)
		}
	}
	if m.Attach != "" {
		if _, err := os.Stat(m.Attach); os.IsNotExist(err) {
			return fmt.Errorf("attachment not found: %s", m.Attach)
		}
	}
	return nil
}

// AttachSize returns the size of the attachment file in bytes, or 0 if no attachment.
func (m *Message) AttachSize() int64 {
	if m.Attach == "" {
		return 0
	}
	info, err := os.Stat(m.Attach)
	if err != nil {
		return 0
	}
	return info.Size()
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// ValidateEmail returns true if s looks like a valid email address.
func ValidateEmail(s string) bool {
	return emailRegex.MatchString(s)
}

// Send sends a MIME multipart email with an optional attachment.
func Send(cfg Config, msg Message) error {
	if err := msg.Validate(); err != nil {
		return err
	}

	mimeBody, boundary, err := buildMIME(cfg, msg)
	if err != nil {
		return fmt.Errorf("could not build email: %w", err)
	}

	allRecipients := make([]string, 0, len(msg.To)+len(msg.CC))
	allRecipients = append(allRecipients, msg.To...)
	allRecipients = append(allRecipients, msg.CC...)

	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)

	headers := buildHeaders(cfg, msg, boundary)
	var fullMsg bytes.Buffer
	fullMsg.WriteString(headers)
	fullMsg.Write(mimeBody)

	if cfg.Port == 465 {
		return sendTLS(addr, auth, cfg.From, allRecipients, fullMsg.Bytes())
	}
	return smtp.SendMail(addr, auth, cfg.From, allRecipients, fullMsg.Bytes())
}

func buildHeaders(cfg Config, msg Message, boundary string) string {
	var h strings.Builder
	h.WriteString(fmt.Sprintf("From: %s\r\n", cfg.From))
	h.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(msg.To, ", ")))
	if len(msg.CC) > 0 {
		h.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(msg.CC, ", ")))
	}
	h.WriteString(fmt.Sprintf("Subject: %s\r\n", msg.Subject))
	h.WriteString("MIME-Version: 1.0\r\n")
	h.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n", boundary))
	h.WriteString("\r\n")
	return h.String()
}

func buildMIME(cfg Config, msg Message) ([]byte, string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Text body part
	textHeader := make(textproto.MIMEHeader)
	textHeader.Set("Content-Type", "text/plain; charset=utf-8")
	textHeader.Set("Content-Transfer-Encoding", "quoted-printable")
	part, err := writer.CreatePart(textHeader)
	if err != nil {
		return nil, "", err
	}
	if _, err := part.Write([]byte(msg.Body)); err != nil {
		return nil, "", err
	}

	// Attachment part
	if msg.Attach != "" {
		data, err := os.ReadFile(msg.Attach)
		if err != nil {
			return nil, "", fmt.Errorf("could not read attachment %s: %w", msg.Attach, err)
		}

		filename := filepath.Base(msg.Attach)
		attachHeader := make(textproto.MIMEHeader)
		attachHeader.Set("Content-Type", "application/octet-stream")
		attachHeader.Set("Content-Transfer-Encoding", "base64")
		attachHeader.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
		part, err := writer.CreatePart(attachHeader)
		if err != nil {
			return nil, "", err
		}

		encoded := base64.StdEncoding.EncodeToString(data)
		// Split into 76-char lines per RFC 2045
		for i := 0; i < len(encoded); i += 76 {
			end := i + 76
			if end > len(encoded) {
				end = len(encoded)
			}
			if _, err := part.Write([]byte(encoded[i:end] + "\r\n")); err != nil {
				return nil, "", err
			}
		}
	}

	boundary := writer.Boundary()
	if err := writer.Close(); err != nil {
		return nil, "", err
	}

	return buf.Bytes(), boundary, nil
}

func sendTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{})
	if err != nil {
		return fmt.Errorf("TLS connection failed: %w", err)
	}
	defer conn.Close()

	host, _, _ := net.SplitHostPort(addr)
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("SMTP client creation failed: %w", err)
	}
	defer client.Close()

	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP authentication failed: %w", err)
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, addr := range to {
		if err := client.Rcpt(addr); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	return w.Close()
}
