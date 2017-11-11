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
	// Default block replicas
	Replicas int

	// Data directory
	DataDir string

	// Wal buffer size when seeding data
	WalSeedBuffSize int

	// Wal parallel go-routines for seeding
	WalSeedParallel int

	// Any existing peers. This will automatically cause the node to join the
	// cluster
	Peers []string

	Memberlist *memberlist.Config
	Hexalog    *hexalog.Config
	DHT        *kelips.Config

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
		Replicas:        2,
		WalSeedBuffSize: 32,
		WalSeedParallel: 2,
		Peers:           []string{},
		Hexalog:         hexalog.DefaultConfig(""),
		DHT:             kelips.DefaultConfig(""),
		GRPCServer:      grpc.NewServer(),
	}
	conf.Hexalog.Votes = 2
	conf.SetHashFunc(sha256.New)

	return conf
}