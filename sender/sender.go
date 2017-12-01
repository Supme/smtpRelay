package sender

import (
	"github.com/supme/directEmail"
	"github.com/supme/smtpRelay/model"
	"sync"
	"time"
)

func Run() {
	//go sendQueue(model.GetNewQueue)
	//go sendQueue(model.GetRepeatQueue)
	go sendQueue()
}

//func sendQueue(getFunc func(uint) []model.Email) {
func sendQueue() {
	for {
		emails := model.GetNewQueue(10) // limit max count
		if len(emails) == 0 {
			emails = model.GetRepeatQueue(10)
			if len(emails) == 0 {
				time.Sleep(10 * time.Second)
			}
		}
		var wg sync.WaitGroup
		for i := range emails {
			wg.Add(1)
			go func(wg *sync.WaitGroup, email model.Email) {
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
					model.SetStatus(email.ID, email.From, email.Rcpt, email.MessageID, "250 2.0.0 Ok")
				}
				wg.Done()
			}(&wg, emails[i])
		}
		wg.Wait()
	}
}
