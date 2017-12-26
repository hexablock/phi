package phi

import (
	"encoding/binary"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/memberlist"

	"github.com/hexablock/go-kelips"
	"github.com/hexablock/hexatype"
	"github.com/hexablock/log"
	"github.com/hexablock/vivaldi"
)

// AckPayload sends local node coordinates. It satisfies the PingDelegate
// interface
func (del *delegate) AckPayload() []byte {
	ltime := del.ltime.Time()
	ltb := make([]byte, 8)
	binary.BigEndian.PutUint64(ltb, uint64(ltime))

	coord := del.coord.GetCoordinate()
	b, err := proto.Marshal(coord)
	if err != nil {
		log.Println("[ERROR] Failed marshal coordinate:", err)
	}
	return append(ltb, b...)
}

func (del *delegate) updateCoords(id string, other *vivaldi.Coordinate, rtt time.Duration) error {
	local, err := del.coord.Update(id, other, rtt)
	if err != nil {
		return err
	}

	// Update local coordinates
	tuple := kelips.TupleHost(del.local.Address)
	if err = del.dht.PingNode(tuple.String(), local, 0); err != nil {
		log.Println("[ERROR]", err)
	}

	// Update remote coordinates
	if err = del.dht.PingNode(id, other, rtt); err != nil {
		log.Println("[ERROR]", err)
	}

	return err
}

// NotifyPingComplete updates local coordinate based on remote information. It
// satisfies the PingDelegate interface
func (del *delegate) NotifyPingComplete(node *memberlist.Node, rtt time.Duration, payload []byte) {
	ltb := payload[:8]
	ltime := binary.BigEndian.Uint64(ltb)
	del.ltime.Witness(hexatype.LamportTime(ltime))

	var other vivaldi.Coordinate
	err := proto.Unmarshal(payload[8:], &other)
	if err != nil {
		log.Println("[ERROR] Failed marshal coordinate:", err)
		return
	}

	var remoteNode hexatype.Node
	if err = proto.Unmarshal(node.Meta, &remoteNode); err != nil {
		log.Println("[ERROR]", err)
		return
	}

	if rtt > 0 {
		//tuple := kelips.TupleHost(remoteNode.Address)
		//if err = phi.updateCoords(tuple.String(), other.Clone(), rtt); err != nil {
		if err = del.updateCoords(remoteNode.Host(), other.Clone(), rtt); err != nil {
			log.Println("[ERROR]", err)
		}
	}
}
