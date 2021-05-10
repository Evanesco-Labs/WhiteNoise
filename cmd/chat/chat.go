package chat

import (
	"context"
	"fmt"
	"os"
	"whitenoise/sdk"
)

func Chat(nick, id, room string, sdk *sdk.WhiteNoiseClient) {
	// parse some flags to set our nickname and the room to join
	ctx := context.Background()

	// join the chat room
	cr, err := JoinChatRoom(ctx, nick, id, room, sdk)
	if err != nil {
		panic(err)
	}

	// draw the UI
	ui := NewChatUI(cr)
	ui.service = sdk

	go cr.readLoop()

	if err = ui.Run(); err != nil {
		printErr("error running text UI: %s", err)
	}
}

// printErr is like fmt.Printf, but writes to stderr.
func printErr(m string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, m, args...)
}
