package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/urfave/cli"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/runtime/protoimpl"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
	"whitenoise/cmd/chat"
	"whitenoise/common"
	"whitenoise/common/config"
	"whitenoise/common/log"
	"whitenoise/network"
	"whitenoise/secure"
)

var node *network.Node
var (
	PortFlag = cli.IntFlag{
		Name:  "port, p",
		Value: 3331,
	}
	RpcPortFlag = cli.StringFlag{
		Name:  "rpcport, rpc",
		Value: "6661",
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
	ProxyIdFlag = cli.StringFlag{
		Name:  "proxy",
		Usage: "Register to proxy with this peerId.",
		Value: "",
	}
	DesIdFlag = cli.StringFlag{
		Name:  "des, d",
		Usage: "PeerId of the destination of the circuit.",
		Value: "",
	}
	SessionIdFlag = cli.StringFlag{
		Name:  "session, s",
		Value: "",
	}
	RelayIdFlag = cli.StringFlag{
		Name:  "relay, r",
		Value: "",
	}
	JointIdFlag = cli.StringFlag{
		Name:  "joint, j",
		Value: "",
	}
	MsgFlag = cli.StringFlag{
		Name:  "msg, m",
		Usage: "Message to be sent on a circuit",
		Value: "",
	}
	MyRoleFlag = cli.IntFlag{
		Name:  "my",
		Value: int(common.CallerRole),
	}
	OtherRoleFlag = cli.IntFlag{
		Name:  "other",
		Value: int(common.EntryRole),
	}
	LogLevelFlag = cli.IntFlag{
		Name:  "log, l",
		Value: 2,
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
				RpcPortFlag,
				PortFlag,
				BootStrapFlag,
				ModeFlag,
				LogLevelFlag,
			},
		},

		{
			Name:   "chat",
			Usage:  "Start chat",
			Action: StartChat,
			Flags: []cli.Flag{
				RpcPortFlag,
				PortFlag,
				BootStrapFlag,
				NodeFlag,
				LogLevelFlag,
			},
		},
		{
			Name:   "reg",
			Usage:  "register proxy",
			Action: Register,
			Flags: []cli.Flag{
				RpcPortFlag,
				ProxyIdFlag,
				LogLevelFlag,
			},
		},
		{
			Name:   "session",
			Usage:  "gen new session to peer",
			Action: GenSession,
			Flags: []cli.Flag{
				RpcPortFlag,
				DesIdFlag,
				SessionIdFlag,
				MyRoleFlag,
				OtherRoleFlag,
				LogLevelFlag,
			},
		},
		{
			Name:   "extend",
			Usage:  "extend session",
			Action: ExtendSession,
			Flags: []cli.Flag{
				RpcPortFlag,
				JointIdFlag,
				RelayIdFlag,
				SessionIdFlag,
				LogLevelFlag,
			},
		},
		{
			Name:   "circuit",
			Usage:  "gen new circuit to peer",
			Action: GenCircuit,
			Flags: []cli.Flag{
				RpcPortFlag,
				DesIdFlag,
				SessionIdFlag,
				LogLevelFlag,
			},
		},
		{
			Name:   "gossip",
			Usage:  "gossip",
			Action: Gossip,
			Flags: []cli.Flag{
				RpcPortFlag,
				DesIdFlag,
				JointIdFlag,
				SessionIdFlag,
				LogLevelFlag,
			},
		},
		{
			Name:   "relay",
			Usage:  "send relay message",
			Action: SendRelay,
			Flags: []cli.Flag{
				RpcPortFlag,
				MsgFlag,
				SessionIdFlag,
				LogLevelFlag,
			},
		},
	}
	return app
}

func Start(ctx *cli.Context) {
	RpcPort := ctx.String("rpcport")
	port := ctx.Int("port")
	bootstrap := ctx.String("bootstrap")
	clientMode := ctx.Bool("client")

	logLevel := ctx.Int("log")
	log.InitLog(logLevel, os.Stdout, log.PATH)

	con := context.Background()
	cfg := config.NetworkConfig{
		RendezvousString: "whitenoise",
		ListenHost:       "127.0.0.1",
		ListenPort:       port,
		BootStrapPeers:   bootstrap,
		Mode:             config.ServerMode,
	}
	if clientMode {
		cfg.Mode = config.ClientMode
	}

	var err error
	node, err = network.NewNode(con, &cfg)
	if err != nil {
		panic(err)
	}
	node.Start(&cfg)
	StartRpc(RpcPort)
	waitToExit()
}

func StartChat(ctx *cli.Context) {
	logLevel := ctx.Int("log")
	log.InitLog(logLevel, os.Stdout, log.PATH)

	RpcPort := ctx.String("rpcport")
	port := ctx.Int("port")
	bootstrap := ctx.String("bootstrap")
	n := ctx.String("node")
	con := context.Background()
	cfg := config.NetworkConfig{
		RendezvousString: "whitenoise",
		ListenHost:       "127.0.0.1",
		ListenPort:       port,
		BootStrapPeers:   bootstrap,
		Mode:             config.ClientMode,
	}

	var err error
	node, err = network.NewNode(con, &cfg)
	if err != nil {
		panic(err)
	}
	node.Start(&cfg)
	log.Info("start finished")
	go StartRpc(RpcPort)
	log.Info("begin to print peer...")

	//强制同步DHT表
	t := time.NewTicker(time.Second * 10)
	select {
	case <-t.C:
		log.Warn("Timeout for refresh route table with default 10 sec")
		break
	case <-node.DHTService.Dht().ForceRefresh():
		log.Info("refresh route table done")
	}

	peers := node.DHTService.Dht().RoutingTable().ListPeers()

	// 获得可用的入口节点
	index := rand.New(rand.NewSource(time.Now().UnixNano())).Int() % len(peers)
	entry := peers[index]
	log.Info("entry:", entry.String(), ",index:", index)

	// 连接入口节点
	var isConnection bool
	for _, p := range node.NoiseService.Host().Peerstore().Peers() {
		if p.String() == entry.String() {
			isConnection = true
			break
		}
	}

	if isConnection == false {
		if conns := node.NoiseService.Host().Network().ConnsToPeer(entry); conns != nil {
			log.Info("conns:", conns)
		}
	}

	if err := node.NoiseService.RegisterProxy(entry); err != nil {
		log.Error("register proxy err:", err.Error())
	}

	if n != "" {
		remoteID, err := peer.Decode(n)
		if err == nil {
			if err := node.NoiseService.NewCircuit(remoteID, "chat-random"); err != nil {
			} else {
				log.Info("remote.addr:", remoteID)
			}
		} else {
			log.Error("ID from string err:", err.Error())
		}

		chat.Chat(node.NoiseService.Host().ID().Pretty()[:8], "chat-random", node)
		waitToExit()
	} else {
		chat.Chat(node.NoiseService.Host().ID().Pretty()[:8], "chat-random", node)
		waitToExit()
	}
}

func StartRpc(RpcPort string) {
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%s", RpcPort))
	if err != nil {
		panic("failed to listen: " + RpcPort)
	}
	server := grpc.NewServer()
	RegisterRpcServer(server, newServer())
	server.Serve(lis)
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

func Register(ctx *cli.Context) error {
	logLevel := ctx.Int("log")
	log.InitLog(logLevel, os.Stdout, log.PATH)

	proxyString := ctx.String("proxy")
	RpcPort := ctx.String("rpcport")
	_, err := peer.Decode(proxyString)
	if err != nil {
		PrintErrorMsg("PeerId decode err %v", err)
		return err
	}
	client, err := NewClient(RpcPort)
	if err != nil {
		PrintErrorMsg("New client err %v", err)
		return err
	}
	regCmd := RegProxyCmd{
		PeerId: proxyString,
	}

	con, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := client.RegisterProxy(con, &regCmd)
	if err != nil {
		PrintErrorMsg("Client request err %v", err)
		return err
	} else {
		PrintResMsg("result %v %v", res.Ok, string(res.Data))
		return nil
	}
}

func GenSession(ctx *cli.Context) error {
	logLevel := ctx.Int("log")
	log.InitLog(logLevel, os.Stdout, log.PATH)

	RpcPort := ctx.String("rpcport")
	client, err := NewClient(RpcPort)
	if err != nil {
		PrintErrorMsg("New client err %v", err)
		return err
	}
	con, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	peerId := ctx.String("des")
	sessionId := ctx.String("session")
	myRole := ctx.Int("my")
	otherRole := ctx.Int("other")

	cmd := GenSessionCmd{
		state:     protoimpl.MessageState{},
		PeerId:    peerId,
		SessionId: sessionId,
		MyRole:    int32(myRole),
		OtherRole: int32(otherRole),
	}

	res, err := client.GenSessiontoPeer(con, &cmd)
	if err != nil {
		PrintErrorMsg("Client request err %v", err)
		return err
	} else {
		PrintResMsg("result %v %v", res.Ok, string(res.Data))
		return nil
	}
}

func ExtendSession(ctx *cli.Context) error {
	logLevel := ctx.Int("log")
	log.InitLog(logLevel, os.Stdout, log.PATH)

	RpcPort := ctx.String("rpcport")
	client, err := NewClient(RpcPort)
	if err != nil {
		PrintErrorMsg("New client err %v", err)
		return err
	}
	con, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	join := ctx.String("joint")
	relay := ctx.String("relay")
	sessionId := ctx.String("session")

	cmd := ExtendCmd{
		Relay:     relay,
		Joint:     join,
		SessionId: sessionId,
	}

	res, err := client.ExtendSession(con, &cmd)
	if err != nil {
		PrintErrorMsg("Client request err %v", err)
		return err
	} else {
		PrintResMsg("result %v %v", res.Ok, string(res.Data))
		return nil
	}
}

func GenCircuit(ctx *cli.Context) error {
	logLevel := ctx.Int("log")
	log.InitLog(logLevel, os.Stdout, log.PATH)

	RpcPort := ctx.String("rpcport")
	client, err := NewClient(RpcPort)
	if err != nil {
		PrintErrorMsg("New client err %v", err)
		return err
	}
	con, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	des := ctx.String("des")
	sessionId := ctx.String("session")

	cmd := GenCircuitCmd{
		Des:       des,
		SessionId: sessionId,
	}

	res, err := client.GenCircuit(con, &cmd)
	if err != nil {
		PrintErrorMsg("Client request err %v", err)
		return err
	} else {
		PrintResMsg("result %v %v", res.Ok, string(res.Data))
		return nil
	}
}

func Gossip(ctx *cli.Context) error {
	logLevel := ctx.Int("log")
	log.InitLog(logLevel, os.Stdout, log.PATH)

	RpcPort := ctx.String("rpcport")
	client, err := NewClient(RpcPort)
	if err != nil {
		PrintErrorMsg("New client err %v", err)
		return err
	}
	con, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	joint := ctx.String("joint")
	desString := ctx.String("des")
	sessionId := ctx.String("session")

	des, err := peer.Decode(desString)
	if err != nil {
		PrintErrorMsg("Deocde des peerId err %v", err)
		return err
	}

	hash := sha256.Sum256([]byte(des.String()))

	cmd := GossipMsg{
		DesHash:   secure.EncodeID(hash[:]),
		Join:      joint,
		SessionId: sessionId,
	}

	res, err := client.GossipJoint(con, &cmd)
	if err != nil {
		PrintErrorMsg("Client request err %v", err)
		return err
	} else {
		PrintResMsg("result %v %v", res.Ok, string(res.Data))
		return nil
	}
}

func SendRelay(ctx *cli.Context) error {
	logLevel := ctx.Int("log")
	log.InitLog(logLevel, os.Stdout, log.PATH)

	RpcPort := ctx.String("rpcport")
	client, err := NewClient(RpcPort)
	if err != nil {
		PrintErrorMsg("New client err %v", err)
		return err
	}
	con, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msg := ctx.String("msg")
	sessionId := ctx.String("session")

	cmd := RelayMsg{
		SessionID: sessionId,
		Msg:       []byte(msg),
	}
	res, err := client.SendRelayMsg(con, &cmd)
	if err != nil {
		PrintErrorMsg("Client request err %v", err)
		return err
	} else {
		PrintResMsg("result %v %v", res.Ok, string(res.Data))
		return nil
	}
}

func PrintErrorMsg(format string, a ...interface{}) {
	format = fmt.Sprintf("\033[31m[ERROR] %s\033[0m\n", format) //Print error msg with red color
	fmt.Printf(format, a...)
}

func PrintResMsg(format string, a ...interface{}) {
	format = fmt.Sprintf("\033[32m[Success] %s\033[0m\n", format) //Print res msg with green color
	fmt.Printf(format, a...)
}
