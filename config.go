package phi

import (
	"crypto/sha256"
	"hash"

	"google.golang.org/grpc"

	"github.com/hashicorp/memberlist"
	kelips "github.com/hexablock/go-kelips"
	"github.com/hexablock/hexalog"
)

// Config is the fidias config
type Config struct {
	// Block replicas
	Replicas int

	// Data directory
	DataDir string

	// WAL buffer size when seeding data on bootstrap
	WalSeedBuffSize int

	// WAL parallel go-routines for seeding
	WalSeedParallel int

	// Any existing peers. This will automatically cause the node to join the
	// cluster
	Peers []string

	// Membership and fault-tolerance
	Memberlist *memberlist.Config

	// WAL
	Hexalog *hexalog.Config

	DHT *kelips.Config

	// Hexalog jury selection algorithm to use
	Jury Jury

	// Grpc server to allow user services to be registered
	GRPCServer *grpc.Server
}

// HashFunc returns the hash function used for the fidias as a whole.  These
// will always match between the the dht, blox and hexalog
func (config *Config) HashFunc() hash.Hash {
	return config.Hexalog.Hasher()
}

// SetHashFunc sets the hash function for hexalog and the dht
func (config *Config) SetHashFunc(hf func() hash.Hash) {
	config.Hexalog.Hasher = hf
	config.DHT.HashFunc = hf
}

// DefaultConfig returns a minimally required config
func DefaultConfig() *Config {
	conf := &Config{
		Replicas:        1,
		WalSeedBuffSize: 32,
		WalSeedParallel: 2,
		Peers:           []string{},
		Hexalog:         hexalog.DefaultConfig(""),
		DHT:             kelips.DefaultConfig(""),
		GRPCServer:      grpc.NewServer(),
		Jury:            &SimpleJury{},
	}
	conf.DHT.NumGroups = 3
	conf.Hexalog.Votes = 2
	conf.SetHashFunc(sha256.New)

	return conf
}
