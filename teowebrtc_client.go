// Copyright 2021-2022 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Webrts client package
package teowebrtc_client

import (
	"encoding/json"

	"github.com/pion/webrtc/v3"
	"github.com/teonet-go/teowebrtc_log"
	"github.com/teonet-go/teowebrtc_signal_client"
)

var log = teowebrtc_log.GetLog(teowebrtc_log.Package_teowebrtc_client)

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
		log.Println("Signaling state change:", state)
	})

	// Add handlers for setting up the connection.
	pc.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		log.Printf("ICE connection state has change: %s\n", connectionState.String())
		switch connectionState.String() {
		case "connected":
			connected(server, NewDataChannel(dc))
		case "disconnected":
			dc.Close()
			wait <- struct{}{}
		}
	})

	// Initiates the offer
	offer, _ := pc.CreateOffer(nil)

	// Send offer and get answer
	message, err := signal.WriteOffer(server, offer)
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
	d, _ := json.Marshal(sig.Data)
	err = json.Unmarshal(d, &answer)
	if err != nil {
		log.Println(errMsg, err, err, "message: '"+string(message)+"'",
			"sig.Data: '"+string(d)+"'")
		return
	}
	log.Printf("got answer from %s", sig.Peer)

	// Send AddICECandidate to remote peer
	pc.OnICECandidate(func(i *webrtc.ICECandidate) {
		if i != nil {
			log.Println("ICECandidate:", i)
			signal.WriteCandidate(peer, i.ToJSON())
		} else {
			log.Println("collection of candidates is finished")
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

	// Get remote ICECandidate
	GetICECandidates(signal, pc)

	// Close signal server connection
	// signal.Close()
	<-wait

	return
}

// Get remote ICECandidates
func GetICECandidates(signal *teowebrtc_signal_client.SignalClient,
	pc *webrtc.PeerConnection) {
	for {
		sig, err := signal.WaitSignal()
		if err != nil {
			break
		}

		// Unmarshal ICECandidate signal
		data, err := json.Marshal(sig.Data)
		if err != nil || len(data) == 0 || string(data) == "null" {
			log.Println("all ICECandidate processed")
			break
		}

		// Convert to ICECandidateInit
		var ii webrtc.ICECandidateInit
		log.Printf("got ICECandidate from: %s\n", sig.Peer)
		err = json.Unmarshal(data, &ii)
		if err != nil {
			log.Println("can't Unmarshal ICECandidateInit, error:", err)
			continue
		}

		// Add servers ICECandidate
		err = pc.AddICECandidate(ii)
		if err != nil {
			log.Println("can't add ICECandidateInit, error:", err)
		}
	}
}

func NewDataChannel(dc *webrtc.DataChannel) *DataChannel {
	return &DataChannel{dc, nil}
}

type DataChannel struct {
	dc   *webrtc.DataChannel
	user interface{}
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

func (d *DataChannel) Close() error {
	return d.dc.Close()
}

func (d *DataChannel) GetUser() interface{} {
	return d.user
}

func (d *DataChannel) SetUser(user interface{}) {
	d.user = user
	// log.Printf("user type %T\n", user)
}
