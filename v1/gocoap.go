package gocoap

import (
	"fmt"
	"log"

	"time"

	coap "github.com/dustin/go-coap"
	// "github.com/eriklupander/dtls"
	"github.com/moroen/dtls"
)

type RequestParams struct {
	Host    string
	Port    int
	Uri     string
	Id      string
	Key     string
	Req     coap.Message
	Payload string
}

var _listener *dtls.Listener
var _peer *dtls.Peer

var retryLimit = 3

func _processMessage(msg coap.Message) error {
	switch msg.Code {
	case coap.MethodNotAllowed:
		return MethodNotAllowed
	case coap.NotFound:
		return UriNotFound
	case coap.Content:
		return nil
	case coap.Changed:
		return nil
	case coap.Created:
		return nil
	case coap.BadRequest:
		return BadRequest
	case coap.Unauthorized:
		return Unauthorized
	}

	return ErrorUnknownError
}

func _request(params RequestParams) (retmsg coap.Message, err error) {
	conn, err := coap.Dial("udp", fmt.Sprintf("%s:%d", params.Host, params.Port))
	if err != nil {
		return retmsg, err
	}

	resp, err := conn.Send(params.Req)
	if err != nil {
		return retmsg, err
	}

	err = _processMessage(*resp)

	return *resp, err
}

func getDTLSConnection(params RequestParams) (*dtls.Listener, *dtls.Peer, error) {
	if _listener == nil {
		mks := dtls.NewKeystoreInMemory()
		dtls.SetKeyStores([]dtls.Keystore{mks})
		mks.AddKey(params.Id, []byte(params.Key))

		newListner, err := dtls.NewUdpListener(":0", time.Second*900)
		if err != nil {
			return nil, nil, ErrorHandshake
		}
		_listener = newListner
	}

	if _peer == nil {
		peerParams := &dtls.PeerParams{
			Addr:             fmt.Sprintf("%s:%d", params.Host, params.Port),
			Identity:         params.Id,
			HandshakeTimeout: time.Second * 3}

		newPeer, err := _listener.AddPeerWithParams(peerParams)
		if err != nil {
			return nil, nil, ErrorHandshake
		}

		newPeer.UseQueue(true)
		_peer = newPeer
	}

	return _listener, _peer, nil
}

func _requestDTLS(params RequestParams, retry int) (retmsg coap.Message, err error) {

	listner, peer, err := getDTLSConnection(params)
	if err != nil {
		return coap.Message{}, err
	}

	data, err := params.Req.MarshalBinary()
	if err != nil {
		return coap.Message{}, ErrorUnknownError
	}

	err = peer.Write(data)
	if err != nil {
		log.Println("Read Timeout")

		listner.Shutdown()
		_peer = nil
		_listener = nil

		if retry < retryLimit {
			log.Println("Retrying Write request")
			return _requestDTLS(params, retry+1)
		} else {
			return coap.Message{}, err
		}
	}

	respData, err := peer.Read(time.Second)
	if err != nil {
		log.Println("Read Timeout")

		listner.Shutdown()
		_peer = nil
		_listener = nil

		if retry < retryLimit {
			log.Println("Retrying Read request")
			return _requestDTLS(params, retry+1)
		} else {
			return coap.Message{}, err
		}
	}

	msg, err := coap.ParseMessage(respData)
	if err != nil {
		return coap.Message{}, ErrorBadData
	}

	err = _processMessage(msg)
	return msg, err
}

// Observe a uri

// GetRequest sends a default get
func GetRequest(params RequestParams) (response []byte, err error) {
	params.Req = coap.Message{
		Type:      coap.Confirmable,
		Code:      coap.GET,
		MessageID: 1,
	}

	params.Req.SetPathString(params.Uri)

	var msg coap.Message

	if params.Id != "" {
		msg, err = _requestDTLS(params, 0)
	} else {
		msg, err = _request(params)
	}
	return msg.Payload, err
}

// PutRequest sends a default Put-request
func PutRequest(params RequestParams) (response []byte, err error) {

	params.Req = coap.Message{
		Type:      coap.Confirmable,
		Code:      coap.PUT,
		MessageID: 1,
		Payload:   []byte(params.Payload),
	}

	params.Req.SetPathString(params.Uri)

	var msg coap.Message

	if params.Id != "" {
		msg, err = _requestDTLS(params, 0)
	} else {
		msg, err = _request(params)
	}

	return msg.Payload, err
}

// PostRequest sends a default Post-request
func PostRequest(params RequestParams) (response []byte, err error) {
	params.Req = coap.Message{
		Type:      coap.Confirmable,
		Code:      coap.POST,
		MessageID: 1,
		Payload:   []byte(params.Payload),
	}

	params.Req.SetPathString(params.Uri)

	var msg coap.Message

	if params.Id != "" {
		msg, err = _requestDTLS(params, 0)
	} else {
		msg, err = _request(params)
	}

	return msg.Payload, err
}
