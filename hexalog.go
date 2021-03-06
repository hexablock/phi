package phi

import (
	"context"
	"hash"
	"time"

	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexatype"
	"github.com/hexablock/log"
)

// WriteStats contains stats regarding a write operation to the log
type WriteStats struct {
	BallotTime   time.Duration
	ApplyTime    time.Duration
	Participants []*hexalog.Participant
}

// RetryOptions for WAL proposals
type RetryOptions struct {
	Retries       int
	RetryInterval time.Duration
}

// Jury implements an interface to get participants for an entry proposal. These
// are the peers participating in the voting process
type Jury interface {
	Participants(key []byte, min int) ([]*hexalog.Participant, error)
	RegisterDHT(dht DHT)
}

// Hexalog is a network aware Hexalog.  It implements selecting the
// participants from the network for consistency
type Hexalog struct {
	// hash function to use
	hashFunc func() hash.Hash

	// min votes for any log entry
	minVotes int

	// Hexalog
	trans WALTransport

	// Jury selector for voting rounds
	jury Jury
}

// DefaultRetryOptions returns a default set of RetryOptions
func DefaultRetryOptions() *RetryOptions {
	return &RetryOptions{
		Retries:       1,
		RetryInterval: 30 * time.Millisecond,
	}
}

// NewHexalog inita a new DHT aware WAL
func NewHexalog(trans WALTransport, minVotes int, hashFunc func() hash.Hash) *Hexalog {
	return &Hexalog{trans: trans, minVotes: minVotes, hashFunc: hashFunc}
}

// RegisterJury registers a jury interface used to get participants
func (hexlog *Hexalog) RegisterJury(jury Jury) {
	hexlog.jury = jury
}

// NewEntry returns a new Entry for the given key from Hexalog.  It returns an
// error if the node is not part of the location set or a lookup error occurs
func (hexlog *Hexalog) NewEntry(key []byte) (*hexalog.Entry, []*hexalog.Participant, error) {
	peers, err := hexlog.jury.Participants(key, hexlog.minVotes)
	if err != nil {
		return nil, nil, err
	}

	opt := &hexalog.RequestOptions{}
	var entry *hexalog.Entry

	for _, loc := range peers {
		if entry, err = hexlog.trans.NewEntry(loc.Host, key, opt); err == nil {
			return entry, peers, nil
		}
	}

	return nil, peers, err
}

// NewEntryFrom creates a new entry based on the given entry.  It uses the
// given height and previous hash of the entry to determine the values for
// the new entry.  This is essentially a compare and set
func (hexlog *Hexalog) NewEntryFrom(entry *hexalog.Entry) (*hexalog.Entry, []*hexalog.Participant, error) {
	peers, err := hexlog.jury.Participants(entry.Key, hexlog.minVotes)
	if err != nil {
		return nil, nil, err
	}

	nentry := &hexalog.Entry{
		Key:       entry.Key,
		Previous:  entry.Hash(hexlog.hashFunc()),
		Height:    entry.Height + 1,
		Timestamp: uint64(time.Now().UnixNano()),
	}

	return nentry, peers, nil
}

// GetEntry tries to get an entry from the network from all known locations
func (hexlog *Hexalog) GetEntry(key, id []byte) (*hexalog.Entry, error) {
	peers, err := hexlog.jury.Participants(key, hexlog.minVotes)
	if err != nil {
		return nil, err
	}

	opt := &hexalog.RequestOptions{}

	for _, p := range peers {
		ent, er := hexlog.trans.GetEntry(p.Host, key, id, opt)
		if er == nil {
			return ent, nil
		}
		err = er
	}

	return nil, err
}

// ProposeEntry finds locations for the entry and proposes it to those locations
// It retries the specified number of times before returning.  It returns a an
// entry id on success and error otherwise
func (hexlog *Hexalog) ProposeEntry(entry *hexalog.Entry, opts *hexalog.RequestOptions, retry *RetryOptions) (eid []byte, stats *WriteStats, err error) {
	if retry == nil {
		retry = DefaultRetryOptions()
	} else {
		if retry.Retries < 1 {
			retry.Retries = 1
		}

		if retry.RetryInterval == 0 {
			retry.RetryInterval = 30 * time.Millisecond
		}
	}

	ps := len(opts.PeerSet)

	for i := 0; i < retry.Retries; i++ {
		log.Printf("[DEBUG] Proposing key=%s participants=%d try=%d/%d", entry.Key, ps, i, retry.Retries)
		// Propose with retries.  Retry only on a ErrPreviousHash error
		var resp *hexalog.ReqResp
		if resp, err = hexlog.trans.ProposeEntry(context.Background(), opts.PeerSet[0].Host, entry, opts); err == nil {

			eid = entry.Hash(hexlog.hashFunc())
			stats = &WriteStats{
				BallotTime:   time.Duration(resp.BallotTime),
				ApplyTime:    time.Duration(resp.ApplyTime),
				Participants: opts.PeerSet,
			}
			return

		} else if err == hexatype.ErrPreviousHash {
			time.Sleep(retry.RetryInterval)
		} else {
			return
		}

	}

	return
}
