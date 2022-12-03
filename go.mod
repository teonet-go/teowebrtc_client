module github.com/teonet-go/teowebrtc_client

// For testing
// replace github.com/kirill-scherba/teowebrtc/teowebrtc_signal_client => ../teowebrtc_signal_client

go 1.16

require (
	github.com/pion/webrtc/v3 v3.1.1
	github.com/teonet-go/teowebrtc_signal_client v0.0.6
	golang.org/x/crypto v0.0.0-20210921155107-089bfa567519 // indirect
	golang.org/x/net v0.0.0-20210929193557-e81a3d93ecf6 // indirect
	golang.org/x/sys v0.0.0-20210930141918-969570ce7c6c // indirect
)
