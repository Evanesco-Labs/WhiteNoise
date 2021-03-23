package chat

import (
	"context"
	"fmt"
	"os"
	"whitenoise/network"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
)

func Chat(nick, room string, node *network.Node) {
	// parse some flags to set our nickname and the room to join
	ctx := context.Background()

	// use the nickname from the cli flag, or a default if blank
	if len(nick) == 0 {
		nick = defaultNick(node.Host().ID())
	}

	// join the chat room
	cr, err := JoinChatRoom(ctx, node.Host().ID(), nick, room)
	if err != nil {
		panic(err)
	}

	// draw the UI
	ui := NewChatUI(cr)
	ui.service = node

	go cr.readLoop()

	if err = ui.Run(); err != nil {
		printErr("error running text UI: %s", err)
	}
}

// printErr is like fmt.Printf, but writes to stderr.
func printErr(m string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, m, args...)
}

// defaultNick generates a nickname based on the $USER environment variable and
// the last 8 chars of a peer ID.
func defaultNick(p peer.ID) string {
	return fmt.Sprintf("%s-%s", os.Getenv("USER"), shortID(p))
}

// shortID returns the last 8 chars of a base58-encoded peer id.
func shortID(p peer.ID) string {
	pretty := p.Pretty()
	return pretty[:8]
}

// discoveryNotifee gets notified when we find a new peer via mDNS discovery
type discoveryNotifee struct {
	h host.Host
}

// HandlePeerFound connects to peers discovered via mDNS. Once they're connected,
// the PubSub system will automatically start interacting with them if they also
// support PubSub.
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	fmt.Printf("discovered new peer %s\n", pi.ID.Pretty())
	err := n.h.Connect(context.Background(), pi)
	if err != nil {
		fmt.Printf("error connecting to peer %s: %s\n", pi.ID.Pretty(), err)
	}
}
