# whitenoise

## Build Demo
```
cd ./cmd
go build
```

## Example
### 1. Run some node services

#### 1.1 Run a node as kad-dht bootstrap
Run the bootstrap node with command:

`cmd start --port 3331 --rpcport 6661`

The MultiAddress of this bootstrap node is shown: */ip4/127.0.0.1/tcp/3331/p2p/QmVV9rBZHdnYoJZqioTKaSnEEa6KMDoNvpSNaGH8FEm8sQ*

`2021/01/29 17:00:57.307499 [INFO ] GID 1, [whitenoise] /ip4/127.0.0.1/tcp/3331/p2p/QmVV9rBZHdnYoJZqioTKaSnEEa6KMDoNvpSNaGH8FEm8sQ`
#### 1.2 Run MainNet Nodes

Start the first node with this command, we use `-b` flag to set which node to bootstrap from.

`cmd start --port 3331 --rpcport 6661 -b /ip4/127.0.0.1/tcp/3331/p2p/QmVV9rBZHdnYoJZqioTKaSnEEa6KMDoNvpSNaGH8FEm8sQ`

Info shows the PeerId of this node.

`PeerId: QmVV9rBZHdnYoJZqioTKaSnEEa6KMDoNvpSNaGH8FEm8sQ`

Then run some other nodes on different ports (at least 4 mainnet nodes in total):

`cmd start --port 3332 --rpcport 6662 -b /ip4/127.0.0.1/tcp/3331/p2p/QmVV9rBZHdnYoJZqioTKaSnEEa6KMDoNvpSNaGH8FEm8sQ`

...

#### 1.3 Run Client Node
Here we run two client node Alice and Bob with flag `--client` on. Alice at rpcport 6667 and Bob at rpcport 6668.

`cmd start --port 3337 --rpcport 6667  --client -b /ip4/127.0.0.1/tcp/3331/p2p/QmVV9rBZHdnYoJZqioTKaSnEEa6KMDoNvpSNaGH8FEm8sQ`

`cmd start --port 3338 --rpcport 6668  --client -b /ip4/127.0.0.1/tcp/3331/p2p/QmVV9rBZHdnYoJZqioTKaSnEEa6KMDoNvpSNaGH8FEm8sQ`


### 2. Register Proxy
Lets call node with rpcport 6667 *Alice*.

*Alice* choose mainnet node with peerid *QmUmQQCMFfJ7VXbSfQ43dGEaBbukR98k6rfu2Jx3kSr2R8* as her proxy, and register with following command. 

`cmd reg --rpcport 6667 --proxy QmUmQQCMFfJ7VXbSfQ43dGEaBbukR98k6rfu2Jx3kSr2R8`

This node is the *entry* for Alice to the MainNet.

Let's say *Bob* is the node with rpcport 6668, and register Bob to another proxy.

`cmd reg --rpcport 6668 --proxy Qmach4RuRESwb7vVBYscqPSLqXce68Ubx9D7ebMDrqnXYQ`
 
### 3. Build Circuit
Now Alice wants to build a circuit to Bob through the MainNet. She can send a requset to her proxy to build this circuit.

`cmd circuit --rpcport 6667 --des Qmb9Ju24kzUrT5hVeAF32q3MgcKLAydyYoEfjnVUkvnan7 --session AliceCallingBob `

The *des* flag should be Bob's PeerId, and Alice can set a sessionID for this circuit with the *session* flag.
 
### 4. Let's Chat
 
 After building the circuit, Alice and Bob can send messages to each other.
 
 `cmd relay --rpcport 6667 --session AliceCallingBob --msg HiBob`
 
 `cmd relay --rpcport 6668 --session AliceCallingBob --msg HiAlice`

## Help
There are more command and you can see helps with command

`cmd -h`