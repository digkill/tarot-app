package mailer

import (
	"crypto/tls"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"time"
)

type Config struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
	FromName string
}

type Sender struct {
	cfg Config
}

func New(cfg Config) *Sender {
	return &Sender{cfg: cfg}
}

func (s *Sender) Enabled() bool {
	return s != nil && s.cfg.Host != "" && s.cfg.Username != "" && s.cfg.Password != "" && s.cfg.From != ""
}

func (s *Sender) Send(to, subject, plain, htmlBody string) error {
	if !s.Enabled() {
		return fmt.Errorf("smtp is not configured")
	}

	from := s.cfg.From
	msg := buildMessage(s.cfg.FromName, from, to, subject, plain, htmlBody)
	addr := net.JoinHostPort(s.cfg.Host, s.cfg.Port)
	dialer := &net.Dialer{Timeout: 20 * time.Second}

	if s.cfg.Port == "465" {
		err := sendImplicitTLS(dialer, addr, s.cfg, from, to, msg, "PLAIN")
		if err != nil {
			if err2 := sendImplicitTLS(dialer, addr, s.cfg, from, to, msg, "LOGIN"); err2 == nil {
				return nil
			}
			return err
		}
		return nil
	}

	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("smtp client: %w", err)
	}
	defer func() { _ = client.Close() }()
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err = client.StartTLS(&tls.Config{ServerName: s.cfg.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}
	return sendWithClient(client, s.cfg.Username, s.cfg.Password, s.cfg.Host, from, to, msg, "")
}

func sendImplicitTLS(dialer *net.Dialer, addr string, cfg Config, from, to string, msg []byte, method string) error {
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12})
	if err != nil {
		return fmt.Errorf("smtp tls dial: %w", err)
	}
	defer conn.Close()
	client, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer func() { _ = client.Close() }()
	return sendWithClient(client, cfg.Username, cfg.Password, cfg.Host, from, to, msg, method)
}

func sendWithClient(client *smtp.Client, username, password, host, from, to string, msg []byte, forceMethod string) error {
	if err := authenticate(client, username, password, host, forceMethod); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err = w.Write(msg); err != nil {
		_ = w.Close()
		return fmt.Errorf("smtp write: %w", err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}
	return client.Quit()
}

func authenticate(client *smtp.Client, username, password, host, forceMethod string) error {
	ok, methods := client.Extension("AUTH")
	if !ok {
		return errors.New("server does not support AUTH")
	}
	upper := strings.ToUpper(methods)
	useLogin := forceMethod == "LOGIN" || (forceMethod == "" && !strings.Contains(upper, "PLAIN") && strings.Contains(upper, "LOGIN"))
	if forceMethod == "PLAIN" && !strings.Contains(upper, "PLAIN") {
		useLogin = strings.Contains(upper, "LOGIN")
	}
	if !useLogin {
		if forceMethod == "PLAIN" || strings.Contains(upper, "PLAIN") {
			return client.Auth(&plainAuth{username: username, password: password, host: host})
		}
	}
	if useLogin || strings.Contains(upper, "LOGIN") {
		return client.Auth(&loginAuth{username: username, password: password})
	}
	return fmt.Errorf("unsupported AUTH methods %q", methods)
}

// plainAuth is AUTH PLAIN without net/smtp.PlainAuth's TLS flag check.
// Needed for implicit TLS on port 465, where smtp.Client does not mark the connection as TLS.
type plainAuth struct {
	username, password, host string
}

func (a *plainAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	if server.Name != a.host {
		return "", nil, errors.New("wrong host name")
	}
	resp := []byte("\x00" + a.username + "\x00" + a.password)
	return "PLAIN", resp, nil
}

func (a *plainAuth) Next(_ []byte, more bool) ([]byte, error) {
	if more {
		return nil, errors.New("unexpected server challenge")
	}
	return nil, nil
}

type loginAuth struct {
	username, password string
}

func (a *loginAuth) Start(_ *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", nil, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	prompt := strings.ToLower(strings.TrimSpace(string(fromServer)))
	switch {
	case strings.Contains(prompt, "username"):
		return []byte(a.username), nil
	case strings.Contains(prompt, "password"):
		return []byte(a.password), nil
	default:
		return nil, fmt.Errorf("unexpected smtp login challenge")
	}
}

func buildMessage(fromName, from, to, subject, plain, htmlBody string) []byte {
	boundary := "tarot-alt"
	var b strings.Builder
	if fromName != "" {
		fmt.Fprintf(&b, "From: %s <%s>\r\n", mimeHeader(fromName), from)
	} else {
		fmt.Fprintf(&b, "From: %s\r\n", from)
	}
	fmt.Fprintf(&b, "To: %s\r\n", to)
	fmt.Fprintf(&b, "Subject: %s\r\n", mimeHeader(subject))
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().UTC().Format(time.RFC1123Z))
	b.WriteString("MIME-Version: 1.0\r\n")
	fmt.Fprintf(&b, "Content-Type: multipart/alternative; boundary=%s\r\n", boundary)
	b.WriteString("\r\n")
	fmt.Fprintf(&b, "--%s\r\n", boundary)
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
	b.WriteString(plain)
	b.WriteString("\r\n")
	fmt.Fprintf(&b, "--%s\r\n", boundary)
	b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
	b.WriteString(htmlBody)
	b.WriteString("\r\n")
	fmt.Fprintf(&b, "--%s--\r\n", boundary)
	return []byte(b.String())
}

func mimeHeader(s string) string {
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\r", " "), "\n", " ")
	return mime.QEncoding.Encode("UTF-8", s)
}
