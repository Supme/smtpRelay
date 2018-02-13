package model

import (
	"encoding/base64"
	"fmt"
	"github.com/XS4ALL/go-smtpd/smtpd"
	_ "github.com/denisenkom/go-mssqldb" // MSSQL driver
	_ "github.com/go-sql-driver/mysql"   // MySQL driver
	"github.com/go-xorm/xorm"
	_ "github.com/lib/pq"           // Postgres driver
	_ "github.com/mattn/go-sqlite3" // SQLite driver
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

	whereInterval string
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
	QueueDb, err = xorm.NewEngine(Config.QueueDbDialect, Config.QueueDbConnect)
	if err != nil {
		return
	}
	QueueDb.ShowSQL(Config.Debug)
	QueueDb.SetTZLocation(time.Local)
	QueueDb.SetTZDatabase(time.Local)
	whereInterval = createWhereInterval()

	return QueueDb.Sync2(new(Queue))
}

// OpenStatusDb open status database
func OpenStatusDb() (err error) {
	StatusDb, err = xorm.NewEngine(Config.StatusDbDialect, Config.StatusDbConnect)
	if err != nil {
		return
	}
	StatusDb.ShowSQL(Config.Debug)
	StatusDb.SetTZLocation(time.Local)
	StatusDb.SetTZDatabase(time.Local)
	return StatusDb.Sync2(new(status))
}

func createWhereInterval() string {
	var where string
	switch QueueDb.Dialect().DriverName() {
	case "sqlite3":
		where = "updated_at<DATETIME('NOW', '-%d Minute')"
	case "mssql":
		where = "updated_at<DATEADD(mi, -%d, getdate())"
	case "mysql":
		where = "updated_at<NOW() - INTERVAL %d MINUTE)"
	case "postgres":
		where = "updated_at<now()::time - INTERVAL '%d min'"
	default:
		log.Fatal("unsuported database driver")
	}
	return fmt.Sprintf(where, Config.RepeatIntervalMinutes)
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
		if _, err := session.Query(`
INSERT INTO "queue"
  ("created_at","updated_at","message_type","message_id","from","from_hostname","rcpt","rcpt_hostname","data","repeat","later_status")
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0, NULL)`,
			time.Now(),
			time.Now(),
			messageType,
			messageID,
			from.Email(),
			from.Hostname(),
			rcpt.Email(),
			rcpt.Hostname(),
			base64.StdEncoding.EncodeToString(data)); err != nil {
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
	if err := QueueDb.Where(whereInterval).
		And("repeat > 0").
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
			if _, err := QueueDb.Exec(`UPDATE "queue" SET "repeat" = ?, "later_status" = ?, "updated_at" = ? WHERE "id"=?`,
				email.Repeat,
				email.LaterStatus,
				time.Now(),
				email.ID,
			); err != nil {
				log.Print(err)
			}
		} else {
			setStatus(email)
		}
	}
}

func setStatus(email *Queue) {
	if _, err := StatusDb.Query(`
INSERT INTO "status"
  ("queued_at","sending_at","from","rcpt","message_type","message_id","status")
VALUES (?, ?, ?, ?, ?, ?, ?)`,
		email.CreatedAt,
		time.Now(),
		email.From,
		email.Rcpt,
		email.MessageType,
		email.MessageID,
		email.LaterStatus); err != nil {
		log.Print(err)
	}
	if _, err := QueueDb.Delete(&Queue{ID: email.ID}); err != nil {
		log.Print(err)
	}
}
