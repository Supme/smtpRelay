package server

import (
	"bytes"
	"errors"
	"github.com/XS4ALL/go-smtpd/smtpd"
	"github.com/supme/smtpRelay/model"
	"log"
	"regexp"
)

var (
	messageIDre   = regexp.MustCompile(`[Mm][Ee][Ss][Ss][Aa][Gg][Ee].?[Ii][Dd]:\s*(.+)`)
	messageTypeRe = regexp.MustCompile(`[Xx].?[Pp][Oo][Ss][Tt][Mm][Aa][Ss][Tt][Ee][Rr].?[Mm][Ss][Gg][Tt][Yy][Pp][Ee]:\s*(.+)`)
)

type env struct {
	rcpts       []smtpd.MailAddress
	from        smtpd.MailAddress
	data        bytes.Buffer
	messageID   string
	messageType string
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
	m := messageIDre.FindSubmatch(line)
	if m != nil && len(m) == 2 {
		e.messageID = string(m[1])
	}
	t := messageTypeRe.FindSubmatch(line)
	if t != nil && len(t) == 2 {
		e.messageType = string(t[1])
	}
	_, err := e.data.Write(line)
	if err != nil {
		return smtpd.SMTPError("550 " + err.Error())
	}
	return nil
}

func (e *env) Close() error {
	err := model.AddToQueue(e.messageType, e.messageID, e.from, e.rcpts, e.data.Bytes())
	e.data.Reset()
	if err != nil {
		return smtpd.SMTPError("452 Internal server error")
	}
	return nil
}

func onNewMail(c smtpd.Connection, from smtpd.MailAddress) (smtpd.Envelope, error) {
	if !model.CheckAllow(c.Addr().String(), from.Hostname()) {
		return new(env), errors.New("550 Access denied")
	}
	env := new(env)
	env.from = from
	return env, nil
}

// Run started SMTP server
func Run() {
	s := &smtpd.Server{
		Hostname:  model.Config.Hostname,
		Addr:      model.Config.SMTPListenAddr,
		OnNewMail: onNewMail,
	}
	s.OnProtoError = func(err error) {
		log.Print(err.Error())
	}
	log.Printf("Starting SMTP server on %s", model.Config.SMTPListenAddr)
	if err := s.ListenAndServe(); err != nil {
		log.Fatalf("ListenAndServe: %v", err)
	}
}
