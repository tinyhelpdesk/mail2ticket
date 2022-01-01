package util

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Ticket struct {
	ID          primitive.ObjectID `bson:"_id"`
	CreatedAt   time.Time          `bson:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at"`
	Sender      string             `bson:"sender"`
	Recipient   string             `bson:"recipient"`
	Subject     string             `bson:"subject"`
	MessageText string             `bson:"message_text"`
	MessageID   string             `bson:"message_id"`
	Completed   bool               `bson:"completed"`
}

type TicketDatabase struct {
	url        string
	port       int
	username   string
	password   string
	database   string
	collection string
	mclt       *mongo.Client
	mcol       *mongo.Collection
	mctx       context.Context
}

func NewDatabase() (*TicketDatabase, error) {
	log.Println("NewDatabase called.")

	err := godotenv.Load("../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	port, _ := strconv.Atoi(os.Getenv("MONGO_PORT"))
	td := TicketDatabase{
		url:        os.Getenv("MONGO_URL"),
		port:       port,
		username:   os.Getenv("MONGO_USERNAME"),
		password:   os.Getenv("MONGO_PASSWORD"),
		database:   os.Getenv("MONGO_DATABASE"),
		collection: os.Getenv("MONGO_COLLECTION"),
	}

	clientOptions := options.Client().ApplyURI(fmt.Sprintf("mongodb://%s:%s@%s:%d", td.username, td.password, td.url, td.port))
	client, err := mongo.NewClient(clientOptions)
	if err != nil {
		log.Printf("Cannot create MongoDB client. %s", err)
	}

	// log.Printf("clientOptions type: %v\n", reflect.TypeOf(clientOptions))
	// log.Printf("client type: %v\n", reflect.TypeOf(client))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("MongoDB connection established.")

	td.mclt = client
	td.mcol = client.Database(td.database).Collection(td.collection)
	td.mctx = ctx

	defer cancel()
	defer client.Disconnect(ctx)

	// dbs, err := td.mclt.ListDatabaseNames(ctx, bson.M{})
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// log.Printf("MongoDB Databases: %v", dbs)

	td.manageTickets()

	return &td, nil
}

func (tdb *TicketDatabase) manageTickets() {
	log.Println("ManageTickets called.")

	sub := createRandomStrings(15)
	msg := createRandomStrings(125)
	mid := createRandomStrings(1)

	t1 := &Ticket{
		ID:          primitive.NewObjectID(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Sender:      "Björn Hahn <info@bjoerntorben.de>",
		Recipient:   "BTH Support <support@bjoerntorben.de>",
		MessageText: msg,
		MessageID:   mid,
		Subject:     sub,
		Completed:   false,
	}

	if err := tdb.createTicket(t1); err != nil {
		log.Printf("createTicket Error: %s", err)
	}

	tickets, err := tdb.getAllTickets()
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Println("No tickets found")
		}
	}
	tdb.printTickets(tickets)

	if _, err := tdb.getPendingTickets(); err != nil {
		log.Printf("getPendingTickets Error: %s", err)
	}

	if _, err := tdb.getCompletedTickets(); err != nil {
		log.Printf("getCompletedTickets Error: %s", err)
	}
}

func createRandomStrings(lt int) string {
	rand.Seed(time.Now().Unix())
	var output strings.Builder
	charSet := "abcdedfghijklmnopqrstABCDEFGHIJKLMNOP"
	length := lt
	for i := 0; i < length; i++ {
		random := rand.Intn(len(charSet))
		randomChar := charSet[random]
		output.WriteString(string(randomChar))
	}

	return output.String()
}

func (tdb *TicketDatabase) createTicket(tt *Ticket) error {
	log.Println("createTicket called.")
	_, err := tdb.mcol.InsertOne(tdb.mctx, tt)
	return err
}

func (tdb *TicketDatabase) filterTickets(filter interface{}) ([]*Ticket, error) {
	log.Println("filterTickets called.")
	var tickets []*Ticket

	cur, err := tdb.mcol.Find(tdb.mctx, filter)
	if err != nil {
		return tickets, err
	}
	// Iterate through the cursor and decode each document one at a time
	for cur.Next(tdb.mctx) {
		var t Ticket
		err := cur.Decode(&t)
		if err != nil {
			return tickets, err
		}

		tickets = append(tickets, &t)
	}

	if err := cur.Err(); err != nil {
		return tickets, err
	}

	// once exhausted, close the cursor
	cur.Close(tdb.mctx)

	if len(tickets) == 0 {
		return tickets, mongo.ErrNoDocuments
	}

	return tickets, nil
}

func (tdb *TicketDatabase) getAllTickets() ([]*Ticket, error) {
	log.Println("getAllTickets called.")
	filter := bson.D{{}}
	return tdb.filterTickets(filter)
}

func (tdb *TicketDatabase) completeTicket(mt string) error {
	log.Println("completeTicket called.")
	filter := bson.D{primitive.E{Key: "MessageText", Value: mt}}

	update := bson.D{primitive.E{Key: "$set", Value: bson.D{
		primitive.E{Key: "completed", Value: true},
	}}}
	t := &Ticket{}
	return tdb.mcol.FindOneAndUpdate(tdb.mctx, filter, update).Decode(t)
}

func (tdb *TicketDatabase) getPendingTickets() ([]*Ticket, error) {
	log.Println("getPendingTickets called.")

	filter := bson.D{
		primitive.E{Key: "completed", Value: false},
	}

	return tdb.filterTickets(filter)
}

func (tdb *TicketDatabase) getCompletedTickets() ([]*Ticket, error) {
	log.Println("addTicket called.")
	filter := bson.D{
		primitive.E{Key: "completed", Value: true},
	}

	return tdb.filterTickets(filter)
}

func (tdb *TicketDatabase) deleteTicket(mt string) error {
	log.Println("deleteTicket called.")
	filter := bson.D{primitive.E{Key: "MessageText", Value: mt}}

	res, err := tdb.mcol.DeleteOne(tdb.mctx, filter)
	if err != nil {
		return err
	}

	if res.DeletedCount == 0 {
		return errors.New("no tickets were deleted")
	}

	return nil
}

func (tdb *TicketDatabase) printTickets(ticket []*Ticket) {
	for i, v := range ticket {
		if v.Completed {
			log.Printf("%d. Subject: %s\n", i+1, v.Subject)
			log.Printf("%d. MessageText: %s\n", i+1, v.MessageText)
		} else {
			log.Printf("%d. Subject: %s\n", i+1, v.Subject)
			log.Printf("%d. MessageText: %s\n", i+1, v.MessageText)
		}
	}
}
