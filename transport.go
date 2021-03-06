package phi

import (
	"context"
	"time"

	"github.com/hexablock/hexalog"
)

type localHexalogTransport struct {
	host string

	// Hexalog
	hexlog *hexalog.Hexalog

	// Network transport
	remote hexalog.Transport
}

func newLocalHexalogTransport(host string, remote hexalog.Transport) *localHexalogTransport {
	return &localHexalogTransport{
		host:   host,
		remote: remote,
	}
}

func (trans *localHexalogTransport) NewEntry(host string, key []byte, opt *hexalog.RequestOptions) (*hexalog.Entry, error) {
	if trans.host == host {
		return trans.hexlog.New(key), nil
	}
	return trans.remote.NewEntry(host, key, opt)
}

func (trans *localHexalogTransport) ProposeEntry(ctx context.Context, host string, entry *hexalog.Entry, opts *hexalog.RequestOptions) (*hexalog.ReqResp, error) {
	// Remote host
	if trans.host != host {
		return trans.remote.ProposeEntry(ctx, host, entry, opts)
	}

	// Local
	resp := &hexalog.ReqResp{}
	ballot, err := trans.hexlog.Propose(entry, opts)
	if err != nil {
		return resp, err
	}

	if !opts.WaitBallot {
		return resp, nil
	}
	if err = ballot.Wait(); err != nil {
		return resp, err
	}
	resp.BallotTime = ballot.Runtime().Nanoseconds()

	if opts.WaitApply {
		fut := ballot.Future()
		_, err = fut.Wait(time.Duration(opts.WaitApplyTimeout) * time.Millisecond)
		resp.ApplyTime = fut.Runtime().Nanoseconds()
	}

	return resp, err
}

// GetEntry gets a local or remote entry based on host
func (trans *localHexalogTransport) GetEntry(host string, key, id []byte, opt *hexalog.RequestOptions) (*hexalog.Entry, error) {
	if trans.host == host {
		return trans.hexlog.Get(key, id)
	}
	return trans.remote.GetEntry(host, key, id, opt)
}
