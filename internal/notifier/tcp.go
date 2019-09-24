package notifier

import (
	"bytes"
	"context"
	"encoding/gob"
	"log"
	"net"
)

func Connect(addr string, client Client) {

}

func pingTCP(ctx context.Context, msg Client) {
	conn, err := net.Dial("tcp4", msg.Addr)
	if err != nil {
		log.Printf("unable to dial: %v", err)
		return
	}

	h := handlers{
		MsgTypeApprove: func(msg Message) error {
			log.Print("approved")
			return nil
		},
		MsgTypePong: func(msg Message) error {
			log.Print("pong")
			return nil
		},
		MsgTypePing: func(msg Message) error {
			log.Print("ping")
			return sendPong(conn)
		},
	}

	handleConnection(conn, h)
}

const maxDataSize = 4 << 10 // 4 KiB

func handleConnection(conn net.Conn, h handlers) {
	defer func() {
		_ = conn.Close()
		log.Print("connection has been closed")
	}()

	var err error
	var msg Message
	buf := make([]byte, maxDataSize)

	for n := 0; err == nil; n, err = conn.Read(buf) {
		data := bytes.NewBuffer(buf[:n])
		errgob := gob.NewDecoder(data).Decode(&msg)
		if errgob != nil {
			sendError(conn, err)
			log.Printf("unable to unmarshal: %v", errgob)
			return
		}

		handler, ok := h[msg.Type]
		if !ok {
			log.Print("unable to find handler, skiping")
			continue
		}

		if errh := handler(msg); errh != nil {
			sendError(conn, err)
			log.Printf("error during handling message: %v", errh)
			return
		}
	}
}

func sendHello(conn net.Conn, client Client) error {
	return gob.NewDecoder(conn).Decode(&client)
}

func sendPing(conn net.Conn) error {
	msg := Message{Type: MsgTypePing}
	return gob.NewDecoder(conn).Decode(&msg)
}

func sendPong(conn net.Conn) error {
	msg := Message{Type: MsgTypePong}
	return gob.NewDecoder(conn).Decode(&msg)
}

func sendError(conn net.Conn, err error) {
	msg := Message{
		Type: MsgTypeError,
		Data: []byte(err.Error()),
	}

	_ = gob.NewEncoder(conn).Encode(&msg)
}

type handlers map[MsgType]handlerFunc

type handlerFunc func(msg Message) error

type MsgType uint64

const (
	MsgTypeError   MsgType = iota + 1 // 1
	MsgTypeHello                      // 2
	MsgTypeApprove                    // 3
	MsgTypePing                       // 4
	MsgTypePong                       // 5
)

type Message struct {
	Type MsgType
	Data []byte
}
