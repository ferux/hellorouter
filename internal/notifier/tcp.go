package notifier

import (
	"context"
	"encoding/gob"
	"net"
	"time"
)

type NotifierError string

func (err NotifierError) Error() string {
	return string(err)
}

const (
	// ErrUnknownDialer is returned when NewTCPClient was unable to cast net.Conn to *net.TCPConn.
	ErrUnknownDialer NotifierError = "unknown tcp connection type"

	// ErrUnregistered is returned when NewTCPClient unable to provide registration at service.
	ErrUnregistered NotifierError = "unable to register"

	// ErrUnexpectedMessageType error
	ErrUnexpectedMessageType NotifierError = "unexpected message type"
)

func nilShutdown(context.Context) error { return nil }

type TCPClient struct {
	conn   *net.TCPConn
	info   Client
	writer *gob.Encoder
	reader *gob.Decoder
	h      handlers
}

// NewTCPClient inits connection to service at tries to register there.
// TODO: move to TLS
func NewTCPClient(info Client) (shutdown func(context.Context) error, err error) {
	time.Sleep(time.Second)
	conn, err := net.Dial("tcp4", info.Addr)
	if err != nil {
		println("unable to dial: ", err.Error())
		return nil, err
	}

	tcpconn, ok := conn.(*net.TCPConn)
	if !ok {
		return nilShutdown, ErrUnknownDialer
	}

	gob.Register(Client{})

	conn.SetDeadline(time.Now().Add(info.Delay))
	client := TCPClient{
		conn:   tcpconn,
		info:   info,
		writer: gob.NewEncoder(conn),
		reader: gob.NewDecoder(conn),
	}

	if err = client.register(); err != nil {
		_ = conn.Close()
		return nilShutdown, err
	}

	client.h = map[MsgType]handlerFunc{
		MsgTypePing: func(msg Message) error {
			println("ping message")
			out := Message{
				Type: MsgTypePong,
			}
			return client.writer.Encode(&out)
		},
	}

	go client.run()

	return client.Shutdown, nil
}

// Shutdown active connection.
func (c *TCPClient) Shutdown(_ context.Context) error {
	_ = c.conn.Close()
	return nil
}

func (c *TCPClient) register() error {
	println("[tcpclient] registering at server")

	var msg Message
	if err := c.reader.Decode(&msg); err != nil {
		return err
	}

	if msg.Type != MsgHelloRequest {
		return ErrUnexpectedMessageType
	}

	msg = Message{
		Type: MsgHelloResponse,
		Data: c.info,
	}

	// update delay becase we need to get approve from server
	c.conn.SetDeadline(time.Now().Add(c.info.Delay))
	if err := c.writer.Encode(&msg); err != nil {
		return err
	}

	if err := c.reader.Decode(&msg); err != nil {
		return err
	}

	if msg.Type != MsgTypeApprove {
		return ErrUnexpectedMessageType
	}

	// reset deadline because server answered successfuly.
	c.conn.SetDeadline(time.Time{})

	return nil
}

func (c *TCPClient) run() {
	var msg Message
	var err error
	for err = c.reader.Decode(&msg); err == nil; err = c.reader.Decode(&msg) {
		h, ok := c.h[msg.Type]
		if !ok {
			println("unexpected message type: ", msg.Type.String())
		}

		err = h(msg)
		if err != nil {
			sendError(c.writer, err)
		}
	}

	println("tcpclient.run(): ", err.Error())
}

const maxDataSize = 4 << 10 // 4 KiB

func sendError(enc *gob.Encoder, err error) {
	msg := Message{
		Type: MsgTypeError,
		Data: err.Error(),
	}

	_ = enc.Encode(&msg)
}

type handlers map[MsgType]handlerFunc

type handlerFunc func(msg Message) error

type Message struct {
	Type MsgType
	Data interface{}
}
