package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type Client struct {
	ID        string    `json:"id"`
	Name      string    `json:"name,omitempty"`
	Type      string    `json:"type,omitempty"`
	Revision  string    `json:"revision,omitempty"`
	Branch    string    `json:"branch,omitempty"`
	BuildTime time.Time `json:"build_time,omitempty"`

	Addr  string        `json:"-"`
	Delay time.Duration `json:"-"`
}

const defaultDelay = time.Second * 60

// Start registers this application to remote service and sends pings periodically.
func Start(ctx context.Context, msg Client) {
	if msg.Delay == 0 {
		msg.Delay = defaultDelay
	}

	ping(ctx, msg)
}

const contentType = "application/json"

func ping(ctx context.Context, msg Client) {
	var err error
	var resp *http.Response

	client := http.Client{
		Timeout: msg.Delay,
	}

	data, err := json.Marshal(&msg)
	if err != nil {
		log.Printf("unable to endode data: %v", err)
	}

	t := time.NewTimer(0)
	for {
		select {
		case <-ctx.Done():
			if !t.Stop() {
				<-t.C
			}
			return
		case <-t.C:
			t.Reset(msg.Delay)

			resp, err = client.Post(msg.Addr, contentType, bytes.NewReader(data))
			if err != nil {
				log.Printf("unable to post data: %v", err)
				continue
			}

			if resp.StatusCode != http.StatusOK {
				log.Printf("expected 200, got %d", resp.StatusCode)
				continue
			}

			_, _ = io.Copy(ioutil.Discard, resp.Body)

			_ = resp.Body.Close()
		}
	}
}
