package chat

import (
	"fmt"
	"github.com/lukaspj/go-chord/chord"
	"net/rpc"
	"net"
	"net/http"
	"net/url"
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
	client.info = chord.ContactInfo{
		Address: fmt.Sprintf("127.0.0.1:%d", port1),
		Id:      chord.NewNodeIDFromHash(fmt.Sprintf("127.0.0.1:%d", port2)),
		Payload: []byte(fmt.Sprintf("127.0.0.1:%d", port2)),
	}
	client.peer = chord.NewPeer(client.info, port1)
	client.msgPort = port2

	client.inbox = make(chan string, 15)
	client.outbox = make(chan string, 15)
	client.stop = make(chan bool)
	client.done = make(chan bool)

	return
}

func (client *Client) Call(contact chord.ContactInfo, method string, args, reply interface{}) (err error) {
	href := url.URL{}
	href.UnmarshalBinary(contact.Payload) // TODO handle err
	logger.Info("%s -> %s", method, href.String())
	var rpcClient *rpc.Client
	if rpcClient, err = rpc.DialHTTP("tcp", href.Host); err == nil {
		err = rpcClient.Call(method, args, reply)
		rpcClient.Close()
	}
	return
}

func (client *Client) HandleRPC(request *RPCHeader, response *RPCHeader) error {
	// TODO handle standard headers
	/*if !request.ReceiverId.IsZero() && !request.ReceiverId.Equals(peer.Info.Id) {
		return errors.New(fmt.Sprintf("Expected network ID %s, got %s",
			peer.Info.Id, request.ReceiverId))
	}

	response.Sender = peer.Info
	response.ReceiverId = request.Sender.Id*/
	return nil
}

func (client *Client) StartChatting() {
	rpc.Register(&ClientApi{client})

	oldMux := http.DefaultServeMux
	mux := http.NewServeMux()
	http.DefaultServeMux = mux

	rpc.HandleHTTP()

	http.DefaultServeMux = oldMux

	if l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", client.msgPort)); err == nil {
		go http.Serve(l, nil)
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
				succ := client.peer.GetSuccessor()

				err := client.TransferMessage(succ, s)
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

func (client *Client) SendMessage(s string) {
	client.outbox <- s
}

func (client *Client) TransferMessage(info chord.ContactInfo, msg string) (err error) {
	req := SendMessageRequest{
		RPCHeader: RPCHeader{},
		Owner:     client.info,
		Message:   msg,
	}

	resp := SendMessageResponse{}
	err = client.Call(info, "ClientApi.SendMessage", req, &resp)
	return
}

func (client *Client) Close() {
	client.stop <- true
	<-client.done
}
