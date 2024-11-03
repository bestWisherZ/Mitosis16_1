package p2p

import (
	"github.com/KyrinCode/Mitosis/config"
	"github.com/KyrinCode/Mitosis/eventbus"
)

// Protocol
type Protocol struct {
	node   *P2PNode
	config *config.Config
}

// NewProtocol create a new Protocol object
func NewProtocol(broker *eventbus.EventBus, config *config.Config) *Protocol {
	br := NewBaseReader(broker)
	node := NewP2PNode(br, config)
	return &Protocol{
		node:   node,
		config: config,
	}
}

func (p *Protocol) Start() {
	go p.node.Launch()
}

func (p *Protocol) Gossip(msg []byte, shardId uint32) {
	p.node.Gossip(msg, shardId)
}

// GShard only
func (p *Protocol) GossipAll(msg []byte) {
	if p.node.conf.ShardId != 0 {
		return
	}
	p.node.GossipAll(msg)
}

// type GossipLevel int

// const (
// 	GossipInLocalShard GossipLevel = iota
// 	GossipInLeaderNet
// 	GossipInALlShard
// )

// func (p *Protocol) Gossip(msg []byte, s GossipLevel) {
// 	switch s {
// 	case GossipInLocalShard:
// 		p.GossipInLocalShard(msg)
// 	case GossipInLeaderNet:
// 		p.GossipInLeaderNet(msg)
// 	case GossipInALlShard:
// 		p.GossipInAllShards(msg)
// 	}
// }
// func (p *Protocol) GossipInLocalShard(msg []byte) {
// 	p.shardNode.Gossip(msg)
// }

// func (p *Protocol) GossipInLeaderNet(msg []byte) {
// 	if !p.config.IsLeader {
// 		return
// 	}
// 	p.leaderNode.Gossip(msg)
// }

// func (p *Protocol) GossipInAllShards(msg []byte) {
// 	if !p.config.IsLeader {
// 		return
// 	}
// 	p.leaderNode.Gossip(msg)
// 	p.shardNode.Gossip(msg)
// }
