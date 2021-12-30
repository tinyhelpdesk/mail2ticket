package tinyhelpdesk

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"
	"tinyhelpdesk/internal/util"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"
	"github.com/joho/godotenv"
)

type MailServer struct {
	username      string
	password      string
	url           string
	port          int
	cl            *client.Client
	subscriptions []chan *Mail
}

type Mail struct {
	AttachmentFilename string
	Body               []byte
	Date               time.Time
	From               string
	MessageID          string
	Subject            string
	To                 []*mail.Address
}

func New() (*MailServer, error) {
	// Read .env file
	err := godotenv.Load("../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	log.Println("Connecting to your mail server.")

	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	ms := MailServer{
		username: os.Getenv("SMTP_USERNAME"),
		password: os.Getenv("SMTP_PASSWORD"),
		url:      os.Getenv("SMTP_URL"),
		port:     port,
	}

	// Connect to server
	c, err := client.DialTLS(fmt.Sprintf("%s:%d", ms.url, ms.port), nil)
	if err != nil {
		log.Fatal(err)
	}

	// Login
	if err := c.Login(ms.username, ms.password); err != nil {
		log.Fatal(err)
	}
	log.Println("Authentication was successfull.")
	ms.cl = c

	return &ms, nil
}

func (ms *MailServer) FetchUnreadMessages() {
	log.Println("FetchUnreadMessages called.")

	ms.cl.Select("INBOX", false)
	imap.CharsetReader = charset.Reader

	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{"\\Seen"}

	uids, err := ms.cl.UidSearch(criteria)
	if err != nil {
		log.Println(err)
	}
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uids...)

	section := &imap.BodySectionName{}
	items := []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags, imap.FetchInternalDate, section.FetchItem()}
	messages := make(chan *imap.Message)
	go func() {
		if err := ms.cl.UidFetch(seqSet, items, messages); err != nil {
			log.Printf("No unread mails available.")
			//log.Printf("More information below: %s\n", err)
		}
	}()

	for {
		message := <-messages
		if message == nil {
			return
		}

		r := message.GetBody(section)
		if r == nil {
			log.Printf("Server didn't returned message body.\n")
		}

		mr, err := mail.CreateReader(r)
		if err != nil {
			log.Fatal(err)
		}

		header := mr.Header
		m := Mail{}

		if date, err := header.Date(); err == nil {
			// log.Println("Date:", date)
			m.Date = date
		}
		if from, err := header.AddressList("From"); err == nil {
			// log.Println("From:", from)
			encFrom := util.ParseHeaderToUtf8(fmt.Sprintf("%s", from))
			m.From = encFrom
		}
		if to, err := header.AddressList("To"); err == nil {
			// log.Println("To:", to)
			m.To = to
		}
		if subject, err := header.Subject(); err == nil {
			// log.Println("Subject:", subject)
			m.Subject = subject
		}
		if messageid, err := header.MessageID(); err == nil {
			// log.Println("MessageID:", messageid)
			m.MessageID = messageid
		}

		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			} else if err != nil {
				panic(err) // TODO: dont use panic
			}

			switch h := p.Header.(type) {
			case *mail.InlineHeader:
				b, _ := ioutil.ReadAll(p.Body)
				// log.Printf("Got text: %v", string(b))
				m.Body = b
			case *mail.AttachmentHeader:
				filename, _ := h.Filename()
				// log.Printf("Got attachment: %v", filename)
				m.AttachmentFilename = filename
			}
		}
		ms.publish(&m)
	}
}

func (ms *MailServer) Subscribe(ch chan *Mail) {
	log.Println("Subscribe called.")
	ms.subscriptions = append(ms.subscriptions, ch)
}

func (ms *MailServer) publish(m *Mail) {
	log.Println("publish called.")
	for _, ch := range ms.subscriptions {
		ch <- m
	}
}

func (ms *MailServer) worker() {
	log.Println("worker called.")
	st := 900
	counter := 0
	for {
		if counter >= 20*1000 {
			log.Println("worker for called.")
			ms.FetchUnreadMessages()
			counter = 0
		}
		time.Sleep(time.Duration(st) * time.Millisecond)
		counter += st
	}
}

func (ms *MailServer) Run() error {
	log.Println("Run called.")

	go ms.worker()

	return nil
}

func (ms *MailServer) Close() error {
	log.Println("Close called.")

	for _, ch := range ms.subscriptions {
		close(ch)
	}

	return ms.cl.Close()
}
