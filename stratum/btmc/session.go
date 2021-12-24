package btmc

import (
	"math/rand"

	"github.com/cornelk/hashmap"

	ss "github.com/bytom/btmcpool/stratum"
)

type btmcSessionData struct {
	nonce    uint32
	worker   *ss.Worker
	submitId *hashmap.HashMap
}

func (s *btmcSessionData) GetWorker() *ss.Worker {
	return s.worker
}

func (s *btmcSessionData) SetWorker(worker *ss.Worker) {
	s.worker = worker
}

func (s *btmcSessionData) getNonce() uint32 {
	return s.nonce
}

type btmcSessionDataBuilder struct {
	id              uint64
	maxSessions     int
	sessionIdOffset uint
}

func NewBtmcSessionDataBuilder(serverId uint64, maxSessions int) *btmcSessionDataBuilder {
	return &btmcSessionDataBuilder{
		id:          serverId,
		maxSessions: maxSessions,
	}
}

// Build builds a btmSession
func (b *btmcSessionDataBuilder) Build() ss.SessionData {
	return &btmcSessionData{
		nonce:    rand.Uint32(),
		worker:   nil,
		submitId: &hashmap.HashMap{},
	}
}
