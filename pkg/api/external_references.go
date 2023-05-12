package api

import (
	"github.com/emicklei/go-restful/v3"
)

//var RequestBody []byte

type ExternalReference struct{}

func (i ExternalReference) WebService() *restful.WebService {
	tags := []string{"Extref"}
	log.Info(tags)
	ws := new(restful.WebService)

	ws.Path("/api/extref").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	return ws
}
