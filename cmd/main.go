package main

import (
	"fmt"
	"log"
	mail "tinyhelpdesk/pkg"
)

func main() {
	ms, err := mail.New()
	if err != nil {
		log.Fatal(err)
	}
	defer ms.Close()

	nm := make(chan *mail.Mail)
	ms.Subscribe(nm)
	ms.Run()

	for m := range nm {
		log.Printf("Receive new mail From: %s\n", m.From)
		log.Printf("Receive new mail Date: %s\n", m.Date)
		log.Printf("Receive new mail MessageID: %s\n", m.MessageID)
		log.Printf("Receive new mail Subject: %s\n", m.Subject)
		log.Printf("Receive new mail To: %s\n", m.To)
		//log.Printf("Receive new mail Attachment: %s\n", m.AttachmentFilename)
		log.Printf("Receive new mail Body: %s\n", m.Body)
		fmt.Println()
	}
}
