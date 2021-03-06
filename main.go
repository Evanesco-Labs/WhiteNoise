package main

import (
	"context"
	"fmt"
	"github.com/Evanesco-Labs/WhiteNoise/cmd/chat"
	"github.com/Evanesco-Labs/WhiteNoise/common/account"
	"github.com/Evanesco-Labs/WhiteNoise/common/config"
	"github.com/Evanesco-Labs/WhiteNoise/common/log"
	"github.com/Evanesco-Labs/WhiteNoise/crypto"
	"github.com/Evanesco-Labs/WhiteNoise/network"
	"github.com/Evanesco-Labs/WhiteNoise/sdk"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var node *network.Node
var wnSDK *sdk.WhiteNoiseClient
var (
	PortFlag = cli.IntFlag{
		Name:  "port, p",
		Value: 3331,
	}

	BootStrapFlag = cli.StringFlag{
		Name:  "bootstrap, b",
		Usage: "PeerId of the node to bootstrap from.",
		Value: "",
	}
	NodeFlag = cli.StringFlag{
		Name:  "node, n",
		Usage: "PeerId of the node to connect to.",
		Value: "",
	}
	ModeFlag = cli.BoolFlag{
		Name:     "client, c",
		Usage:    "Build a client node if flag is on and MainNet Node by default.",
		Required: false,
		Hidden:   false,
	}
	BootFlag = cli.BoolFlag{
		Name:     "boot",
		Usage:    "Build a boot node if flag is on and MainNet Node by default.",
		Required: false,
		Hidden:   false,
	}
	LogLevelFlag = cli.IntFlag{
		Name:  "log, l",
		Value: 2,
	}
	NickFlag = cli.StringFlag{
		Name:  "nick",
		Usage: "Set nick name for chat example client",
		Value: "Alice",
	}

	WhiteListFlag = cli.BoolFlag{
		Name:     "whitelist",
		Usage:    "Only serves clients in the whitelist.yml",
		Required: false,
		Hidden:   false,
	}

	AccountFromFileFlag = cli.StringFlag{
		Name:  "account, acc",
		Usage: "Load WhiteNoise account from key file at this path",
		Value: "",
	}

	KeyFlag = cli.StringFlag{
		Name:  "keytype, ktype",
		Usage: "Set key type",
		Value: "ed25519",
	}
)

func main() {
	if err := initApp().Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func initApp() *cli.App {
	app := cli.NewApp()
	app.Usage = " whitenoise protocol implement"
	app.Commands = []cli.Command{
		{
			Name:   "start",
			Usage:  "Start Service",
			Action: Start,
			Flags: []cli.Flag{
				PortFlag,
				BootStrapFlag,
				ModeFlag,
				LogLevelFlag,
				BootFlag,
				WhiteListFlag,
				AccountFromFileFlag,
				KeyFlag,
			},
		},

		{
			Name:   "chat",
			Usage:  "Start chat",
			Action: StartChat,
			Flags: []cli.Flag{
				BootStrapFlag,
				NodeFlag,
				LogLevelFlag,
				NickFlag,
				AccountFromFileFlag,
				KeyFlag,
			},
		},
	}
	return app
}

func Start(ctx *cli.Context) {
	port := ctx.Int("port")
	bootstrap := ctx.String("bootstrap")
	clientMode := ctx.Bool("client")
	bootMode := ctx.Bool("boot")
	whitelist := ctx.Bool("whitelist")
	pemPath := ctx.String("account")
	keyTypeStr := ctx.String("keytype")

	//decode keytype
	keyType := 0
	switch keyTypeStr {
	case "ed25519":
		keyType = crypto.Ed25519
	case "secp256k1":
		keyType = crypto.Secpk1
	case "ecdsa":
		keyType = crypto.ECDSA
	default:
		keyType = crypto.Ed25519
	}

	logLevel := ctx.Int("log")
	log.InitLog(logLevel, os.Stdout, log.PATH)
	con := context.Background()
	cfg := config.NetworkConfig{
		RendezvousString: "whitenoise",
		ListenHost:       "127.0.0.1",
		ListenPort:       port,
		BootStrapPeers:   bootstrap,
		Mode:             config.ServerMode,
		WhiteList:        whitelist,
	}

	if clientMode {
		cfg.Mode = config.ClientMode
	}

	if bootMode {
		cfg.Mode = config.BootMode
	}

	// read whitelist from config
	if whitelist {
		InitWhiteList()
	}

	acc := account.GetAccountFromFile(pemPath)
	if acc == nil || acc.KeyType != keyType {
		switch keyType {
		case crypto.ECDSA:
			acc = account.GetAccount(crypto.ECDSA)
		case crypto.Ed25519:
			acc = account.GetAccount(crypto.Ed25519)
		case crypto.Secpk1:
			acc = account.GetAccount(crypto.Secpk1)
		default:
			panic("key type not support")
		}
	}

	var err error
	node, err = network.NewNode(con, &cfg, acc)
	if err != nil {
		panic(err)
	}
	node.Start(&cfg)
	waitToExit()
}

func StartChat(ctx *cli.Context) {
	logLevel := ctx.Int("log")
	log.InitLog(logLevel, os.Stdout, log.PATH)
	bootstrap := ctx.String("bootstrap")
	n := ctx.String("node")
	con := context.Background()
	nick := ctx.String("nick")
	pemPath := ctx.String("account")
	keyTypeStr := ctx.String("keytype")

	//decode keytype
	keyType := 0
	switch keyTypeStr {
	case "ed25519":
		keyType = crypto.Ed25519
	case "secp256k1":
		keyType = crypto.Secpk1
	case "ecdsa":
		keyType = crypto.ECDSA
	default:
		keyType = crypto.Ed25519
	}

	sdk.BootStrapPeers = bootstrap

	acc := account.GetAccountFromFile(pemPath)
	if acc == nil || acc.KeyType != keyType {
		switch keyType {
		case crypto.ECDSA:
			acc = account.GetAccount(crypto.ECDSA)
		case crypto.Ed25519:
			acc = account.GetAccount(crypto.Ed25519)
		case crypto.Secpk1:
			acc = account.GetAccount(crypto.Secpk1)
		default:
			panic("key type not support")
		}
	}

	var err error
	wnSDK, err = sdk.NewClient(con, acc)
	if err != nil {
		panic(err)
	}

	peers, err := wnSDK.GetMainNetPeers(10)
	if err != nil {
		panic(err)
	}
	if len(peers) == 0 {
		panic("No peers exist")
	}

	index := rand.New(rand.NewSource(time.Now().UnixNano())).Int() % len(peers)
	entry := peers[index]
	log.Info("entry:", entry.String(), ",index:", index)
	err = wnSDK.Register(entry)
	if err != nil {
		panic(err)
	}
	time.Sleep(time.Millisecond * 100)
	if n != "" {
		_, sessionID, err := wnSDK.Dial(n)
		if err != nil {
			panic(err)
		}
		log.Debug("NewCircuit done")
		chat.Chat(nick, wnSDK.GetWhiteNoiseID(), sessionID, wnSDK)
		waitToExit()
	} else {
		chat.Chat(nick, wnSDK.GetWhiteNoiseID(), "", wnSDK)
		waitToExit()
	}
}

func InitWhiteList() {
	config.WhiteListPeers = make(map[peer.ID]bool)
	var ymlConfig = config.YmlConfig{Whitelist: make([]string, 0)}

	whitelistConfig, err := ioutil.ReadFile("./whitelist.yml")
	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal(whitelistConfig, &ymlConfig)
	if err != nil {
		panic(err)
	}
	for _, c := range ymlConfig.Whitelist {
		whiteNoiseID, err := crypto.WhiteNoiseIDfromString(c)
		if err != nil {
			panic(err)
		}
		peerId, err := whiteNoiseID.GetPeerID()
		if err != nil {
			panic(err)
		}
		config.WhiteListPeers[peerId] = true
	}
}

func waitToExit() {
	exit := make(chan bool, 0)
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		for sig := range sc {
			fmt.Printf("received exit signal:%v", sig.String())
			close(exit)
			break
		}
	}()
	<-exit
}
