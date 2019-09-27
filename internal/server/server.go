package server

import (
	"context"
	"encoding/gob"
	"errors"
	"net"
	"time"

	"github.com/ferux/hellorouter/internal/notifier"
)

type TCP struct {
	l net.Listener
	h handlers
}

func NewTCP(ctx context.Context, listen string) {
	l, err := net.Listen("tcp4", listen)
	if err != nil {
		println("unable to start listen: ", err.Error())
		return
	}

	go func() {
		<-ctx.Done()
		_ = l.Close()
	}()

	for {
		println("ready to accept connection")
		conn, err := l.Accept()
		if err != nil {
			println("unable to accept: ", err.Error())
			return
		}

		println("accepted ", conn.RemoteAddr().String())

		device, err := registerDevice(conn)
		if err != nil {
			println("unable to register device: ", err.Error())
		}

		println(device.info.ID)
	}
}

func registerDevice(conn net.Conn) (d tcpDevice, err error) {
	enc := gob.NewEncoder(conn)
	dec := gob.NewDecoder(conn)

	msg := notifier.Message{Type: notifier.MsgHelloRequest}
	if err = enc.Encode(&msg); err != nil {
		return d, err
	}

	if err = dec.Decode(&msg); err != nil {
		return d, err
	}

	if msg.Type != notifier.MsgHelloResponse {
		return d, errors.New("expected helloresponse, got " + msg.Type.String())
	}

	info, ok := msg.Data.(notifier.Client)
	if !ok {
		return d, errors.New("unable to cast data to client info")
	}

	msg = notifier.Message{Type: notifier.MsgTypeApprove}
	if err = enc.Encode(&msg); err != nil {
		return d, err
	}

	d = tcpDevice{
		conn: conn,
		h: handlers{
			notifier.MsgTypeError: func(msg notifier.Message) error {
				errMsg := msg.Data.(string)
				println(info.ID, ": ", errMsg)
				return nil
			},
			notifier.MsgTypePong: func(_ notifier.Message) error {
				println("extending time for ", info.ID)
				return conn.SetDeadline(time.Now().Add(time.Second * 15))
			},
		},
		enc:  enc,
		dec:  dec,
		info: info,
	}

	go func() {
		for {
			time.Sleep(time.Second * 5)
			println("pinging ", d.info.ID)
			err = d.enc.Encode(&notifier.Message{
				Type: notifier.MsgTypePing,
			})
		}
	}()

	go func() {
		var msg notifier.Message
		var err error
		for {
			err = d.dec.Decode(&msg)
			if err != nil {
				println("unable to decode: ", err)
				return
			}

			h, ok := d.h[msg.Type]
			if !ok {
				println("unknown handler for ", msg.Type.String())
				continue
			}

			err = h(msg)
			if err != nil {
				println("error handling message: ", err.Error())
			}
		}
	}()

	return d, nil
}

type tcpDevice struct {
	conn net.Conn
	h    handlers
	enc  *gob.Encoder
	dec  *gob.Decoder

	info notifier.Client
}

// Shutdown closes connection.
func (d *tcpDevice) Shutdown(_ context.Context) error {
	_ = d.conn.Close()
	return nil
}

type handlers map[notifier.MsgType]handler

type handler func(msg notifier.Message) error
