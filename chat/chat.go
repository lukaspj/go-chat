package chat

import (
	"fmt"
	"github.com/lukaspj/go-chord/chord"
	"net"
	"net/url"
	"context"
	"github.com/lukaspj/go-chat/api"
	"google.golang.org/grpc"
)

type Client struct {
	info    chord.ContactInfo
	msgPort int
	peer    chord.Peer
	inbox   chan string
	outbox  chan string
	stop    chan bool
	done    chan bool
}

func NewChatClient(port1, port2 int) (client *Client) {
	client = new(Client)
	chatUrlStr := fmt.Sprintf("http://127.0.0.1:%d", port2)
	chatUrl, err := url.Parse(chatUrlStr)
	if err != nil {
		panic(fmt.Sprintf("Unable to parse chatUrl: %s: %v", chatUrlStr, err))
	}

	payload, err := chatUrl.MarshalBinary()

	client.info = chord.ContactInfo{
		Address: fmt.Sprintf("127.0.0.1:%d", port1),
		Id:      chord.NewNodeIDFromHash(fmt.Sprintf("127.0.0.1:%d", port2)),
		Payload: payload,
	}
	client.peer = chord.NewPeer(&client.info, port1)
	client.msgPort = port2

	client.inbox = make(chan string, 15)
	client.outbox = make(chan string, 15)
	client.stop = make(chan bool)
	client.done = make(chan bool)

	return
}

func (client *Client) StartChatting() {
	grpcServer := grpc.NewServer()
	api.RegisterChatServer(grpcServer, client)

	if l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", client.msgPort)); err == nil {
		go grpcServer.Serve(l)
	}

	go func() {
		defer close(client.inbox)
		defer close(client.outbox)
		defer func() { client.done <- true }()
		for {
			select {
			case s := <-client.inbox:
				println(s)
			case s := <-client.outbox:
				err := client.TransferMessageToSuccessor(s)
				if err != nil {
					logger.Error("error when transferring message: %v", err)
				}
			}
		}
	}()
}

func (client *Client) Connect(address string) {
	client.peer.Listen()
	client.peer.Connect(address)
	client.StartChatting()
}

func (client *Client) StartRoom() {
	client.peer.Listen()
	client.StartChatting()
}

func (client *Client) QueueMessage(s string) {
	client.outbox <- s
}

func (client *Client) Call(contact *chord.ContactInfo, cb func(chatClient api.ChatClient) error) (err error) {
	href := url.URL{}
	err = href.UnmarshalBinary(contact.Payload) // TODO handle err

	conn, err := grpc.Dial(href.Host, grpc.WithInsecure())
	defer conn.Close()

	if err != nil {
		logger.Error("error communicating with grpc server [%s]: %v", href, err)
		return
	}
	chatClient := api.NewChatClient(conn)
	err = cb(chatClient)

	return
}

func (client *Client) SendMessage(ctx context.Context, msg *api.Message) (v *api.Void, err error) {
	v = &api.Void{}

	ownerId := chord.NodeID{Val: msg.Owner.Id.Val}
	if !ownerId.Equals(client.info.Id) {
		client.inbox <- msg.Message
		succ := client.peer.GetSuccessor()

		err = client.Call(succ, func(chatClient api.ChatClient) error {
			_, err = chatClient.SendMessage(context.Background(), msg)
			return err
		})
	}

	return
}

func (client *Client) TransferMessageToSuccessor(msg string) (err error) {
	info := client.peer.GetSuccessor()
	err = client.Call(info, func(chatClient api.ChatClient) error {
		_, err = chatClient.SendMessage(context.Background(), &api.Message{
			Owner:   client.info.ToAPI(),
			Message: msg,
		})
		return err
	})

	return
}

func (client *Client) Close() {
	client.stop <- true
	<-client.done
}
