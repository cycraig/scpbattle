package handler

import (
	"sync"

	"github.com/cycraig/scpbattle/store"
)

// Handler is a simple encapsulating class so http handlers can access the SCP database on requests.
type Handler struct {
	scpCache *store.SCPCache
	scpLock  map[uint]*sync.Mutex // lock per SCP to prevent lost votes
	imageDir string
}

// NewHandler instantiates a Handler with the given SCPCache.
// The imageDir field must end with a trailing slash, e.g. "images/".
func NewHandler(scpCache *store.SCPCache, imageDir string) *Handler {
	return &Handler{
		scpCache: scpCache,
		scpLock:  make(map[uint]*sync.Mutex),
		imageDir: imageDir,
	}
}
