module pol-openwrt

go 1.27

replace bpb/scanner => ./third_party/bpb/scanner

require (
	bpb/networking v0.0.0
	bpb/scanner v0.0.0
	golang.org/x/crypto v0.54.0
	golang.org/x/net v0.56.0
)

require (
	github.com/quic-go/quic-go v0.65.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
)

replace bpb/networking => ./third_party/bpb/networking
