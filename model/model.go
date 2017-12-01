package model

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/mssql"
	"github.com/sfreiberg/go-smtpd/smtpd"
	"log"
	"time"
)

var (
	QueueDb               *gorm.DB
	StatusDb              *gorm.DB
	Hostname              string
	MaxRepeatSend         uint
	RepeatIntervalMinutes uint
)

type Email struct {
	ID           uint
	MessageID    string
	From         string
	Rcpt         string
	RcptHostname string
	Data         []byte
	Repeat       uint
}

type queue struct {
	ID           uint `gorm:"primary_key"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	MessageID    string
	From         string
	FromHostname string
	Rcpt         string
	RcptHostname string
	Data         []byte
	Repeat       uint
	LaterStatus  string
}

type status struct {
	ID        uint `gorm:"primary_key"`
	From      string
	Rcpt      string
	MessageID string
	UpdatedAt time.Time
	Status    string
}

func OpenQueueDb(dialect string, conn string) (err error) {
	QueueDb, err = gorm.Open(dialect, conn)
	if err != nil {
		return
	}
	QueueDb.AutoMigrate(&queue{})
	//	QueueDb.LogMode(true)
	return
}

func OpenStatusDb(dialect string, conn string) (err error) {
	StatusDb, err = gorm.Open(dialect, conn)
	if err != nil {
		return
	}
	StatusDb.AutoMigrate(&status{})
	//	QueueDb.LogMode(true)
	return
}

func AddToQueue(messageID string, from smtpd.MailAddress, rcpts []smtpd.MailAddress, data []byte) {
	for _, rcpt := range rcpts {
		QueueDb.Create(&queue{
			MessageID:    messageID,
			From:         from.Email(),
			FromHostname: from.Hostname(),
			Rcpt:         rcpt.Email(),
			RcptHostname: rcpt.Hostname(),
			Data:         data,
		})
	}
}

func GetRepeatQueue(limit uint) []Email {
	var (
		emails []Email
	)
	rows, err := QueueDb.Model(queue{}).
		Select("`id`, `message_id`, `from`, `rcpt`, `rcpt_hostname`, `data`, `repeat`").
		Where("updated_at < ? AND repeat>0", time.Now().Local().Add(-1*time.Minute*time.Duration(RepeatIntervalMinutes))).
		Limit(int(limit)).
		Rows()
	if err != nil {
		log.Print(err)
	}
	defer rows.Close()

	for rows.Next() {
		var email Email
		rows.Scan(&email.ID, &email.MessageID, &email.From, &email.Rcpt, &email.RcptHostname, &email.Data, &email.Repeat)
		emails = append(emails, email)
	}

	return emails
}

func GetNewQueue(limit uint) []Email {
	var (
		emails []Email
	)
	rows, err := QueueDb.Model(queue{}).
		Select("`id`, `message_id`, `from`, `rcpt`, `rcpt_hostname`, `data`, `repeat`").
		Where("repeat=0").
		Limit(int(limit)).
		Rows()
	if err != nil {
		log.Print(err)
	}
	defer rows.Close()

	for rows.Next() {
		var email Email
		rows.Scan(&email.ID, &email.MessageID, &email.From, &email.Rcpt, &email.RcptHostname, &email.Data, &email.Repeat)
		emails = append(emails, email)
	}

	return emails
}

func SetErrorStatus(queueID uint, message string) {
	if message[0:1] != "4" {
		QueueDb.Model(&queue{ID: queueID}).UpdateColumn("repeat", gorm.Expr("repeat + ?", 1))
		QueueDb.Model(&queue{ID: queueID}).UpdateColumn("later_status", message)
	} else {
		QueueDb.Delete(&queue{ID: queueID})
	}
}

func SetStatus(queueID uint, from, rcpt, messageID, message string) {
	StatusDb.Create(&status{
		From:      from,
		Rcpt:      rcpt,
		MessageID: messageID,
		Status:    message,
	})
	QueueDb.Delete(&queue{ID: queueID})
}
