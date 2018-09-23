package main

// #cgo pkg-config: python3
// #define Py_LIMITED_API
// #include <Python.h>
// int PyArg_ParseTuple_LL(PyObject *, long long *, long long *);
// int PyArg_ParseTuple_S(PyObject *, char *);
// char * ParseStringArgument(PyObject *);
import "C"

import (
	"errors"
	"time"

	"github.com/moroen/canopus"
)

var globalGatewayConfig GatewayConfig

type GatewayConfig struct {
	Gateway  string
	Identity string
	Passkey  string
}

type CoapResult struct {
	msg canopus.MessagePayload
	err error
}

var ErrorTimeout = errors.New("COAP Error: Connection timeout")
var ErrorBadIdent = errors.New("COAP DTLS Error: Wrong credentials?")
var ErrorNoConfig = errors.New("COAP Error: No config")

func GetConfig() (conf GatewayConfig, err error) {
	if globalGatewayConfig == (GatewayConfig{}) {
		err = ErrorNoConfig
	}
	return globalGatewayConfig, err
}

func _getRequest(URI string, c chan CoapResult) {

	var result CoapResult
	var conn canopus.Connection
	var err error

	conf, err := GetConfig()
	if err != nil {
		result.err = ErrorNoConfig
		c <- result
		return
	}

	if conf.Identity != "" {
		conn, err = canopus.DialDTLS(conf.Gateway, conf.Identity, conf.Passkey)

	} else {
		conn, err = canopus.Dial(conf.Gateway)

	}

	if err != nil {
		result.err = err
		c <- result
		return
	}

	req := canopus.NewRequest(canopus.MessageConfirmable, canopus.Get)
	req.SetStringPayload("Hello, canopus")
	req.SetRequestURI(URI)

	resp, err := conn.Send(req)
	if err != nil {
		result.err = ErrorBadIdent
		c <- result
		return
	}

	// response := resp.GetMessage().GetPayload()
	result.err = nil
	result.msg = resp.GetMessage().GetPayload()
	c <- result
}

func _putRequest(URI, payload string, c chan CoapResult) {
	var result CoapResult

	var conn canopus.Connection
	var err error

	conf, err := GetConfig()
	if err != nil {
		result.err = ErrorNoConfig
		c <- result
		return
	}

	if conf.Identity != "" {
		conn, err = canopus.DialDTLS(conf.Gateway, conf.Identity, conf.Passkey)

	} else {
		conn, err = canopus.Dial(conf.Gateway)

	}

	if err != nil {
		result.err = err
		c <- result
		return
	}

	req := canopus.NewRequest(canopus.MessageConfirmable, canopus.Put)
	req.SetRequestURI(URI)
	req.SetStringPayload(payload)

	resp, err := conn.Send(req)
	if err != nil {
		result.err = ErrorBadIdent
		c <- result
		return
	}

	result.msg = resp.GetMessage().GetPayload()
	result.err = nil
	c <- result
}

func GetRequest(URI string) (msg canopus.MessagePayload, err error) {
	c := make(chan CoapResult)

	go _getRequest(URI, c)

	select {
	case res := <-c:
		return res.msg, res.err
	case <-time.After(time.Second * 5):
		return nil, ErrorTimeout
	}
}

func PutRequest(URI, payload string) (msg canopus.MessagePayload, err error) {
	c := make(chan CoapResult)

	go _putRequest(URI, payload, c)

	select {
	case _ = <-c:
		return GetRequest(URI)
	case <-time.After(time.Second * 5):
		return nil, ErrorTimeout
	}
}

/* func Observe(URI string) {

	conf, err := GetConfig()
	if err != nil {
		result.err = ErrorNoConfig
		c <- result
		return
	}

	conn, err := canopus.DialDTLS(conf.Gateway, conf.Identity, conf.Passkey)

	tok, err := conn.ObserveResource("/15001/65540")
	if err != nil {
		panic(err.Error())
	}

	obsChannel := make(chan canopus.ObserveMessage)
	done := make(chan bool)
	go conn.Observe(obsChannel)

	notifyCount := 0
	go func() {
		for {
			select {
			case obsMsg, open := <-obsChannel:
				if open {
					if notifyCount == 5 {
						fmt.Println("[CLIENT >> ] Canceling observe after 5 notifications..")
						go conn.CancelObserveResource("watch/this", tok)
						go conn.StopObserve(obsChannel)
						done <- true
						return
					} else {
						notifyCount++
						// msg := obsMsg.Msg\
						resource := obsMsg.GetResource()
						val := obsMsg.GetValue()

						fmt.Println("[CLIENT >> ] Got Change Notification for resource and value: ", notifyCount, resource, val)
					}
				} else {
					done <- true
					return
				}
			}
		}
	}()
	<-done
	fmt.Println("Done")
}
*/
func main() {}
