package chat

import (
	"github.com/lukaspj/go-chord/chord"
	"net/url"
)

type ClientApi struct {
	client *Client
}

type RPCHeader struct {

}

type SendMessageRequest struct {
	RPCHeader
	Owner   chord.ContactInfo
	Message string
}

type SendMessageResponse struct {
	RPCHeader
}

func (pw *ClientApi) SendMessage(args *SendMessageRequest, response *SendMessageResponse) (err error) {
	if err = pw.client.HandleRPC(&args.RPCHeader, &response.RPCHeader); err == nil {
		href := url.URL{}
		err = href.UnmarshalBinary(args.Owner.Payload) // TODO handle err
		logger.Info("At %s: Message %s from %s\n", string(pw.client.info.Payload), args.Message, href.String())
		if !args.Owner.Id.Equals(pw.client.info.Id) {
			pw.client.inbox <- args.Message
			succ := pw.client.peer.GetSuccessor()
			pw.client.Call(succ, "ClientApi.SendMessage", args, response)
		}
	} else {
		logger.Error("error happened: %v\n", err)
	}
	return
}