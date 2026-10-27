// Package mailer sends plain-text email over SMTP, e.g. a Gmail account with an
// app password (smtp.gmail.com:587).
package mailer

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

type SMTP struct {
	Host     string
	Port     string // 587 (STARTTLS) or 465 (implicit TLS)
	Username string
	Password string
	From     string // defaults to Username
	FromName string
}

func (m *SMTP) Configured() bool {
	return m != nil && m.Host != "" && m.Username != "" && m.Password != ""
}

// Send delivers one message. to must already be a validated bare address.
func (m *SMTP) Send(ctx context.Context, to, subject, body string) error {
	if !m.Configured() {
		return errors.New("SMTP is not configured")
	}
	from := m.From
	if from == "" {
		from = m.Username
	}
	if strings.ContainsAny(to+subject, "\r\n") {
		return errors.New("invalid header value")
	}
	msg, err := m.message(from, to, subject, body)
	if err != nil {
		return err
	}

	deadline := time.Now().Add(20 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	addr := net.JoinHostPort(m.Host, m.Port)
	dialer := &net.Dialer{Deadline: deadline}
	tlsConfig := &tls.Config{ServerName: m.Host, MinVersion: tls.VersionTLS12}
	var conn net.Conn
	if m.Port == "465" {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return err
	}
	_ = conn.SetDeadline(deadline)
	c, err := smtp.NewClient(conn, m.Host)
	if err != nil {
		conn.Close()
		return err
	}
	defer c.Close()
	if m.Port != "465" {
		// Never send the password over a plain connection.
		if ok, _ := c.Extension("STARTTLS"); !ok {
			return errors.New("SMTP server does not offer STARTTLS")
		}
		if err := c.StartTLS(tlsConfig); err != nil {
			return err
		}
	}
	if err := c.Auth(smtp.PlainAuth("", m.Username, m.Password, m.Host)); err != nil {
		return err
	}
	if err := c.Mail(from); err != nil {
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

func (m *SMTP) message(from, to, subject, body string) ([]byte, error) {
	id := make([]byte, 12)
	if _, err := rand.Read(id); err != nil {
		return nil, err
	}
	domain := from[strings.LastIndex(from, "@")+1:]
	sender := (&mail.Address{Name: m.FromName, Address: from}).String()
	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", sender)
	fmt.Fprintf(&b, "To: %s\r\n", to)
	fmt.Fprintf(&b, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", subject))
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	fmt.Fprintf(&b, "Message-ID: <%s@%s>\r\n", hex.EncodeToString(id), domain)
	b.WriteString("MIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Transfer-Encoding: 8bit\r\n\r\n")
	b.WriteString(strings.ReplaceAll(strings.ReplaceAll(body, "\r\n", "\n"), "\n", "\r\n"))
	return []byte(b.String()), nil
}
