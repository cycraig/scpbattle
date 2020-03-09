package handler

import "github.com/cycraig/scpbattle/store"

/*
Encapsulating class so we can access the SCP store on requests.
*/
type Handler struct {
	scpCache *store.SCPCache
}

func NewHandler(scpCache *store.SCPCache) *Handler {
	return &Handler{
		scpCache: scpCache,
	}
}
