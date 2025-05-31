package picker

import (
	"sync"

	"github.com/thomaswinkler/c8y-session-1password/pkg/core"
)

type randomItemGenerator struct {
	sessions []*core.CumulocitySession
	index    int
	mtx      *sync.Mutex
}

func (r *randomItemGenerator) Len() int {
	return len(r.sessions)
}

func (r *randomItemGenerator) reset() {
	r.mtx = &sync.Mutex{}
}

func (r *randomItemGenerator) Next() *core.CumulocitySession {
	if r.mtx == nil {
		r.reset()
	}

	r.mtx.Lock()
	defer r.mtx.Unlock()

	i := &core.CumulocitySession{
		SessionURI: r.sessions[r.index].SessionURI,
		Host:       r.sessions[r.index].Host,
		Tenant:     r.sessions[r.index].Tenant,
		Username:   r.sessions[r.index].Username,
		ItemID:     r.sessions[r.index].ItemID,
		ItemName:   r.sessions[r.index].ItemName,
		VaultID:    r.sessions[r.index].VaultID,
		VaultName:  r.sessions[r.index].VaultName,
		Tags:       r.sessions[r.index].Tags,
	}

	r.index++
	if r.index >= len(r.sessions) {
		r.index = 0
	}

	return i
}
