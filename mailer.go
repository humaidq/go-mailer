// Package mailer is a simple e-mail sender for Go Programming Language
package mailer

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/mail"
	"net/smtp"
	"os/exec"
	"strings"

	"github.com/valyala/bytebufferpool"
)

const (
	// Version current version semantic number of the "go-mailer" package.
	Version = "0.0.2"
)

// Mailer is the main struct which contains the nessecary fields
// for sending emails, either with unix command "sendmail"
// or by following the configuration's properties.
type Mailer struct {
	config        Config
	fromAddr      mail.Address
	auth          smtp.Auth
	authenticated bool
}

// New creates and returns a new mail sender.
func New(cfg Config) *Mailer {
	m := &Mailer{config: cfg}
	addr := cfg.FromAddr
	if addr == "" {
		addr = cfg.Username
	}

	if cfg.FromAlias == "" {
		if !cfg.UseCommand && cfg.Username != "" && strings.Contains(cfg.Username, "@") {
			m.fromAddr = mail.Address{Name: cfg.Username[0:strings.IndexByte(cfg.Username, '@')], Address: addr}
		}
	} else {
		m.fromAddr = mail.Address{Name: cfg.FromAlias, Address: addr}
	}
	return m
}

// UpdateConfig overrides the current configuration.
func (m *Mailer) UpdateConfig(cfg Config) {
	m.config = cfg
}

// Send sends an email to the recipient(s)
// the body can be in HTML format as well.
//
// Note: you can change the UseCommand in runtime.
func (m *Mailer) Send(subject string, body string, to ...string) error {
	if m.config.UseCommand {
		return m.sendCmd(subject, body, to)
	}

	return m.sendSMTP(subject, body, to)
}

const (
	contentTypeHTML         = `text/html; charset=\"utf-8\"`
	mimeVer                 = "1.0"
	contentTransferEncoding = "base64"
)

var buf bytebufferpool.Pool

func (m *Mailer) sendSMTP(subject string, body string, to []string) error {
	buffer := buf.Get()
	defer buf.Put(buffer)

	if !m.authenticated {
		cfg := m.config
		if cfg.Username == "" || cfg.Password == "" || cfg.Host == "" || cfg.Port <= 0 {
			return fmt.Errorf("username, password, host or port missing")
		}
		m.auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
		m.authenticated = true
	}

	fullhost := fmt.Sprintf("%s:%d", m.config.Host, m.config.Port)

	header := make(map[string]string, 6)
	header["From"] = m.fromAddr.String()
	header["To"] = strings.Join(to, ",")
	header["Subject"] = subject
	header["MIME-Version"] = mimeVer
	header["Content-Type"] = contentTypeHTML
	header["Content-Transfer-Encoding"] = contentTransferEncoding

	for k, v := range header {
		buffer.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}

	buffer.WriteString("\r\n" + base64.StdEncoding.EncodeToString([]byte(body)))

	return smtp.SendMail(
		fmt.Sprintf(fullhost),
		m.auth,
		m.config.Username,
		to,
		buffer.Bytes(),
	)
}

func (m *Mailer) sendCmd(subject string, body string, to []string) error {
	buffer := new(bytes.Buffer)

	header := make(map[string]string, 5)
	header["To"] = strings.Join(to, ",")
	header["Subject"] = subject
	header["MIME-Version"] = mimeVer
	header["Content-Type"] = contentTypeHTML
	header["Content-Transfer-Encoding"] = contentTransferEncoding

	for k, v := range header {
		buffer.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	buffer.WriteString("\r\n" + base64.StdEncoding.EncodeToString([]byte(body)))

	cmd := exec.Command("sendmail", "-F", m.fromAddr.Name, "-f", m.fromAddr.Address, "-t")
	cmd.Stdin = buffer
	_, err := cmd.CombinedOutput()
	return err
}
