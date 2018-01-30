package rest

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
	"novaforge.bull.com/starlings-janus/janus/log"
)

func (s *Server) getResourcesProvidersUsageHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
	provName := params.ByName("providerName")
	log.Debugf("Get resources usage from provider name:%q", provName)

	resourcesProvider, err := reg.GetResourcesProvider(provName)
	if err != nil {
		log.Printf("[ERROR] No resources provider found with name:%q", provName)
		writeError(w, r, errNotFound)
		return
	}

	//FIXME register task and execute it here instead
	resourcesUsage, err := resourcesProvider.GetResourcesUsage()
	if err != nil {
		log.Panic(err)
	}
	encodeJSONResponse(w, r, resourcesUsage)
}
