# Chat Exmaple

## Build

Building WhiteNoise require a Go compiler (version 1.14 or later). Then, simply run

```
make
```

## Start WhiteNoise Network

#### Start Bootstrap Node

Start WhiteNoise node in a Bootstrap mode, run

```shell
$ WhiteNoise start --log 2 --port 3331 --boot
```

Set **LogLevel** with the  `--log` flag, port to listen WhiteNoise network with the `--port` flag, port to listen rpc.

After successfully start a mainnet node, three main INFOs are shown in log as the following.

```verilog
 [INFO ] GID 1, [account] get default account from leveldb
 [INFO ] GID 1, [network] WhiteNoiseID: 245GY8skCWpVb1HLCzsYZS6nifUNFdDUDTjXyPzWLHguk
 [INFO ] GID 1, [noise] PeerId: QmXkCpR1CtDqPWQ7RrUgSg2aikFPMFbk8oZYo1wRn3wLGH
 [INFO ] GID 1, [noise] MultiAddrs:
/ip4/127.0.0.1/tcp/3331/p2p/QmdLEFWxMNZ5dKGKNn8tJHZG2RDnMXrzBkp94heQeUZYCr
```

WhitenosieID PeerID are shown in log. They remain the same if you start in the same directory.

#### Start WhiteNoise Node

Start WhiteNoise node in a Default mode, run

```shell
$ WhiteNoise start --log 2 --port 3332 --bootstrap /ip4/127.0.0.1/tcp/3331/p2p/QmdLEFWxMNZ5dKGKNn8tJHZG2RDnMXrzBkp94heQeUZYCr
```

It is basically the same as bootstrap node, except add the `--bootstrap` flag set the **MultiAddrs** of the node to bootstrap from.



## Chat Client

In this example, clients use WhiteNoise Network to build circuit for P2P instant chatting. This example may help get better understand of WhiteNoise and its usage. We don't have to specially build this example, because it was built together with the Golang WhiteNoise implementation. But you have to start a WhiteNoise Mainnet with at least 6 nodes or know a Bootstrap node's **Multiaddrs** of such a network.

You can test the example as follows

1. Start a Client as Answer

   First, start a WhiteNoise client as the answer of a circuit. Set a nickname with flag `--nick` and a bootstrap node with flag `--bootstrap`.

   ```shell
   $ WhiteNoise chat -l 4 --nick Bob -b /ip4/127.0.0.1/tcp/3331/p2p/QmdLEFWxMNZ5dKGKNn8tJHZG2RDnMXrzBkp94heQeUZYCr
   ```

2. Start a Client as Caller

   Similarly, we start a client as caller, set one more flag `--node` as the WhiteNoiseID of the answer we want to connect.

   ```shell
   ./WhiteNoise chat -l 4 --nick ALice -n iLthZzAPC7BkVoxHTPQ84FDs7wHU86Vqm1LmhvYNf2Kt -b /ip4/127.0.0.1/tcp/3331/p2p/QmdLEFWxMNZ5dKGKNn8tJHZG2RDnMXrzBkp94heQeUZYCr
   ```

After starting these two clients, we get two terminal UIs. Then we can start chatting through multi-hop circuit of WhiteNoise Network.
