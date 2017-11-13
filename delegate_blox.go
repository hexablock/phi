package phi

import (
	"log"

	"github.com/hexablock/blox/device"
	kelips "github.com/hexablock/go-kelips"
)

// BlockSet is the blox delegate to handle block inserts to the dht
func (phi *Phi) BlockSet(index device.IndexEntry) {
	tuple := kelips.TupleHost(phi.local.Address)

	if err := phi.dht.Insert(index.ID(), tuple); err != nil {
		log.Printf("[ERROR] Failed to insert to dht: %s", err)
	}

}

// BlockRemove is the blox delegate to handle block removals from the dht
func (phi *Phi) BlockRemove(id []byte) {
	tuple := kelips.TupleHost(phi.local.Address)
	if err := phi.dht.Delete(id, tuple); err != nil {
		log.Printf("[ERROR] Failed to delete from dht: %s", err)
	}
}
