package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

type MyClient struct {
	WAClient       *whatsmeow.Client
	eventHandlerID uint32
	msgChannel     chan *events.Message
}

type RecentMessage struct {
	Sender  string
	Name    string
	Message string
}

var recentMessages []RecentMessage

func (mycli *MyClient) processMessages() {
	var messages []*events.Message

	for {
		select {
		case msg := <-mycli.msgChannel:
			messages = append(messages, msg)
			// Wait for a 5-second delay before processing messages
			select {
			case msg := <-mycli.msgChannel:
				messages = append(messages, msg)
			case <-time.After(2 * time.Second):
				// No more messages within the 5-second delay, process all received messages
				for _, msg := range messages {
					mycli.handleMessage(msg)
				}
				messages = nil
			}
		}
	}
}

func (mycli *MyClient) handleMessage(msg *events.Message) {
	newMessage := msg.Message
	fmt.Println("Message from:", msg.Info.Sender.User, "->", newMessage.GetConversation())

	senderName := msg.Info.PushName
	if senderName == "" {
		senderName = msg.Info.Sender.User
	}

	addRecentMessage(&msg.Info.Sender, senderName, msg.Message.GetConversation())

	data := map[string]interface{}{
		"message":         msg.Message.GetConversation(),
		"message_history": getRecentMessages(25),
		"person":          senderName,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return
	}

	// Send a POST request with JSON data
	req, err := http.NewRequest("POST", "http://localhost:5001/chat", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return
	}

	// Read the response
	var jsonResponse map[string]string
	err = json.NewDecoder(resp.Body).Decode(&jsonResponse)
	if err != nil {
		fmt.Println("Error decoding JSON response:", err)
		return
	}
	newMsg := jsonResponse["response"]
	fmt.Println("Response:", newMsg)

	// encode out as a string
	response := &waProto.Message{Conversation: proto.String(string(newMsg))}

	userJid := types.NewJID(msg.Info.Sender.User, types.DefaultUserServer)
	mycli.WAClient.SendMessage(context.Background(), userJid, "", response)
}

func (mycli *MyClient) register() {
	mycli.eventHandlerID = mycli.WAClient.AddEventHandler(mycli.eventHandler)
	mycli.msgChannel = make(chan *events.Message, 100) // Create a buffered channel for messages
	go mycli.processMessages()                         // Start a goroutine to process messages
}

func addRecentMessage(sender *types.JID, senderName string, message string) {
	recentMessages = append(recentMessages, RecentMessage{Sender: sender.User, Name: senderName, Message: message})

	// Keep only the last 50 messages to prevent the slice from growing indefinitely
	if len(recentMessages) > 50 {
		recentMessages = recentMessages[1:]
	}
}

func getRecentMessages(n int) []RecentMessage {
	start := 0
	if len(recentMessages) > n {
		start = len(recentMessages) - n
	}
	return recentMessages[start:]
}

func (mycli *MyClient) eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		mycli.msgChannel <- v // Add the message to the channel for processing
	}
}

func main() {
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	// Make sure you add appropriate DB connector imports, e.g. github.com/mattn/go-sqlite3 for SQLite
	container, err := sqlstore.New("sqlite3", "file:examplestore.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}
	// If you want multiple sessions, remember their JIDs and use .GetDevice(jid) or .GetAllDevices() instead.
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		panic(err)
	}
	clientLog := waLog.Stdout("Client", "DEBUG", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)
	// add the eventHandler
	mycli := &MyClient{WAClient: client}
	mycli.register()

	if client.Store.ID == nil {
		// No ID stored, new login
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			panic(err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				// Render the QR code here
				// e.g. qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				// or just manually `echo 2@... | qrencode -t ansiutf8` in a terminal
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				//				fmt.Println("QR code:", evt.Code)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		// Already logged in, just connect
		err = client.Connect()
		if err != nil {
			panic(err)
		}
	}

	// Listen to Ctrl+C (you can also do something else that prevents the program from exiting)
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	client.Disconnect()
}
