// Command vapid prints a new VAPID key pair for Web Push. Generate it once and
// keep it: changing the keys silently breaks every existing subscription.
//
//	go run ./cmd/vapid
package main

import (
	"fmt"
	"log"

	webpush "github.com/SherClockHolmes/webpush-go"
)

func main() {
	priv, pub, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("VAPID_PUBLIC_KEY=%s\nVAPID_PRIVATE_KEY=%s\n", pub, priv)
}
