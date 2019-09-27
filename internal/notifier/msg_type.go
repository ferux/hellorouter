package notifier

type MsgType uint64

const (
	MsgTypeError     MsgType = iota + 1 // 1
	MsgHelloResponse                    // 2
	MsgHelloRequest                     // 3
	MsgTypeApprove                      // 4
	MsgTypePing                         // 5
	MsgTypePong                         // 6
)

// nolint:checknoglobals
var msgToString = map[MsgType]string{
	MsgTypeError:     "TYPE_ERROR",
	MsgHelloResponse: "HELLO_RESPONSE",
	MsgHelloRequest:  "HELLO_REQUEST",
	MsgTypeApprove:   "TYPE_APPROVE",
	MsgTypePing:      "TYPE_PING",
	MsgTypePong:      "TYPE_PONG",
}

func (m MsgType) String() string {
	return msgToString[m]
}
