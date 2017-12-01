package sender

import (
	"github.com/supme/directEmail"
	"github.com/supme/smtpRelay/model"
	"sync"
	"time"
)

func Run() {
	go sendQueue(model.GetNewQueue)
	go sendQueue(model.GetRepeatQueue)
}

func sendQueue(getFunc func(uint) []model.Email) {
	for {
		emails := getFunc(10) // limit max count
		if len(emails) == 0 {
			time.Sleep(time.Second * 10)
		}
		var wg sync.WaitGroup
		for i := range emails {
			wg.Add(1)
			go func() {
				sendEmail(&emails[i])
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

func sendEmail(email *model.Email) {
	sender := directEmail.New()
	sender.Host = model.Hostname
	sender.FromEmail = email.From
	sender.ToEmail = email.Rcpt
	sender.SetRawMessageBytes(email.Data)
	err := sender.Send()
	if err != nil {
		if email.Repeat >= model.MaxRepeatSend {
			model.SetStatus(email.ID, email.From, email.Rcpt, email.MessageID, err.Error())
		} else {
			model.SetErrorStatus(email.ID, err.Error())
		}
	} else {
		model.SetStatus(email.ID, email.From, email.Rcpt, email.MessageID, "Ok")
	}

}
