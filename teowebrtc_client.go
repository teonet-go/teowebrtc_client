// Copyright 2021 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Webrts client package
package teowebrtc_client

import (
	"encoding/json"
	"log"

	"github.com/teonet-go/teowebrtc_signal_client"
	"github.com/pion/webrtc/v3"
)

func Connect(scheme, signalServerAddr, login, server string, connected func(peer string, dc *DataChannel)) (err error) {

	var wait = make(chan interface{})
	defer close(wait)

	// Create signal server client
	signal := teowebrtc_signal_client.New()

	// Connect to signal server
	err = signal.Connect(scheme, signalServerAddr, login)
	if err != nil {
		log.Println("can't connect to signal server")
		return
	}
	log.Println()

	// Prepare the configuration
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	// Create a new RTCPeerConnection
	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return
	}
	defer pc.Close()

	// Create DataChannel
	dc, err := pc.CreateDataChannel("teo", nil)
	if err != nil {
		return
	}

	pc.OnSignalingStateChange(func(state webrtc.SignalingState) {
		log.Println("Signal changed:", state)
	})

	// Add handlers for setting up the connection.
	pc.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		log.Printf("ICE Connection State has changed: %s\n", connectionState.String())
		switch connectionState.String() {
		case "connected":
			connected(server, &DataChannel{dc})
		case "disconnected":
			dc.Close()
			wait <- struct{}{}
		}
	})

	// Initiates the offer
	offer, _ := pc.CreateOffer(nil)

	// Send offer and get answer
	offerData, err := json.Marshal(offer)
	if err != nil {
		return
	}
	message, err := signal.WriteOffer(server, offerData)
	if err != nil {
		return
	}

	// Unmarshal answer
	var errMsg = "can't unmarshal answer, error:"
	var sig teowebrtc_signal_client.Signal
	err = json.Unmarshal(message, &sig)
	if err != nil {
		log.Println(errMsg, err, "message: '"+string(message)+"'")
		return
	}
	peer := sig.Peer
	var answer webrtc.SessionDescription
	err = json.Unmarshal(sig.Data, &answer)
	if err != nil {
		log.Println(errMsg, err, err, "message: '"+string(message)+"'",
			"sig.Data: '"+string(sig.Data)+"'")
		return
	}
	log.Printf("Got answer from %s", sig.Peer)

	// Send AddICECandidate to remote peer
	pc.OnICECandidate(func(i *webrtc.ICECandidate) {
		if i != nil {
			log.Println("ICECandidate:", i)
			candidateData, err := json.Marshal(i)
			if err != nil {
				log.Panicln("can't marshal ICECandidate, error:", err)
				return
			}
			signal.WriteCandidate(peer, candidateData)
		} else {
			log.Println("Collection of candidates is finished ")
			signal.WriteCandidate(peer, nil)
		}
	})

	// Set local SessionDescription
	err = pc.SetLocalDescription(offer)
	if err != nil {
		log.Println("SetLocalDescription error, err:", err)
		return
	}

	// Set remote SessionDescription
	err = pc.SetRemoteDescription(answer)
	if err != nil {
		log.Println("SetRemoteDescription error, err:", err)
		return
	}

	// Get servers ICECandidate
	for {
		sig, err := signal.WaitSignal()
		if err != nil {
			break
		}

		// Unmarshal ICECandidate signal
		var i webrtc.ICECandidate
		if len(sig.Data) == 0 {
			log.Println("All ICECandidate processed")
			break
		}
		err = json.Unmarshal(sig.Data, &i)
		if err != nil {
			log.Println("can't unmarshal candidate, error:", err)
			continue
		}
		log.Printf("Got ICECandidatecandidate from %s", sig.Peer)

		// Add servers ICECandidate
		err = pc.AddICECandidate(i.ToJSON())
		if err != nil {
			log.Println("can't add ICECandidate, error:", err)
		}
	}

	// Close signal server connection
	signal.Close()
	<-wait

	return
}

func NewDataChannel(dc *webrtc.DataChannel) *DataChannel {
	return &DataChannel{dc}
}

type DataChannel struct {
	dc *webrtc.DataChannel
}

func (d *DataChannel) OnOpen(f func()) {
	d.dc.OnOpen(f)
}

func (d *DataChannel) OnClose(f func()) {
	d.dc.OnClose(f)
}

func (d *DataChannel) OnMessage(f func(data []byte)) {
	d.dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		f(msg.Data)
	})
}

func (d *DataChannel) Send(data []byte) error {
	return d.dc.Send(data)
}
