package api

import (
	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
)

//var RequestBody []byte

type ExternalReference struct{}

func (i ExternalReference) WebService() *restful.WebService {
	tags := []string{"Extref"}

	ws := new(restful.WebService)

	ws.Path("/api/extref").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/generic/scrape_all").To(i.genericActorScraper).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.POST("/generic/scrape_single").To(i.genericSingleActorScraper).
		Metadata(restfulspec.KeyOpenAPITags, tags))
	return ws
}
