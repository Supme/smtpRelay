package model

import (
	"encoding/base64"
	_ "github.com/denisenkom/go-mssqldb" // MSSQL driver
	_ "github.com/go-sql-driver/mysql"   // MySQL driver
	"github.com/go-xorm/xorm"
	_ "github.com/lib/pq"           // Postgres driver
	_ "github.com/mattn/go-sqlite3" // SQLite driver
	"github.com/XS4ALL/go-smtpd/smtpd"
	"log"
	"strings"
	"time"
)

// Config application config
var Config struct {
	QueueDbDialect        string   `toml:"QueueDbDialect"`
	QueueDbConnect        string   `toml:"QueueDbConnect"`
	StatusDbDialect       string   `toml:"StatusDbDialect"`
	StatusDbConnect       string   `toml:"StatusDbConnect"`
	Hostname              string   `toml:"Hostname"`
	AllowIP               []string `toml:"AllowIP"`
	AllowDomains          []string `toml:"AllowDomains"`
	SMTPListenAddr        string   `toml:"SMTPListenAddr"`
	MaxRepeatSend         uint     `toml:"MaxRepeatSend"`
	RepeatIntervalMinutes uint     `toml:"RepeatIntervalMinutes"`
	SendStream            uint     `toml:"SendStream"`
	ResendStream          uint     `toml:"ResendStream"`
	Debug                 bool     `toml:"Debug"`
}

var (
	// QueueDb queue db connection
	QueueDb *xorm.Engine
	// StatusDb status db connection
	StatusDb *xorm.Engine
)

// Queue queue email model
type Queue struct {
	ID           uint64    `xorm:"serial pk autoincr 'id'"`
	CreatedAt    time.Time `xorm:"created"`
	UpdatedAt    time.Time `xorm:"updated"`
	MessageType  string
	MessageID    string `xorm:"'message_id'"`
	From         string
	FromHostname string
	Rcpt         string
	RcptHostname string
	Data         string `xorm:"LONGTEXT 'data'"`
	Repeat       uint
	LaterStatus  string `xorm:"MEDIUMTEXT 'later_status'"`
}

type status struct {
	ID          uint64 `xorm:"BIGSERIAL pk autoincr 'id'"`
	QueuedAt    time.Time
	SendingAt   time.Time `xorm:"created"`
	From        string
	Rcpt        string
	MessageType string
	MessageID   string `xorm:"'message_id'"`
	Status      string `xorm:"MEDIUMTEXT 'status'"`
}

// OpenQueueDb open queue database
func OpenQueueDb() (err error) {
	QueueDb, err = xorm.NewEngine(Config.StatusDbDialect, Config.StatusDbConnect)
	if err != nil {
		return
	}
	QueueDb.ShowSQL(Config.Debug)
	err = QueueDb.Sync2(new(Queue))
	return err
}

// OpenStatusDb open status database
func OpenStatusDb() (err error) {
	StatusDb, err = xorm.NewEngine(Config.StatusDbDialect, Config.StatusDbConnect)
	if err != nil {
		return
	}
	StatusDb.ShowSQL(Config.Debug)
	err = StatusDb.Sync2(new(status))
	return err
}

// AddToQueue add email to queue
func AddToQueue(messageType, messageID string, from smtpd.MailAddress, rcpts []smtpd.MailAddress, data []byte) error {
	session := QueueDb.NewSession()
	defer session.Close()
	if err := session.Begin(); err != nil {
		log.Println(err)
		return err
	}
	for _, rcpt := range rcpts {
		if _, err := session.InsertOne(&Queue{
			MessageType:  messageType,
			MessageID:    messageID,
			From:         from.Email(),
			FromHostname: from.Hostname(),
			Rcpt:         rcpt.Email(),
			RcptHostname: rcpt.Hostname(),
			Data:         base64.StdEncoding.EncodeToString(data),
		}); err != nil {
			log.Println(err)
			session.Rollback()
			return err
		}
	}
	return session.Commit()
}

// GetRepeatQueue get `limit` number emails for resend
func GetRepeatQueue(limit uint) []Queue {
	var emails []Queue
	if err := QueueDb.Where("updated_at < ? AND repeat > 0",
		time.Now().Local().Add(-1*time.Minute*time.Duration(Config.RepeatIntervalMinutes))).
		Limit(int(limit)).
		Find(&emails); err != nil {
		log.Print(err)
	}
	return emails
}

// GetNewQueue get `limit` number new emails
func GetNewQueue(limit uint) []Queue {
	var emails []Queue
	if err := QueueDb.Where("repeat=0").
		Limit(int(limit)).
		Find(&emails); err != nil {
		log.Print(err)
	}
	return emails
}

// SetStatus set email send status
func SetStatus(email *Queue) {
	email.Repeat++
	if email.Repeat >= Config.MaxRepeatSend {
		setStatus(email)
	} else {
		if strings.HasPrefix(email.LaterStatus, "4") {
			if _, err := QueueDb.Table(new(Queue)).
				ID(email.ID).Update(&Queue{
				Repeat:      email.Repeat,
				LaterStatus: email.LaterStatus,
			}); err != nil {
				log.Print(err)
			}
		} else {
			setStatus(email)
		}
	}
}

func setStatus(email *Queue) {
	if _, err := StatusDb.InsertOne(&status{
		QueuedAt:    email.CreatedAt,
		SendingAt:   time.Now(),
		From:        email.From,
		Rcpt:        email.Rcpt,
		MessageType: email.MessageType,
		MessageID:   email.MessageID,
		Status:      email.LaterStatus,
	}); err != nil {
		log.Print(err)
	}
	if _, err := QueueDb.Delete(&Queue{ID: email.ID}); err != nil {
		log.Print(err)
	}
}
