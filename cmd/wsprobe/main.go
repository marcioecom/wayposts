package main

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func main() {
	base := flag.String("url", "ws://localhost:8080/v1/ws", "URL do WebSocket")
	user := flag.Int64("user", 11, "ID do usuário (token de desenvolvimento)")
	lastSeen := flag.Int64("last-seen", 0, "último post ID conhecido")
	timeout := flag.Duration("timeout", 30*time.Second, "tempo máximo de conexão")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	u, err := url.Parse(*base)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	q := u.Query()
	q.Set("token", strconv.FormatInt(*user, 10))
	q.Set("last_seen", strconv.FormatInt(*lastSeen, 10))
	u.RawQuery = q.Encode()

	conn, _, err := websocket.Dial(ctx, u.String(), nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "dial:", err)
		os.Exit(1)
	}
	defer conn.Close(websocket.StatusNormalClosure, "bye")

	fmt.Println("conectado; aguardando nudge")
	for {
		// Tipo concreto para preservar os IDs como inteiros.
		var msg struct {
			Type  string `json:"type"`
			Count int64  `json:"count"`
			Head  int64  `json:"head"`
		}
		if err := wsjson.Read(ctx, conn, &msg); err != nil {
			if ctx.Err() != nil {
				return
			}
			fmt.Fprintln(os.Stderr, "read:", err)
			os.Exit(1)
		}
		fmt.Printf("nudge: type=%s count=%d head=%d\n", msg.Type, msg.Count, msg.Head)
	}
}
