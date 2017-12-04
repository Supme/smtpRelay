package server

import (
	"bytes"
	"github.com/pingcap/tidb/_vendor/src/github.com/juju/errors"
	"github.com/sfreiberg/go-smtpd/smtpd"
	"github.com/supme/smtpRelay/model"
	"log"
	"regexp"
)

var messageIdRE = regexp.MustCompile(`[Mm][Ee][Ss][Ss][Aa][Gg][Ee].?[Ii][Dd]:\s*<(.+)>`)

type env struct {
	rcpts     []smtpd.MailAddress
	from      smtpd.MailAddress
	data      bytes.Buffer
	messageID string
}

func (e *env) AddRecipient(rcpt smtpd.MailAddress) error {
	e.rcpts = append(e.rcpts, rcpt)
	return nil
}

func (e *env) BeginData() error {
	if len(e.rcpts) == 0 {
		return smtpd.SMTPError("554 5.5.1 Error: no valid recipients")
	}
	return nil
}

func (e *env) Write(line []byte) error {
	m := messageIdRE.FindSubmatch(line)
	if m != nil && len(m) == 2 {
		e.messageID = string(m[1])
	}
	_, err := e.data.Write(line)
	return err
}

func (e *env) Close() error {
	model.AddToQueue(e.messageID, e.from, e.rcpts, e.data.Bytes())
	return nil
}

func onNewMail(c smtpd.Connection, from smtpd.MailAddress) (smtpd.Envelope, error) {
	if !model.CheckAllow(c.Addr().String(), from.Hostname()) {
		return new(env), errors.New("Access denied")
	}
	env := new(env)
	env.from = from
	return env, nil
}

func Run() {
	s := &smtpd.Server{
		Hostname:  model.Config.Hostname,
		Addr:      model.Config.SmtpListenAddr,
		OnNewMail: onNewMail,
	}
	log.Printf("Starting SMTP server on %s", model.Config.SmtpListenAddr)
	err := s.ListenAndServe()
	if err != nil {
		log.Fatalf("ListenAndServe: %v", err)
	}
}
