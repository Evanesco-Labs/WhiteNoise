module github.com/Evanesco-Labs/WhiteNoise

go 1.14

require (
	github.com/AsynkronIT/protoactor-go v0.0.0-20210305101446-d68990342ece
	github.com/Evanesco-Labs/go-evanesco v0.0.3-0.20211027011424-312266f5fae1
	github.com/asaskevich/EventBus v0.0.0-20200907212545-49d423059eef
	github.com/btcsuite/btcd v0.20.1-beta
	github.com/flynn/noise v0.0.0-20180327030543-2492fe189ae6
	github.com/gdamore/tcell/v2 v2.2.1
	github.com/gogo/protobuf v1.3.2
	github.com/golang/protobuf v1.5.2
	github.com/libp2p/go-buffer-pool v0.0.2
	github.com/libp2p/go-libp2p v0.13.0
	github.com/libp2p/go-libp2p-core v0.8.0
	github.com/libp2p/go-libp2p-kad-dht v0.11.1
	github.com/libp2p/go-libp2p-noise v0.1.3
	github.com/libp2p/go-libp2p-pubsub v0.4.1
	github.com/libp2p/go-msgio v0.0.6
	github.com/magiconair/properties v1.8.0
	github.com/mattn/go-sqlite3 v1.14.6
	github.com/mr-tron/base58 v1.2.0
	github.com/multiformats/go-multiaddr v0.3.1
	github.com/nspcc-dev/neofs-crypto v0.3.0
	github.com/rivo/tview v0.0.0-20210427112837-09cec83b1732
	github.com/syndtr/goleveldb v1.0.1-0.20210305035536-64b5b1c73954
	github.com/urfave/cli v1.22.1
	go.dedis.ch/kyber/v3 v3.0.9
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2
	google.golang.org/protobuf v1.26.0
	gopkg.in/yaml.v2 v2.4.0
)

replace go.dedis.ch/kyber/v3 v3.0.9 => github.com/Evanesco-Labs/kyber v0.0.0-20210624090903-1001bc29e5a9
