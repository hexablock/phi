package phi

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/hexablock/blox"
	"github.com/hexablock/blox/device"
	"github.com/hexablock/go-kelips"
	hexaboltdb "github.com/hexablock/hexa-boltdb"
	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexatype"
	"github.com/hexablock/vivaldi"
)

// DHT implements a distributed hash table needed to route keys
type DHT interface {
	LookupNodes(key []byte, min int) ([]*hexatype.Node, error)
	LookupGroupNodes(key []byte) ([]*hexatype.Node, error)
	Lookup(key []byte) ([]*hexatype.Node, error)
	Insert(key []byte, tuple kelips.TupleHost) error
	Delete(key []byte, tuple kelips.TupleHost) error
}

// WALTransport implements an interface for network log operations
type WALTransport interface {
	NewEntry(host string, key []byte, opts *hexalog.RequestOptions) (*hexalog.Entry, error)
	ProposeEntry(ctx context.Context, host string, entry *hexalog.Entry, opts *hexalog.RequestOptions) (*hexalog.ReqResp, error)
	GetEntry(host string, key []byte, id []byte, opts *hexalog.RequestOptions) (*hexalog.Entry, error)
}

// WAL implements an interface to provide p2p distributed consensus
type WAL interface {
	NewEntry(key []byte) (*hexalog.Entry, []*hexalog.Participant, error)
	NewEntryFrom(entry *hexalog.Entry) (*hexalog.Entry, []*hexalog.Participant, error)
	ProposeEntry(entry *hexalog.Entry, opts *hexalog.RequestOptions, retries int, retryInt time.Duration) ([]byte, *WriteStats, error)
	GetEntry(key []byte, id []byte) (*hexalog.Entry, error)
	RegisterJury(jury Jury)
}

// FSM implements a phi fsm using a dht
type FSM interface {
	Apply(entryID []byte, entry *hexalog.Entry) interface{}
	RegisterDHT(dht DHT)
}

// Phi is the core engine for a participating node
type Phi struct {
	conf *Config

	// Local node
	local hexatype.Node

	// Lamport clock
	ltime *hexatype.LamportClock

	// Virtual coorinates
	coord *vivaldi.Client

	// DHT
	dht *kelips.Kelips

	// Gossip delegate
	dlg *delegate

	// Gossip
	memberlist *memberlist.Memberlist

	// DHT enabled BlockDevice for blox API
	dev *BlockDevice

	// DHT enabled hexalog
	wal *Hexalog

	// local hexalog instance
	hexalog *hexalog.Hexalog
}

// Create creates a new Phi instance.  It inits the local node, gossip layer
// and associated delegates
func Create(conf *Config, fsm FSM) (*Phi, error) {
	// Coorinate client
	coord, err := vivaldi.NewClient(vivaldi.DefaultConfig())
	if err != nil {
		return nil, err
	}

	fid := &Phi{
		conf:  conf,
		ltime: &hexatype.LamportClock{},
		coord: coord,
	}

	// The order of initialization is important

	if err = fid.initDHT(); err != nil {
		return nil, err
	}

	if err = fid.initBlockDevice(); err != nil {
		return nil, err
	}

	if err = fid.initHexalog(fsm); err != nil {
		return nil, err
	}

	fid.init()

	if err = fid.startGrpc(); err != nil {
		return nil, err
	}

	fid.memberlist, err = memberlist.Create(conf.Memberlist)

	return fid, err
}

func (phi *Phi) init() {

	phi.dlg = &delegate{
		local:      phi.local,
		coord:      phi.coord,
		dht:        phi.dht,
		broadcasts: make([][]byte, 0),
	}

	//
	c := phi.conf.Memberlist
	c.Delegate = phi.dlg
	c.Ping = phi
	c.Events = phi.dlg
	c.Alive = phi.dlg
	c.Conflict = phi.dlg
}

func (phi *Phi) initDHT() error {
	udpAddr, err := net.ResolveUDPAddr("udp", phi.conf.DHT.AdvertiseHost)
	if err != nil {
		return err
	}

	ln, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return err
	}

	remote := kelips.NewUDPTransport(ln)
	phi.dht = kelips.Create(phi.conf.DHT, remote)
	phi.local = phi.dht.LocalNode()

	return nil
}

// must be called after dht is init'd
func (phi *Phi) initBlockDevice() error {
	ln, err := net.Listen("tcp", phi.conf.DHT.AdvertiseHost)
	if err != nil {
		return err
	}

	dir := filepath.Join(phi.conf.DataDir, "block")
	os.MkdirAll(dir, 0755)
	// Local
	//index := device.NewInmemIndex()
	index := hexaboltdb.NewBlockIndex()
	if err = index.Open(dir); err != nil {
		return err
	}
	raw, err := device.NewFileRawDevice(dir, phi.conf.HashFunc)
	if err != nil {
		return err
	}
	dev := device.NewBlockDevice(index, raw)
	dev.SetDelegate(phi)

	// Sync raw device and index
	dev.Reindex()

	// Remote
	opts := blox.DefaultNetClientOptions(phi.conf.HashFunc)
	remote := blox.NewNetTransport(opts)

	// Local and remote
	trans := blox.NewLocalNetTranport(phi.conf.DHT.AdvertiseHost, remote)

	// DHT block device
	phi.dev = NewBlockDevice(phi.conf.Replicas, phi.conf.HashFunc, trans)
	phi.dev.Register(dev)
	phi.dev.RegisterDHT(phi.dht)

	err = trans.Start(ln.(*net.TCPListener))
	return err
}

func (phi *Phi) initHexalog(fsm FSM) error {
	// Data stores
	//entries := hexalog.NewInMemEntryStore()
	//index := hexalog.NewInMemIndexStore()

	edir := filepath.Join(phi.conf.DataDir, "log", "entry")
	os.MkdirAll(edir, 0755)
	entries := hexaboltdb.NewEntryStore()
	if err := entries.Open(edir); err != nil {
		return err
	}

	edir = filepath.Join(phi.conf.DataDir, "log", "index")
	os.MkdirAll(edir, 0755)
	index := hexaboltdb.NewIndexStore()
	if err := index.Open(edir); err != nil {
		return err
	}

	// Network transport
	hlnet := hexalog.NewNetTransport(30*time.Second, 300*time.Second)
	hexalog.RegisterHexalogRPCServer(phi.conf.GRPCServer, hlnet)

	stable := &hexalog.InMemStableStore{}

	fsm.RegisterDHT(phi.dht)

	c := phi.conf.Hexalog

	hexlog, err := hexalog.NewHexalog(c, fsm, entries, index, stable, hlnet)
	if err != nil {
		return err
	}

	phi.hexalog = hexlog

	trans := &localHexalogTransport{
		host:   c.AdvertiseHost,
		hexlog: hexlog,
		remote: hlnet,
	}

	phi.wal = NewHexalog(trans, c.Votes, c.Hasher)
	phi.wal.RegisterJury(NewSimpleJury(phi.dht))

	return nil
}

func (phi *Phi) startGrpc() error {
	ln, err := net.Listen("tcp", phi.conf.Hexalog.AdvertiseHost)
	if err != nil {
		return err
	}

	go func() {
		if er := phi.conf.GRPCServer.Serve(ln); er != nil {
			log.Fatal(er)
		}
	}()

	log.Println("[INFO] Fidias started:", ln.Addr().String())
	return nil
}

// LocalNode returns the local node from the dht.  This will be different from
// the internal local which is cached
func (phi *Phi) LocalNode() hexatype.Node {
	return phi.dht.LocalNode()
}

// DHT returns a distributed hash table interface
func (phi *Phi) DHT() DHT {
	return phi.dht
}

// BlockDevice returns a cluster aware block device
func (phi *Phi) BlockDevice() *BlockDevice {
	return phi.dev
}

// WAL returns the write-ahead-log for consistent operations
func (phi *Phi) WAL() WAL {
	return phi.wal
}

// Join joins the gossip networking using an existing node
func (phi *Phi) Join(existing []string) error {
	if phi.hexalog == nil {
		return fmt.Errorf("hexalog not initialized")
	}

	n, err := phi.memberlist.Join(existing)
	if err == nil {
		log.Println("[INFO] Joined peers:", n)
	}

	return err
}

// Shutdown performs a complete shutdown of all components
func (phi *Phi) Shutdown() error {
	return fmt.Errorf("TBI")
}
