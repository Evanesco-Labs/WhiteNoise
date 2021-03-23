module whitenoise

go 1.14

require (
	//github.com/AsynkronIT/goconsole v0.0.0-20160504192649-bfa12eebf716
	github.com/AsynkronIT/protoactor-go v0.0.0-20210125121722-bab29b9c335d
	github.com/HouMYt/go-libp2p-pubsub v0.5.1
	github.com/flynn/noise v0.0.0-20180327030543-2492fe189ae6
	github.com/gdamore/tcell/v2 v2.1.0
	github.com/golang/protobuf v1.4.3
	github.com/libp2p/go-buffer-pool v0.0.2
	github.com/libp2p/go-libp2p v0.13.0
	github.com/libp2p/go-libp2p-core v0.8.0
	github.com/libp2p/go-libp2p-kad-dht v0.11.1
	github.com/libp2p/go-libp2p-noise v0.1.1
	github.com/libp2p/go-libp2p-pubsub v0.4.1 // indirect
	github.com/mattn/go-sqlite3 v1.14.6
	github.com/mr-tron/base58 v1.2.0
	github.com/multiformats/go-multiaddr v0.3.1
	github.com/rivo/tview v0.0.0-20210125085121-dbc1f32bb1d0
	github.com/stretchr/testify v1.6.1
	github.com/syndtr/goleveldb v1.0.0
	github.com/urfave/cli v1.22.5
	golang.org/x/crypto v0.0.0-20200423211502-4bdfaf469ed5
	google.golang.org/grpc v1.35.0
	google.golang.org/protobuf v1.25.0
	github.com/libp2p/go-msgio v0.0.6
)

//replace github.com/libp2p/go-libp2p-pubsub => /github.com/HouMYt/go-libp2p-pubsub
