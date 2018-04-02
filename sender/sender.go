package sender

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/supme/smtpRelay/model"
	"github.com/supme/smtpSender"
	"io"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

// Run start sending queue emails
func Run() {
	if model.Config.Hostname == "" {
		h, err := os.Hostname()
		if err != nil {
			log.Print(err)
		}
		model.Config.Hostname = h
	}

	pipeQueue := smtpSender.NewPipe(
		smtpSender.Config{
			Hostname: model.Config.Hostname,
			Stream:   int(model.Config.SendStream),
		},
	)
	pipeQueue.Start()
	go sendQueue(pipeQueue)

	time.Sleep(1 * time.Second)
	pipeResend := smtpSender.NewPipe(
		smtpSender.Config{
			Hostname: model.Config.Hostname,
			Stream:   int(model.Config.ResendStream),
		},
	)
	pipeResend.Start()
	go resendQueue(pipeResend)

	breakSigs := make(chan os.Signal, 1)
	signal.Notify(breakSigs, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGKILL)

	go func() {
		for {
			select {
			case <-breakSigs:
				pipeQueue.Stop()
				pipeResend.Stop()
				goto End
			}
		}
	End:
		log.Println("Stoped all sender for exit")
		os.Exit(0)
	}()

}

func sendQueue(pipe smtpSender.Pipe) {
	for {
		emails := model.GetNewQueue(model.Config.SendStream)
		if len(emails) == 0 {
			time.Sleep(10 * time.Second)
		} else {
			for i := range emails {
				send(pipe, emails[i])
			}
		}
	}
}

func resendQueue(pipe smtpSender.Pipe) {
	for {
		emails := model.GetRepeatQueue(model.Config.ResendStream)
		if len(emails) == 0 {
			time.Sleep(10 * time.Second)
		} else {
			for i := range emails {
				send(pipe, emails[i])
			}
		}
	}
}

func send(pipe smtpSender.Pipe, email model.Queue) {
	e := smtpSender.Email{
		ID:   strconv.FormatUint(email.ID, 10),
		From: email.From,
		To:   email.Rcpt,
		WriteCloser: func(w io.WriteCloser) error {
			dataByte, err := base64.StdEncoding.DecodeString(email.Data)
			if err != nil {
				log.Printf("Send queue base64 decode error: %s", err)
				return fmt.Errorf("Send queue base64 decode error: %s", err)
			}
			bodyBuf := new(bytes.Buffer)
			_, err = bodyBuf.Write(dataByte)
			if err != nil {
				log.Printf("Send queue buffer write error: %s", err)
				return fmt.Errorf("Send queue buffer write error: %s", err)
			}
			_, err = io.Copy(w, bodyBuf)
			if err != nil {
				log.Printf("Send queue copy buffer to writer error: %s", err)
				return fmt.Errorf("Send queue copy buffer to writer error: %s", err)
			}
			return nil
		},
		ResultFunc: func(result smtpSender.Result) {
			if result.Err != nil {
				email.LaterStatus = result.Err.Error()
			} else {
				email.LaterStatus = "250 2.0.0 Ok"
			}
			model.SetStatus(&email)
		},
	}

	err := pipe.Send(e)
	if err != nil {
		log.Printf("Send email id '%d' error %+v\n", email.ID, err)
	}
}
