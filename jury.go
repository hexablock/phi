package phi

import (
	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexatype"
)

// SimpleJury implements a Jury interface.  It returns the first n nodes
// available for a given key.
type SimpleJury struct {
	dht DHT
}

// RegisterDHT registers a dht interface required to get participants
func (jury *SimpleJury) RegisterDHT(dht DHT) {
	jury.dht = dht
}

// Participants gets the AffinityGroup group for the key and returns the nodes
// in that group as participants
func (jury *SimpleJury) Participants(key []byte, min int) ([]*hexalog.Participant, error) {
	nodes, err := jury.dht.LookupNodes(key, min)
	if err != nil {
		return nil, err
	}
	if len(nodes) < min {
		return nil, hexatype.ErrInsufficientPeers
	}

	participants := make([]*hexalog.Participant, 0, len(nodes))
	for i, n := range nodes {
		pcp := jury.participantFromNode(n, int32(i), 0)
		participants = append(participants, pcp)
	}

	return participants, nil
}

func (jury *SimpleJury) participantFromNode(n *hexatype.Node, p int32, i int32) *hexalog.Participant {
	meta := n.Metadata()

	return &hexalog.Participant{
		ID:       n.ID,
		Host:     meta["hexalog"],
		Priority: p,
		Index:    i,
	}
}
