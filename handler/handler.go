package handler

import (
	"sync"

	"github.com/cycraig/scpbattle/store"
)

/*
Handler
Encapsulating class so we can access the SCP store on requests.
*/
type Handler struct {
	scpCache *store.SCPCache
	scpLock  map[uint]*sync.Mutex // lock per SCP to prevent lost votes
}

func NewHandler(scpCache *store.SCPCache) *Handler {
	return &Handler{
		scpCache: scpCache,
		scpLock:  make(map[uint]*sync.Mutex),
	}
}
