package p2p

import (
	"context"
	"sync"
	"time"

	"github.com/KyrinCode/Mitosis/config"
	"github.com/emirpasic/gods/utils"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/multiformats/go-multiaddr"
	logger "github.com/sirupsen/logrus"
)

var logP2P = logger.WithField("process", "p2p")

const (
	RShardIdPrefix      = "RShard-"
	PShardIdPrefix      = "PShard-"
	DiscoveryServiceTag = "mitosis-pubsub-mdns"
	Rendezvous          = "mitosis-pubsub-kaddht"
)

// Node encapsulation of p2p node
type P2PNode struct {
	host host.Host
	pub  map[string]*pubsub.Topic // key: shardId 每个节点要加入全部topic，因为GShard区块要广播至全网来更新topo和nodes，RShard只需要向管辖的PShard广播含区块头的交易及交易证明，PShard间节点多对多的广播（已收到的节点不再继续广播）
	sub  *pubsub.Subscription     // key: shardId 每个节点只订阅所在分片的topic

	broker *BaseReader

	conf *config.Config

	up bool // isReady
	// stop  chan bool  // switch shard
	// mutex sync.Mutex // 切换shard的时候
}

// NewNode return an node of p2p network
func NewP2PNode(broker *BaseReader, config *config.Config) *P2PNode {
	host, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	if err != nil {
		panic(err)
	}

	return &P2PNode{
		host: host,
		pub:  make(map[string]*pubsub.Topic),
		// sub:    make(map[uint32]*pubsub.Subscription),
		broker: broker,
		conf:   config,
		up:     false,
		// stop:   make(chan bool),
	}
}

// Launch start p2p service
func (n *P2PNode) Launch() {
	ctx := context.Background()
	logP2P.WithField("localPeer", n.host.ID().String()).Infof("[Node-%d-%d] starting gossip", n.conf.ShardId, n.conf.NodeId)

	multiAddr, err := multiaddr.NewMultiaddr(n.conf.Bootnode)
	if err != nil {
		panic(err)
	} else {
		// println(multiAddr.String())
	}

	discoveryPeers := []multiaddr.Multiaddr{multiAddr}

	dht, err := initDHT(ctx, n, discoveryPeers)
	if err != nil {
		panic(err)
	}

	// setup peer discovery
	go Discover(ctx, n, dht, Rendezvous)
	// setup local mDNS discovery (optional)
	// if err := setupDiscovery(n.host); err != nil {
	// 	panic(err)
	// }

	time.Sleep(10 * time.Second)

	gossipSub, err := pubsub.NewGossipSub(ctx, n.host, pubsub.WithFloodPublish(true))
	//gossipSub, err := pubsub.NewFloodSub(ctx, n.h)
	if err != nil {
		panic(err)
	}

	// parse all topics from topo
	shardTopics := []string{}
	// shardTopics = append(shardTopics, GShardIdPrefix+utils.ToString(n.conf.Topo.GShardId))
	for _, RShardId := range n.conf.Topo.RShardIds {
		shardTopics = append(shardTopics, idToTopic(RShardId))
		for _, PShardId := range n.conf.Topo.PShardIds[RShardId] {
			shardTopics = append(shardTopics, idToTopic(PShardId))
		}
	}

	// join all topics
	for _, shardTopic := range shardTopics {
		n.pub[shardTopic], err = gossipSub.Join(shardTopic)
		if err != nil {
			panic(err)
		}
		logP2P.Printf("[Node-%d-%d] joined topic-%s", n.conf.ShardId, n.conf.NodeId, shardTopic)
	}

	// subscribe only one topic
	n.sub, err = n.pub[idToTopic(n.conf.ShardId)].Subscribe(pubsub.WithBufferSize(40000))
	if err != nil {
		panic(err)
	}
	go n.subscribe(n.sub, ctx)

	n.up = true
}

func initDHT(ctx context.Context, n *P2PNode, bootstrapPeers []multiaddr.Multiaddr) (*dht.IpfsDHT, error) {
	host := n.host
	kademliaDHT, err := dht.New(ctx, host)
	if err != nil {
		return nil, err
	}

	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	for _, peerAddr := range bootstrapPeers { // dht.DefaultBootstrapPeers
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := host.Connect(ctx, *peerinfo); err != nil {
				logP2P.Errorf("[Node-%d-%d] error while connecting to node %q: %-v\n", n.conf.ShardId, n.conf.NodeId, peerinfo, err)
			} else {
				logP2P.Printf("[Node-%d-%d] connection established with bootstrap node: %q", n.conf.ShardId, n.conf.NodeId, *peerinfo)
			}
		}()
	}
	wg.Wait()

	return kademliaDHT, nil
}

func Discover(ctx context.Context, n *P2PNode, dht *dht.IpfsDHT, rendezvous string) {
	h := n.host
	var routingDiscovery = drouting.NewRoutingDiscovery(dht)

	dutil.Advertise(ctx, routingDiscovery, rendezvous)

	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:

			peers, err := routingDiscovery.FindPeers(ctx, rendezvous)
			if err != nil {
				logP2P.Fatal(err)
			}

			for p := range peers {
				if p.ID == h.ID() {
					continue
				}
				// err := h.Connect(ctx, p)
				// if err != nil {
				// 	fmt.Printf("Failed connecting to %s, error: %s\n", p.ID, err)
				// } else {
				// 	fmt.Printf("Connected to peer: %s\n", p.ID)
				// }
				if h.Network().Connectedness(p.ID) != network.Connected {
					_, err = h.Network().DialPeer(ctx, p.ID)
					if err != nil {
						logP2P.Errorf("[Node-%d-%d] failed connecting to %s, error: %s\n", n.conf.ShardId, n.conf.NodeId, p.ID.String(), err)

						// err = h.Network().ClosePeer(p.ID)
						// if err != nil {
						// 	fmt.Printf("Failed disconnecting to %s, error: %s\n", p.ID.String(), err)
						// }
						continue
					} else {
						logP2P.Printf("[Node-%d-%d] connected to peer: %s with dht", n.conf.ShardId, n.conf.NodeId, p.ID.String())
					}
				}
			}
		}
	}
}

func idToTopic(shardId uint32) string {
	if shardId <= 1000 {
		return RShardIdPrefix + utils.ToString(shardId)
	} else {
		return PShardIdPrefix + utils.ToString(shardId)
	}
}

func (n *P2PNode) Gossip(msg []byte, shardId uint32) {

	if !n.up {
		for {
			if n.up {
				// println("\n", n.up)
				break
			}
			time.Sleep(2 * time.Second)
		}
	}

	// println("node", n.conf.NodeId, "to", "shard", shardId)

	ctx := context.Background()
	if err := n.pub[idToTopic(shardId)].Publish(ctx, msg); err != nil {
		logP2P.Errorf("[Node-%d-%d] publish error:%s\n", n.conf.ShardId, n.conf.NodeId, err)
	}
}

func (n *P2PNode) GossipAll(msg []byte) {
	shardIds := []uint32{}
	// shardIds = append(shardIds, n.conf.Topo.GShardId)
	for _, RShardId := range n.conf.Topo.RShardIds {
		shardIds = append(shardIds, RShardId)
		shardIds = append(shardIds, n.conf.Topo.PShardIds[RShardId]...)
	}

	var wg sync.WaitGroup
	for _, shardId := range shardIds {
		wg.Add(1)
		go func() {
			defer wg.Done()
			n.Gossip(msg, shardId)
		}()
	}
	wg.Wait()
}

func (n *P2PNode) subscribe(subscriber *pubsub.Subscription, ctx context.Context) {
	logP2P.Printf("[Node-%d-%d] subscribing topic-%s", n.conf.ShardId, n.conf.NodeId, subscriber.Topic())
	for {
		msg, err := n.sub.Next(ctx)
		// println(n.conf.NodeId, string(msg.Message.Data))
		if err != nil {
			panic(err)
		}
		if msg.ReceivedFrom == n.host.ID() {
			continue
		}
		go n.broker.ProcessMessage(msg.ReceivedFrom.String(), msg.Message.Data)
		// println(n.conf.ShardId, "-", n.conf.NodeId, "	receive msg:	", string(msg.Message.Data), "	from	", msg.ReceivedFrom.String())
	}
}

// func (n *P2PNode) switchSubscribe(prevShardId uint32, ctx context.Context) { // 需要先更新config再调用这里
// 	n.stop <- true
// 	n.sub[prevShardId].Cancel()

// 	subscriber, err := n.pub[n.conf.ShardId].Subscribe(pubsub.WithBufferSize(40000))
// 	if err != nil {
// 		panic(err)
// 	}
// 	n.sub[n.conf.ShardId] = subscriber
// 	go n.subscribe(n.sub[n.conf.ShardId], ctx, n.stop)

// }

// discoveryNotifee gets notified when we find a new peer via mDNS discovery
type discoveryNotifee struct {
	h host.Host
}

// HandlePeerFound connects to peers discovered via mDNS. Once they're connected,
// the PubSub system will automatically start interacting with them if they also
// support PubSub.
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	// logP2P.Printf("New peer discovered: %s with mdns\n", pi.ID.String())
	err := n.h.Connect(context.Background(), pi)
	if err != nil {
		// logP2P.Printf("Error connecting to peer %s: %s\n", pi.ID.String(), err)
	} else {
		logP2P.Printf("Connected to peer %s", pi.ID.String())
	}
}

// setupDiscovery creates an mDNS discovery service and attaches it to the libp2p Host.
// This lets us automatically discover peers on the same LAN and connect to them.
func setupDiscovery(h host.Host) error {
	// setup mDNS discovery to find local peers
	s := mdns.NewMdnsService(h, DiscoveryServiceTag, &discoveryNotifee{h: h})
	return s.Start()
}
