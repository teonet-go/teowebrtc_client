// Copyright 2021 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Webrts client sample application
package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/kirill-scherba/teowebrtc/teowebrtc_client"
)

var addr = flag.String("addr", "localhost:8080", "http service address")
var name = flag.String("name", "client-1", "client name")
var server = flag.String("server", "server-1", "server name")

func main() {
	flag.Parse()
	log.SetFlags(0)

	err := teowebrtc_client.Connect(*addr, *name, *server, func(peer string, d *teowebrtc_client.DataChannel) {
		log.Println("Connected to", peer)
		// Send messages to created data channel
		var id = 0
		d.OnOpen(func() {
			for {
				id++
				d.Send([]byte(fmt.Sprintf("Hello from %s with id %d!", *name, id)))
				time.Sleep(5 * time.Second)
			}
		})
	})
	if err != nil {
		log.Fatalln("connect error:", err)
	}

	select {}
}