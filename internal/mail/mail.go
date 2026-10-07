package mail

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/nodemode-ai/form.md/internal/settings"
)

type Sender interface {
	Send(ctx context.Context, from, to string, msg []byte) error
}

type Mailer struct {
	set   *settings.Service
	oauth map[string]Sender
}

func New(set *settings.Service) *Mailer {
	return &Mailer{set: set, oauth: map[string]Sender{}}
}

func (m *Mailer) Register(mode string, s Sender) { m.oauth[mode] = s }

func (m *Mailer) Enabled() bool {
	if m.set.Get("mail.from") == "" {
		return false
	}
	switch mode := m.set.Get("mail.mode"); mode {
	case "smtp":
		return m.set.Get("mail.smtp.host") != ""
	case "gmail", "graph":
		return m.oauth[mode] != nil && m.set.Get("mail.oauth.refresh_token") != ""
	}
	return false
}

func header(v string) string {
	return mime.QEncoding.Encode("utf-8", v)
}

func Build(from, to, subject, body string) ([]byte, error) {
	f, err := mail.ParseAddress(from)
	if err != nil {
		return nil, fmt.Errorf("mail.from: %w", err)
	}
	if _, err := mail.ParseAddress(to); err != nil {
		return nil, fmt.Errorf("recipient: %w", err)
	}
	id := make([]byte, 12)
	_, _ = rand.Read(id)
	_, domain, _ := strings.Cut(f.Address, "@")
	var b bytes.Buffer
	fmt.Fprintf(&b, "From: %s\r\n", f.String())
	fmt.Fprintf(&b, "To: %s\r\n", to)
	fmt.Fprintf(&b, "Subject: %s\r\n", header(subject))
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	fmt.Fprintf(&b, "Message-ID: <%s@%s>\r\n", hex.EncodeToString(id), domain)
	b.WriteString("MIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n")
	qp := quotedprintable.NewWriter(&b)
	if _, err := qp.Write([]byte(strings.ReplaceAll(body, "\n", "\r\n"))); err != nil {
		return nil, err
	}
	if err := qp.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (m *Mailer) Send(ctx context.Context, to, subject, body string) error {
	if !m.Enabled() {
		return errors.New("mail is not configured")
	}
	from := m.set.Get("mail.from")
	msg, err := Build(from, to, subject, body)
	if err != nil {
		return err
	}
	if mode := m.set.Get("mail.mode"); mode != "smtp" {
		return m.oauth[mode].Send(ctx, from, to, msg)
	}
	return m.smtp(ctx, from, to, msg)
}

func (m *Mailer) smtp(ctx context.Context, from, to string, msg []byte) error {
	host := m.set.Get("mail.smtp.host")
	port := m.set.Int("mail.smtp.port")
	security := m.set.Get("mail.smtp.security")
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	d := &net.Dialer{}
	var conn net.Conn
	var err error
	tlsCfg := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
	if security == "tls" {
		conn, err = (&tls.Dialer{NetDialer: d, Config: tlsCfg}).DialContext(ctx, "tcp", addr)
	} else {
		conn, err = d.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return err
	}
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return err
	}
	defer c.Close()
	if security == "starttls" {
		if err := c.StartTLS(tlsCfg); err != nil {
			return fmt.Errorf("starttls: %w", err)
		}
	}
	if user := m.set.Get("mail.smtp.user"); user != "" {
		if err := c.Auth(smtp.PlainAuth("", user, m.set.Get("mail.smtp.password"), host)); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	f, _ := mail.ParseAddress(from)
	if err := c.Mail(f.Address); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}
