package tinyhelpdesk

import "time"

type Ticket struct {
	ID            uint64
	Subject       string
	Status        string
	CustomerName  string
	CustomerEMail string
	Category      string
	Description   string
	CreationDate  time.Time
	ModifiedDate  time.Time
	Attachment    string
	Assignee      string
}

type Tickets struct {
	Tickets []Ticket
}
