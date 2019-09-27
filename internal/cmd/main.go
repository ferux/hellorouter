package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/ferux/hellorouter/internal/notifier"
	"github.com/ferux/hellorouter/internal/server"
)

var (
	revision  string // nolint:gochecknoglobals
	branch    string // nolint:gochecknoglobals
	buildTime string // nolint:gochecknoglobals
)

func main() {
	if len(os.Args) < 5 {
		println("expected args: address, delay, name, type")
		os.Exit(1)
	}

	addr := os.Args[1]
	delay, err := time.ParseDuration(os.Args[2])
	if err != nil {
		fatal(err)
	}

	addr = strings.TrimSuffix(addr, "/")
	addr += "/api/v1/ping"

	name := os.Args[3]
	if len(name) == 0 {
		println("name should not be empty")
		os.Exit(1)
	}

	kind := os.Args[4]
	if len(kind) == 0 {
		println("device type should not be empty")
		os.Exit(1)
	}

	bt, err := time.Parse(time.UnixDate, buildTime)
	if err != nil {
		fatal(err)
	}

	msg := notifier.Client{
		ID:        getID(),
		Name:      name,
		Type:      kind,
		Revision:  revision,
		Branch:    branch,
		BuildTime: bt,
		Addr:      "localhost:9999",
		// Addr:      addr,
		Delay: delay,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// go func() {
	// 	s := make(chan os.Signal, 1)
	// 	signal.Notify(s, os.Interrupt)

	// 	<-s
	// 	cancel()
	// }()

	go server.NewTCP(ctx, ":9999")

	// notifier.Start(ctx, msg)
	sd, err := notifier.NewTCPClient(msg)
	fatal(err)

	go func() {
		<-ctx.Done()
		_ = sd(nil)
	}()

	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt)

	<-s
	cancel()
}

func getID() string {
	f, err := os.OpenFile(".info", os.O_CREATE|os.O_RDWR, 0755)
	if err != nil {
		fatal(err)
	}

	defer func() { _ = f.Close() }()

	var data = make([]byte, 32)
	n, err := f.Read(data)
	if err != nil && err != io.EOF {
		fatal(err)
	}

	if n == 32 {
		return hex.EncodeToString(data)
	}

	_, err = rand.Read(data)
	if err != nil {
		fatal(err)
	}

	_, err = f.Write(data)
	if err != nil {
		fatal(err)
	}

	return hex.EncodeToString(data)
}

func fatal(err error) {
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}
}
