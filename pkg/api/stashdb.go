package api

import (
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/xbapps/xbvr/pkg/externalreference"
	"github.com/xbapps/xbvr/pkg/scrape"
)

func (i ExternalReference) refreshStashPerformer(req *restful.Request, resp *restful.Response) {
	performerId := req.PathParameter("performerid")
	scrape.RefreshPerformer(performerId)
	resp.WriteHeader(http.StatusOK)
}

func (i ExternalReference) stashSceneApplyRules(req *restful.Request, resp *restful.Response) {
	go externalreference.ApplySceneRules()
}

func (i ExternalReference) matchAkaPerformers(req *restful.Request, resp *restful.Response) {
	go externalreference.MatchAkaPerformers()

}
func (i ExternalReference) stashDbUpdateImages(req *restful.Request, resp *restful.Response) {
	go externalreference.UpdateAllPerformerImages()

}
func (i ExternalReference) stashRunAll(req *restful.Request, resp *restful.Response) {
	go func() {
		scrape.StashDb()

		externalreference.ApplySceneRules()
		externalreference.MatchAkaPerformers()
		externalreference.UpdateAllPerformerImages()
	}()
}
