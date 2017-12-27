package phi

import (
	"github.com/hexablock/blox/device"
	kelips "github.com/hexablock/go-kelips"
	"github.com/hexablock/log"
)

// BlockSet is the blox delegate called when new blocks are set.  It  handles
// block inserts to the dht
func (phi *Phi) BlockSet(index device.IndexEntry) {
	// Update DHT
	tuple := kelips.TupleHost(phi.local.Address)
	if err := phi.dht.Insert(index.ID(), tuple); err != nil {
		log.Printf("[ERROR] Failed to insert to dht: %s", err)
		return
	}

	//
	// TODO: trigger block processing
}

// BlockRemove is the blox delegate to handle block removals from the dht
func (phi *Phi) BlockRemove(id []byte) {
	// Update DHT
	tuple := kelips.TupleHost(phi.local.Address)
	if err := phi.dht.Delete(id, tuple); err != nil {
		log.Printf("[ERROR] Failed to delete from dht: %s", err)
		return
	}

	//
	// TODO: trigger block processing
}
