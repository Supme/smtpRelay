package sender

import (
	"github.com/supme/directEmail"
	"github.com/supme/smtpRelay/model"
	"sync"
	"time"
)

// Run start sending queue emails
func Run() {
	go sendQueue()
	go resendQueue()
}

func sendQueue() {
	for {
		emails := model.GetNewQueue(model.Config.SendStream)
		if len(emails) == 0 {
			time.Sleep(10 * time.Second)
		} else {
			send(emails)
		}
	}
}

func resendQueue() {
	for {
		emails := model.GetRepeatQueue(model.Config.ResendStream)
		if len(emails) == 0 {
			time.Sleep(10 * time.Second)
		} else {
			send(emails)
		}
	}
}

func send(emails []model.Queue) {
	var wg sync.WaitGroup
	for i := range emails {
		wg.Add(1)
		go func(wg *sync.WaitGroup, email model.Queue) {
			sender := directEmail.New()
			sender.Host = model.Config.Hostname
			sender.FromEmail = email.From
			sender.ToEmail = email.Rcpt
			sender.SetRawMessageBytes(email.Data)
			err := sender.Send()
			if err != nil {
				email.LaterStatus = err.Error()
			} else {
				email.LaterStatus = "250 2.0.0 Ok"
			}
			model.SetStatus(&email)
			wg.Done()
		}(&wg, emails[i])
	}
	wg.Wait()
}
