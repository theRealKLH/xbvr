package externalreference

import (
	"encoding/json"
	"math"
	"net/url"
	"regexp"
	"strings"

	"github.com/xbapps/xbvr/pkg/models"
)

func UpdateAllPerformerData() {
	log.Infof("Starting Updating Actor Images")
	db, _ := models.GetDB()
	defer db.Close()

	var performers []models.ExternalReference

	db.Preload("XbvrLinks").
		Joins("JOIN external_reference_links erl on erl.external_reference_id = external_references.id").
		Where("external_references.external_source = 'stashdb performer'").
		Find(&performers)
		// join actors test image url/arr =''

	for _, performer := range performers {
		var data models.StashPerformer
		json.Unmarshal([]byte(performer.ExternalData), &data)

		if len(data.Images) > 0 {
			for _, actorLink := range performer.XbvrLinks {
				var actor models.Actor
				db.Where(models.Actor{ID: actorLink.InternalDbId}).Find(&actor)
				if actor.ImageUrl == "" || actor.ImageArr == "" {
					UpdateXbvrActor(data, actor.ID)

				}

			}
		}
	}
	log.Infof("Updating Actor Images Completed")
}

// this applies rules for matching xbvr scenes to stashdb, it then check if any matched scenes can be used to match actors
func ApplySceneRules() {
	log.Infof("Starting Scene Rule Matching")

	matchOnSceneUrl()

	config := GetSiteUrlMatchingRules()
	for sitename, configSite := range config.Sites {
		if len(configSite.Rules) > 0 {
			if configSite.StashId == "" {
				var ext models.ExternalReference
				ext.FindExternalId("stashdb studio", sitename)
				configSite.StashId = ext.ExternalId
			}
			matchSceneOnRules(sitename)
		}
	}

	checkMatchedScenes()
	log.Infof("Scene Rule Matching Completed")
}

// if an unmatched scene has a trailing number try to match on the  xbvr scene_id for that studio
func matchOnSceneUrl() {

	db, _ := models.GetDB()
	defer db.Close()

	var stashScenes []models.ExternalReference
	var unmatchedXbvrScenes []models.Scene

	db.Joins("Left JOIN external_reference_links erl on erl.external_reference_id = external_references.id").
		Where("external_references.external_source = ? and erl.internal_db_id is null", "stashdb scene").
		Find(&stashScenes)

	db.Joins("left join external_reference_links erl on erl.internal_db_id = scenes.id and external_source='stashdb scene'").
		Where("erl.id is null").
		Find(&unmatchedXbvrScenes)

	for _, stashScene := range stashScenes {
		var scene models.StashScene
		json.Unmarshal([]byte(stashScene.ExternalData), &scene)
		var xbvrId uint
		var xbvrSceneId string

		// see if we can link to an xbvr scene based on the urls
		for _, url := range scene.URLs {
			if url.Type == "STUDIO" {
				var xbvrScene models.Scene
				for _, scene := range unmatchedXbvrScenes {
					sceneurl := removeQueryFromURL(scene.SceneURL)
					tmpurl := removeQueryFromURL(url.URL)
					sceneurl = simplifyUrl(sceneurl)
					tmpurl = simplifyUrl(tmpurl)
					if strings.EqualFold(sceneurl, tmpurl) {
						xbvrScene = scene
					}
				}
				if xbvrScene.ID != 0 {
					xbvrId = xbvrScene.ID
					xbvrSceneId = xbvrScene.SceneID
				}
			}
		}
		if xbvrId != 0 {
			var xbrLink []models.ExternalReferenceLink
			xbrLink = append(xbrLink, models.ExternalReferenceLink{InternalTable: "scenes", InternalDbId: xbvrId, InternalNameId: xbvrSceneId, ExternalSource: "stashdb scene", ExternalId: scene.ID, MatchType: 10})
			stashScene.XbvrLinks = xbrLink
			stashScene.AddUpdateWithId()
		}
	}
}
func removeQueryFromURL(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	parsedURL.RawQuery = ""
	lastSlashIndex := strings.LastIndex(parsedURL.Path, "/")
	if lastSlashIndex == -1 {
		// No forward slash found, return the original input
		return parsedURL.Path
	}
	cleanedURL := parsedURL.String()
	return cleanedURL
}
func simplifyUrl(url string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(url, "http://", ""), "https://", ""), "www.", ""), "/", ""), "-", ""), "_", "")
}

// if an unmatched scene has a trailing number try to match on the  xbvr scene_id for that studio
func matchSceneOnRules(sitename string) {

	db, _ := models.GetDB()
	defer db.Close()

	config := GetSiteUrlMatchingRules()
	if config.Sites[sitename].StashId == "" {
		var ext models.ExternalReference
		ext.FindExternalId("stashdb studios", sitename)
		site := config.Sites[sitename]
		site.StashId = ext.ExternalId
		config.Sites[sitename] = site
	}

	log.Infof("Matching on rules for %s Stashdb Id: %s", sitename, config.Sites[sitename].StashId)
	var stashScenes []models.ExternalReference
	stashId := config.Sites[sitename].StashId
	if stashId == "" {
		return
	}

	var xbrScenes []models.Scene
	db.Preload("Cast").Where("scraper_id = ?", sitename).Find(&xbrScenes)

	db.Joins("Left JOIN external_reference_links erl on erl.external_reference_id = external_references.id").
		Where("external_references.external_source = ? and erl.internal_db_id is null and external_data like ?", "stashdb scene", "%"+stashId+"%").
		Find(&stashScenes)

	for _, stashScene := range stashScenes {
		var data models.StashScene
		json.Unmarshal([]byte(stashScene.ExternalData), &data)
	urlLoop:
		for _, url := range data.URLs {
			if url.Type == "STUDIO" {
				for _, rule := range config.Sites[sitename].Rules { // for each rule on this site
					re := regexp.MustCompile(rule.StashRule)
					match := re.FindStringSubmatch(url.URL)
					if match != nil {
						var extrefSite models.ExternalReference
						db.Where("external_source = ? and external_id = ?", "stashdb studio", data.Studio.ID).Find(&extrefSite)
						if extrefSite.ID != 0 {
							var xbvrScene models.Scene
							switch rule.XbvrField {
							case "scene_id":
								for _, scene := range xbrScenes {
									if strings.HasSuffix(scene.SceneID, match[rule.StashMatchResultPosition]) {
										xbvrScene = scene
										break
									}
								}
							case "scene_url":
								for _, scene := range xbrScenes {
									if strings.Contains(strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(scene.SceneURL), "-", " "), "_", " "), strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(match[rule.StashMatchResultPosition]), "-", " "), "_", " ")) {
										xbvrScene = scene
										break
									}
								}
							default:
								log.Errorf("Unkown xbvr field %s", rule.XbvrField)
							}

							if xbvrScene.ID != 0 {
								xbvrLink := models.ExternalReferenceLink{InternalTable: "scenes", InternalDbId: xbvrScene.ID, InternalNameId: xbvrScene.SceneID,
									ExternalReferenceID: stashScene.ID, ExternalSource: stashScene.ExternalSource, ExternalId: stashScene.ExternalId, MatchType: 20}
								stashScene.XbvrLinks = append(stashScene.XbvrLinks, xbvrLink)
								stashScene.Save()
								matchPerformerName(data, xbvrScene, 20)
								break urlLoop
							}
						}
					}
				}
			}
		}
	}

}

// checks if scenes that have a match, can match the scenes performers
func checkMatchedScenes() {
	db, _ := models.GetDB()
	defer db.Close()
	var stashScenes []models.ExternalReference
	db.Joins("JOIN external_reference_links erl on erl.external_reference_id = external_references.id").
		Preload("XbvrLinks").
		Where("external_references.external_source = ?", "stashdb scene").
		Find(&stashScenes)

	for _, extref := range stashScenes {
		var scene models.StashScene
		err := json.Unmarshal([]byte(extref.ExternalData), &scene)
		if err != nil {
			log.Infof("checkMatchedScenes %s %s %s", err, scene.ID, scene.Title)
		}
		var xbvrScene models.Scene

		for _, link := range extref.XbvrLinks {
			db.Where("id = ?", link.InternalDbId).Preload("Cast").Find(&xbvrScene)
			if xbvrScene.ID != 0 {

				for _, performer := range scene.Performers {
					var ref models.ExternalReference
					db.Preload("XbvrLinks").Where(&models.ExternalReference{ExternalSource: "stashdb performer", ExternalId: performer.Performer.ID}).Find(&ref)
					if ref.ID == 0 {
						continue
					}
					var fullPerformer models.StashPerformer
					err := json.Unmarshal([]byte(ref.ExternalData), &fullPerformer)
					if err != nil {
						log.Infof("checkMatchedScenes %s %s %s", err, fullPerformer.ID, fullPerformer.Name)
					}

					// if len(ref.XbvrLinks) == 0 {
					for _, xbvrActor := range xbvrScene.Cast {
						if strings.EqualFold(strings.TrimSpace(xbvrActor.Name), strings.TrimSpace(performer.Performer.Name)) {
							// check if actor already matched
							exists := false
							for _, link := range ref.XbvrLinks {
								if link.InternalDbId == xbvrActor.ID {
									exists = true
								}
							}
							if !exists {
								xbrLink := models.ExternalReferenceLink{InternalTable: "actors", InternalDbId: xbvrActor.ID, InternalNameId: xbvrActor.Name,
									ExternalReferenceID: ref.ID, ExternalSource: ref.ExternalSource, ExternalId: ref.ExternalId, MatchType: link.MatchType}
								ref.XbvrLinks = append(ref.XbvrLinks, xbrLink)
								ref.AddUpdateWithId()
								UpdateXbvrActor(fullPerformer, xbvrActor.ID)
							}
						}
					}
				}
			}
		}
	}
}

// updates an xbvr actor with data from a match stashdb actor
func UpdateXbvrActor(performer models.StashPerformer, xbvrActorID uint) {
	db, _ := models.GetDB()
	defer db.Close()

	changed := false
	actor := models.Actor{ID: xbvrActorID}
	db.Where(&actor).First(&actor)

	if len(performer.Images) > 0 {
		if actor.ImageUrl != performer.Images[0].URL && !actor.CheckForSetImage() {
			changed = true
			actor.ImageUrl = performer.Images[0].URL
		}
	}
	for _, alias := range performer.Aliases {
		changed = actor.AddToAliases(alias) || changed
	}
	if !strings.EqualFold(actor.Name, performer.Name) {
		changed = actor.AddToAliases(performer.Name) || changed
	}

	changed = CheckAndSetStringActorField(&actor.Gender, "gender", performer.Gender, actor.ID) || changed
	changed = CheckAndSetDateActorField(&actor.BirthDate, "birth_date", performer.BirthDate, actor.ID) || changed
	changed = CheckAndSetStringActorField(&actor.Nationality, "nationality", performer.Country, actor.ID) || changed
	changed = CheckAndSetStringActorField(&actor.Ethnicity, "ethnicity", performer.Ethnicity, actor.ID) || changed
	changed = CheckAndSetIntActorField(&actor.Height, "height", performer.Height, actor.ID) || changed
	changed = CheckAndSetStringActorField(&actor.EyeColor, "eye_color", performer.EyeColor, actor.ID) || changed
	changed = CheckAndSetStringActorField(&actor.HairColor, "eye_color", performer.HairColor, actor.ID) || changed
	changed = CheckAndSetStringActorField(&actor.CupSize, "cup_size", performer.CupSize, actor.ID) || changed
	changed = CheckAndSetIntActorField(&actor.BandSize, "band_size", int(math.Round(float64(performer.BandSize)*2.54)), actor.ID) || changed
	changed = CheckAndSetIntActorField(&actor.HipSize, "hip_size", int(math.Round(float64(performer.HipSize)*2.54)), actor.ID) || changed
	changed = CheckAndSetIntActorField(&actor.WaistSize, "waist_size", int(math.Round(float64(performer.WaistSize)*2.54)), actor.ID) || changed
	changed = CheckAndSetStringActorField(&actor.BreastType, "breast_type", performer.BreastType, actor.ID) || changed
	changed = CheckAndSetIntActorField(&actor.StartYear, "start_year", performer.CareerStartYear, actor.ID) || changed
	changed = CheckAndSetIntActorField(&actor.EndYear, "end_year", performer.CareerEndYear, actor.ID) || changed

	for _, tattoo := range performer.Tattoos {
		tattooString := convertBodyModToString(tattoo)
		if !actor.CheckForUserDeletes("tattoos", tattooString) {
			changed = actor.AddToTattoos(tattooString) || changed
		}
	}
	for _, piercing := range performer.Piercings {
		piercingString := convertBodyModToString(piercing)
		if !actor.CheckForUserDeletes("piercings", piercingString) {
			changed = actor.AddToPiercings(piercingString) || changed
		}
	}
	for _, img := range performer.Images {
		if !actor.CheckForUserDeletes("image_arr", img.URL) {
			changed = actor.AddToImageArray(img.URL) || changed
		}
	}
	for _, url := range performer.URLs {
		if !actor.CheckForUserDeletes("urls", url.URL) {
			changed = actor.AddToActorUrlArray(models.ActorLink{Url: url.URL, Type: ""}) || changed
		}
	}
	if changed {
		actor.Save()
	}
}

func addToArray(existingArray string, newValue string) string {
	values := []string{}
	if existingArray != "" {
		err := json.Unmarshal([]byte(existingArray), &values)
		if err != nil {
			log.Errorf("Could not extract array %s", values)
		}
	}
	for _, existingValue := range values {
		if existingValue == newValue {
			return existingArray
		}
	}
	values = append(values, newValue)
	jsonBytes, _ := json.Marshal(values)
	return string(jsonBytes)

}

func convertBodyModToString(bodyMod models.StashBodyModification) string {

	newMod := ""
	if bodyMod.Location != "" {
		newMod = bodyMod.Location
	}
	if bodyMod.Description != "" {
		if newMod != "" {
			newMod += " "
		}
		newMod += bodyMod.Description
	}
	return newMod
}

func matchPerformerName(scene models.StashScene, xbvrScene models.Scene, matchLevl int) {
	db, _ := models.GetDB()
	defer db.Close()

	for _, performer := range scene.Performers {
		var ref models.ExternalReference
		db.Preload("XbvrLinks").Where(&models.ExternalReference{ExternalSource: "stashdb performer", ExternalId: performer.Performer.ID}).Find(&ref)

		if ref.ID != 0 && len(ref.XbvrLinks) == 0 {
			for _, xbvrActor := range xbvrScene.Cast {
				if strings.EqualFold(strings.TrimSpace(xbvrActor.Name), strings.TrimSpace(performer.Performer.Name)) {
					xbvrLink := models.ExternalReferenceLink{InternalTable: "actors", InternalDbId: xbvrActor.ID, InternalNameId: xbvrActor.Name, MatchType: matchLevl,
						ExternalReferenceID: ref.ID, ExternalSource: ref.ExternalSource, ExternalId: ref.ExternalId}
					ref.XbvrLinks = append(ref.XbvrLinks, xbvrLink)
					ref.AddUpdateWithId()

					actor := models.Actor{ID: xbvrActor.ID}
					db.Where(&actor).First(&actor)
					if actor.ImageUrl == "" {
						var data models.StashPerformer
						json.Unmarshal([]byte(ref.ExternalData), &data)
						if len(data.Images) > 0 {
							actor.ImageUrl = data.Images[0].URL
							actor.Save()
						}
					}
				}
			}
		}
	}

}

// tries to match from stash to xbvr using the aka or aliases from stash
func MatchAkaPerformers() {
	log.Info("Starting Match on Actor Aka/Aliases")
	db, _ := models.GetDB()
	defer db.Close()

	type AkaList struct {
		ActorId           string
		AkaName           string
		SceneInternalDbId int
		Aliases           string
	}
	var akaList []AkaList

	var sqlcmd string

	// find performers, that are unmatched, get their scenes, cross join with their aliases
	switch db.Dialect().GetName() {
	case "mysql":
		sqlcmd = `
		select trim('"' from json_extract(value, '$.Performer.id')) as actor_id, trim('"' from json_extract(value, '$.As')) as aka_name, erl_s.internal_db_id scene_internal_db_id, json_extract(value, '$.Performer.aliases') as aliases
		FROM external_references er_p
		left join external_reference_links erl_p on erl_p.external_reference_id = er_p.id
		JOIN external_references er_s on er_s.external_data like CONCAT('%', er_p.external_id, '%') 
		join external_reference_links erl_s on erl_s.external_reference_id = er_s.id
		JOIN JSON_TABLE(er_s.external_data , '$.performers[*]' COLUMNS(value JSON PATH '$' )) u
		where er_p.external_source ='stashdb performer' and erl_p.internal_db_id is null
		`
	case "sqlite3":
		sqlcmd = `
		select json_extract(value, '$.Performer.id') as actor_id, json_extract(value, '$.As') as aka_name, erl_s.internal_db_id scene_internal_db_id,  json_extract(value, '$.Performer.aliases') as aliases
		from external_references er_p  
		left join external_reference_links erl_p on erl_p.external_reference_id = er_p.id
		join external_references er_s on er_s.external_data like '%' || er_p.external_id || '%'
		join external_reference_links erl_s on erl_s.external_reference_id = er_s.id
		Cross Join json_each(json_extract(er_s.external_data, '$.performers')) j
		where er_p.external_source ='stashdb performer' and erl_p.internal_db_id is null
		`
	}
	db.Raw(sqlcmd).Scan(&akaList)

	for _, aka := range akaList {
		var scene models.Scene
		scene.GetIfExistByPK(uint(aka.SceneInternalDbId))
		for _, actor := range scene.Cast {
			var extref models.ExternalReference
			if strings.EqualFold(strings.TrimSpace(actor.Name), strings.TrimSpace(aka.AkaName)) {
				extref.FindExternalId("stashdb performer", aka.ActorId)
				if extref.ID != 0 && len(extref.XbvrLinks) == 0 {
					xbvrLink := models.ExternalReferenceLink{InternalTable: "actors", InternalDbId: actor.ID, InternalNameId: actor.Name, MatchType: 30,
						ExternalReferenceID: extref.ID, ExternalSource: extref.ExternalSource, ExternalId: extref.ExternalId}
					extref.XbvrLinks = append(extref.XbvrLinks, xbvrLink)
					extref.Save()
					var data models.StashPerformer
					json.Unmarshal([]byte(extref.ExternalData), &data)
					UpdateXbvrActor(data, actor.ID)
				}
			}
			if len(extref.XbvrLinks) == 0 {
				var aliases []string
				json.Unmarshal([]byte(aka.Aliases), &aliases)
				for _, alias := range aliases {
					if len(extref.XbvrLinks) == 0 && strings.EqualFold(strings.TrimSpace(actor.Name), strings.TrimSpace(alias)) {
						extref.FindExternalId("stashdb performer", aka.ActorId)
						if extref.ID != 0 && len(extref.XbvrLinks) == 0 {
							xbvrLink := models.ExternalReferenceLink{InternalTable: "actors", InternalDbId: actor.ID, InternalNameId: actor.Name, MatchType: 30,
								ExternalReferenceID: extref.ID, ExternalSource: extref.ExternalSource, ExternalId: extref.ExternalId}
							extref.XbvrLinks = append(extref.XbvrLinks, xbvrLink)
							extref.Save()
							var data models.StashPerformer
							json.Unmarshal([]byte(extref.ExternalData), &data)
							UpdateXbvrActor(data, actor.ID)

						}
					}
				}
			}
		}
	}
	ReverseMatch()
	LinkOnXbvrAkaGroups()
	// reapply edits in case manual change if match_cycle
	log.Info("Match on Actor Aka/Aliases completed")
}

// we match from an xbvr back to stash for cases where the Stash actor name or aka used is different to the xbvr actor name
// if the scene was matched, then we can check the stash actors aliases for a match
func ReverseMatch() {

	log.Infof("Starting Reverse actor match from XBVR to Stashdb ")
	db, _ := models.GetDB()
	defer db.Close()
	var unmatchedActors []models.Actor
	var externalScenes []models.ExternalReference

	// get a list of unmatch xbvr actors
	db.Table("actors").Joins("LEFT JOIN external_reference_links erl on erl.internal_db_id =actors.id and erl.external_source ='stashdb performer'").Where("erl.internal_db_id is null").Find(&unmatchedActors)

	for _, actor := range unmatchedActors {
		// find scenes for the actor that have been matched
		db.Table("scene_cast").
			Joins("JOIN external_reference_links erl on erl.internal_db_id = scene_cast.scene_id and erl.external_source = 'stashdb scene'").
			Joins("JOIN external_references er on er.id =erl.external_reference_id").
			Select("er.*").
			Where("actor_id = ?", actor.ID).
			Find(&externalScenes)
	sceneLoop:
		for _, stashScene := range externalScenes {
			var stashSceneData models.StashScene
			json.Unmarshal([]byte(stashScene.ExternalData), &stashSceneData)
			for _, performance := range stashSceneData.Performers {
				if strings.EqualFold(strings.TrimSpace(actor.Name), strings.TrimSpace(performance.As)) {
					var extref models.ExternalReference
					extref.FindExternalId("stashdb performer", performance.Performer.ID)
					if extref.ID != 0 {
						xbvrLink := models.ExternalReferenceLink{InternalTable: "actors", InternalDbId: actor.ID, InternalNameId: actor.Name, MatchType: 40,
							ExternalReferenceID: extref.ID, ExternalSource: extref.ExternalSource, ExternalId: extref.ExternalId}
						extref.XbvrLinks = append(extref.XbvrLinks, xbvrLink)
						extref.Save()
					} else {
						log.Info("match no actor")
					}
					break sceneLoop
				}
				for _, alias := range performance.Performer.Aliases {
					if strings.EqualFold(strings.TrimSpace(actor.Name), strings.TrimSpace(alias)) {
						var extref models.ExternalReference
						extref.FindExternalId("stashdb performer", performance.Performer.ID)
						if extref.ID != 0 {
							xbvrLink := models.ExternalReferenceLink{InternalTable: "actors", InternalDbId: actor.ID, InternalNameId: actor.Name, MatchType: 40,
								ExternalReferenceID: extref.ID, ExternalSource: extref.ExternalSource, ExternalId: extref.ExternalId}
							extref.XbvrLinks = append(extref.XbvrLinks, xbvrLink)
							extref.Save()
						} else {
							UpdateXbvrActor(performance.Performer, actor.ID)
							log.Info("match")
						}
						break sceneLoop
					}
				}
			}

		}
	}
	log.Info("Reverse actor match from XBVR to Stashdb completed")
}

type ExtRefConfig struct {
	Sites map[string]ExtDbSiteConfig
}
type ExtDbSiteConfig struct {
	StashId     string
	ParentId    string
	TagIdFilter string
	Rules       []MatchRule
}
type MatchRule struct {
	XbvrMatchType            MatchType
	XbvrField                string
	XbvrMatch                string
	XbvrMatchResultPosition  int
	StashMatchType           MatchType
	StashField               string
	StashRule                string
	StashMatchResultPosition int
}
type MatchType string

const (
	RegexMatch MatchType = "regex_match"
	RegexGroup MatchType = "regex_group"
)

func GetSiteUrlMatchingRules() ExtRefConfig {
	db, _ := models.GetDB()
	defer db.Close()

	var config ExtRefConfig
	var kv models.KV

	db.Where(models.KV{Key: "stashdb"}).First(&kv)
	if kv.Value == "" {
		config = initalizeConfig()
	} else {
		json.Unmarshal([]byte(kv.Value), &config)
	}
	return config
}

func initalizeConfig() ExtRefConfig {
	db, _ := models.GetDB()
	defer db.Close()

	var config ExtRefConfig
	var sites []models.Site

	config.Sites = make(map[string]ExtDbSiteConfig)
	config.Sites["allvrporn-vrporn"] = ExtDbSiteConfig{StashId: "44fd483b-85eb-4b22-b7f2-c92c1a50923a"}
	config.Sites["bvr"] = ExtDbSiteConfig{StashId: "1ffbd972-7d69-4ccb-b7da-c6342a9c3d70"}
	config.Sites["cuties-vr"] = ExtDbSiteConfig{StashId: "1e5240a8-29b3-41ed-ae28-fc9231eac449"}
	config.Sites["czechvrintimacy"] = ExtDbSiteConfig{StashId: "ddff31bc-e9d0-475e-9c5b-1cc151eda27b"}
	config.Sites["darkroomvr"] = ExtDbSiteConfig{StashId: "e57f0b82-a8d0-4904-a611-71e95f9b9248"}
	config.Sites["ellielouisevr"] = ExtDbSiteConfig{StashId: "47764349-fb49-42b9-8445-7fa4fb13f9e1"}
	config.Sites["emilybloom"] = ExtDbSiteConfig{StashId: "b359a2fe-dcf0-46e2-8ace-a684df52573e"}
	config.Sites["herpovr"] = ExtDbSiteConfig{StashId: "7d94a83d-2b0b-4076-9e4c-fd9dc6222b8a"}
	config.Sites["jimmydraws"] = ExtDbSiteConfig{StashId: "bf7b7b9a-b96a-401d-8412-ec3f52bcfb6c"}
	config.Sites["kinkygirlsberlin"] = ExtDbSiteConfig{StashId: "7d892a03-dfbe-4476-917d-4940be13fb24"}
	config.Sites["lethalhardcorevr"] = ExtDbSiteConfig{StashId: "3a9883f6-9642-4be1-9a65-d8d13eadbdf0"}
	config.Sites["lustreality"] = ExtDbSiteConfig{StashId: "f31021ba-f4c3-46eb-89c5-b114478d88d2"}
	config.Sites["mongercash"] = ExtDbSiteConfig{StashId: "96ee2435-0b0f-4fb4-8b53-8c929aa493bd"}
	config.Sites["only3xvr"] = ExtDbSiteConfig{StashId: "57391302-bac4-4f15-a64d-7cd9a9c152e0"}
	config.Sites["povcentralvr"] = ExtDbSiteConfig{StashId: "57391302-bac4-4f15-a64d-7cd9a9c152e0"}
	config.Sites["realhotvr"] = ExtDbSiteConfig{StashId: "cf3510db-5fe5-4212-b5da-da27b5352d1c"}
	config.Sites["realitylovers"] = ExtDbSiteConfig{StashId: "3463e72d-6af3-497f-b841-9119065d2916"}
	config.Sites["sinsvr"] = ExtDbSiteConfig{StashId: "805820d0-8fb2-4b04-8c0c-6e392842131b"}
	config.Sites["squeeze-vr"] = ExtDbSiteConfig{StashId: "b2d048da-9180-4e43-b41a-bdb4d265c8ec"}
	config.Sites["swallowbay"] = ExtDbSiteConfig{StashId: "17ff0143-3961-4d38-a80a-fe72407a274d"}
	config.Sites["tonightsgirlfriend"] = ExtDbSiteConfig{StashId: "69a66a95-15de-4b0a-9537-7f15b358392f"}
	config.Sites["virtualrealamateur"] = ExtDbSiteConfig{StashId: "cac0470b-7802-4946-b5ef-e101e166cdaf"}
	config.Sites["virtualtaboo"] = ExtDbSiteConfig{StashId: "1e6defb1-d3a4-4f0c-8616-acd5c343ca2b"}
	config.Sites["virtualxporn"] = ExtDbSiteConfig{StashId: "d55815ac-955f-45a0-a0fa-f6ad335e212d"}
	config.Sites["vrallure"] = ExtDbSiteConfig{StashId: "bb904923-c028-46b7-b269-49dfa54b5332"}
	config.Sites["vrbangers"] = ExtDbSiteConfig{StashId: "f8a826f6-89c2-4db0-a899-1229d11865b3"}
	config.Sites["vrconk"] = ExtDbSiteConfig{StashId: "b038d55c-1e94-41ff-938a-e6aafb0b1759"}
	config.Sites["vrmansion-slr"] = ExtDbSiteConfig{StashId: "a01012bc-42e9-4372-9c25-58f0f94e316b"}
	config.Sites["vrsexygirlz"] = ExtDbSiteConfig{StashId: "b346fe21-5d12-407f-9f50-837f067956d7"}
	config.Sites["vrsolos"] = ExtDbSiteConfig{StashId: "b2d048da-9180-4e43-b41a-bdb4d265c8ec"}
	config.Sites["wankitnowvr"] = ExtDbSiteConfig{StashId: "acb1ed8f-4967-4c5a-b16a-7025bdeb75c5"}

	config.Sites["wetvr"] = ExtDbSiteConfig{StashId: "981887d6-da48-4dfc-88d1-7ed13a2754f2"}
	//config.Sites["czechvr"] = ExtDbSiteConfig{Rules: []MatchRule{{XbvrField: "scene_id", XbvrMatch: `-\d+$`, XbvrMatchResultPosition: 0, StashField: "", StashRule:`xss[^0-9]*(\d+)$`  }}}

	config.Sites["wankzvr"] = ExtDbSiteConfig{StashId: "b04bca51-15ea-45ab-80f6-7b002fd4a02d",
		Rules: []MatchRule{{XbvrMatchType: RegexMatch, XbvrField: "scene_id", XbvrMatch: `-\d+$`, XbvrMatchResultPosition: 0, StashMatchType: RegexGroup, StashField: "", StashRule: `(povr|wankzvr).com\/(.*)(-\d*?)\/?$`, StashMatchResultPosition: 3}}}
	config.Sites["naughtyamericavr"] = ExtDbSiteConfig{StashId: "049c167b-0cf3-4965-aae5-f5150122a928", ParentId: "2be8463b-0505-479e-a07d-5abc7a6edd54", TagIdFilter: "6458e5cf-4f65-400b-9067-582141e2a329",
		Rules: []MatchRule{{XbvrMatchType: RegexMatch, XbvrField: "scene_id", XbvrMatch: `-\d+$`, XbvrMatchResultPosition: 0, StashMatchType: RegexGroup, StashField: "", StashRule: `(naughtyamerica).com\/(.*)(-\d*?)\/?$`, StashMatchResultPosition: 3}}}
	config.Sites["povr-originals"] = ExtDbSiteConfig{StashId: "b95c0ee4-2e95-46cf-aa67-45c82bdcd5fc",
		Rules: []MatchRule{{XbvrMatchType: RegexMatch, XbvrField: "scene_id", XbvrMatch: `-\d+$`, XbvrMatchResultPosition: 0, StashMatchType: RegexGroup, StashField: "", StashRule: `(povr|wankzvr).com\/(.*)(-\d*?)\/?$`, StashMatchResultPosition: 3}}}
	config.Sites["brasilvr"] = ExtDbSiteConfig{StashId: "511e41c8-5063-48b8-a8d9-4e18852da338",
		Rules: []MatchRule{{XbvrMatchType: RegexMatch, XbvrField: "scene_id", XbvrMatch: `-\d+$`, XbvrMatchResultPosition: 0, StashMatchType: RegexGroup, StashField: "", StashRule: `(brasilvr|povr|wankzvr).com\/(.*)(-\d*?)\/?$`, StashMatchResultPosition: 3}}}
	config.Sites["milfvr"] = ExtDbSiteConfig{StashId: "38382977-9f5e-42fb-875b-2f4dd1272b11",
		Rules: []MatchRule{{XbvrMatchType: RegexMatch, XbvrField: "scene_id", XbvrMatch: `-\d+$`, XbvrMatchResultPosition: 0, StashMatchType: RegexGroup, StashField: "", StashRule: `(milfvr|povr|wankzvr).com\/(.*)(-\d*?)\/?$`, StashMatchResultPosition: 3}}}

	config.Sites["czechvr"] = ExtDbSiteConfig{StashId: "a9ed3948-5263-46f6-a3f0-e0dfc059ee73",
		Rules: []MatchRule{{XbvrMatchType: RegexMatch, XbvrField: "scene_url", XbvrMatch: `(czechvrnetwork|czechvr|czechvrcasting|czechvrfetish|vrintimacy).com\/([^\/]+)\/?$`, XbvrMatchResultPosition: 2, StashMatchType: RegexGroup, StashField: "", StashRule: `(czechvrnetwork|czechvr|czechvrcasting|czechvrfetish|vrintimacy).com\/([^\/]+)\/?$`, StashMatchResultPosition: 2}}}
	config.Sites["czechvrcasting"] = ExtDbSiteConfig{StashId: "2fa76fba-ccd7-457d-bc7c-ebc1b09e580b",
		Rules: []MatchRule{{XbvrMatchType: RegexMatch, XbvrField: "scene_url", XbvrMatch: `(czechvrnetwork|czechvr|czechvrcasting|czechvrfetish|vrintimacy).com\/([^\/]+)\/?$`, XbvrMatchResultPosition: 2, StashMatchType: RegexGroup, StashField: "", StashRule: `(czechvrnetwork|czechvr|czechvrcasting|czechvrfetish|vrintimacy).com\/([^\/]+)\/?$`, StashMatchResultPosition: 2}}}
	config.Sites["czechvrfetish"] = ExtDbSiteConfig{StashId: "19399096-7b83-4404-b960-f8f8c641a93e",
		Rules: []MatchRule{{XbvrMatchType: RegexMatch, XbvrField: "scene_url", XbvrMatch: `(czechvrnetwork|czechvr|czechvrcasting|czechvrfetish|vrintimacy).com\/([^\/]+)\/?$`, XbvrMatchResultPosition: 2, StashMatchType: RegexGroup, StashField: "", StashRule: `(czechvrnetwork|czechvr|czechvrcasting|czechvrfetish|vrintimacy).com\/([^\/]+)\/?$`, StashMatchResultPosition: 2}}}
	config.Sites["czechvrintimacy"] = ExtDbSiteConfig{StashId: "ddff31bc-e9d0-475e-9c5b-1cc151eda27b",
		Rules: []MatchRule{{XbvrMatchType: RegexMatch, XbvrField: "scene_url", XbvrMatch: `(czechvrnetwork|czechvr|czechvrcasting|czechvrfetish|vrintimacy).com\/([^\/]+)\/?$`, XbvrMatchResultPosition: 2, StashMatchType: RegexGroup, StashField: "", StashRule: `(czechvrnetwork|czechvr|czechvrcasting|czechvrfetish|vrintimacy).com\/([^\/]+)\/?$`, StashMatchResultPosition: 2}}}
	config.Sites["tmwvrnet"] = ExtDbSiteConfig{StashId: "fd1a7f1d-9cc3-4d30-be0d-1c05b2a8b9c3",
		Rules: []MatchRule{{XbvrMatchType: RegexMatch, XbvrField: "scene_url", XbvrMatch: `(teenmegaworld.net|tmwvrnet.com)(\/trailers)?\/([^\/]+)\/?$`, XbvrMatchResultPosition: 3, StashMatchType: RegexGroup, StashField: "", StashRule: `(teenmegaworld.net|tmwvrnet.com)(\/trailers)?\/([^\/]+)\/?$`, StashMatchResultPosition: 3}}}
	config.Sites["virtualrealporn"] = ExtDbSiteConfig{StashId: "191ba106-00d3-4f01-8c57-0cf0e88a2a50",
		Rules: []MatchRule{{XbvrMatchType: RegexMatch, XbvrField: "scene_url", XbvrMatch: `virtualrealporn`, XbvrMatchResultPosition: 3, StashMatchType: RegexGroup, StashField: "", StashRule: `(\/[^\/]+)\/?$`, StashMatchResultPosition: 1},
			{XbvrMatchType: RegexMatch, XbvrField: "scene_url", XbvrMatch: `virtualrealporn`, XbvrMatchResultPosition: 3, StashMatchType: RegexGroup, StashField: "", StashRule: `(\/[^\/]+)(-\d{3,10}?)\/?$`, StashMatchResultPosition: 1}}}
	config.Sites["realjamvr"] = ExtDbSiteConfig{StashId: "2059fbf9-94fe-4986-8565-2a7cc199636a",
		Rules: []MatchRule{{XbvrMatchType: RegexMatch, XbvrField: "scene_url", XbvrMatch: `(realjamvr.com)(.*)\/(\d*-?)([^\/]+)\/?$`, XbvrMatchResultPosition: 4, StashMatchType: RegexGroup, StashField: "", StashRule: `(realjamvr.com)(.*)\/(\d*-?)([^\/]+)\/?$`, StashMatchResultPosition: 4}}}
	config.Sites["sexbabesvr"] = ExtDbSiteConfig{StashId: "b80d419c-4a81-44c9-ae79-d9614dd30351",
		Rules: []MatchRule{{XbvrMatchType: RegexMatch, XbvrField: "scene_url", XbvrMatch: `(sexbabesvr.com)(.*)\/([^\/]+)\/?$`, XbvrMatchResultPosition: 3, StashMatchType: RegexGroup, StashField: "", StashRule: `(sexbabesvr.com)(.*)\/([^\/]+)\/?$`, StashMatchResultPosition: 3}}}
	config.Sites["lethalhardcorevr"] = ExtDbSiteConfig{StashId: "3a9883f6-9642-4be1-9a65-d8d13eadbdf0",
		Rules: []MatchRule{{XbvrMatchType: RegexMatch, XbvrField: "scene_url", XbvrMatch: `(lethalhardcorevr.com).*\/(\d{6,8})\/.*`, XbvrMatchResultPosition: 2, StashMatchType: RegexGroup, StashField: "", StashRule: `(lethalhardcorevr.com).*\/(\d{6,8})\/.*`, StashMatchResultPosition: 2}}}

	db.Where(&models.Site{IsEnabled: true}).Order("id").Find(&sites)
	for _, site := range sites {
		if strings.HasSuffix(site.Name, "SLR)") {
			siteConfig := config.Sites[site.ID]
			//siteConfig.StashId = studio.Data.Studio.ID
			siteConfig.Rules = append(siteConfig.Rules, MatchRule{XbvrMatchType: RegexMatch, XbvrField: "scene_id", XbvrMatch: `-\d+$`, XbvrMatchResultPosition: 0, StashMatchType: RegexGroup, StashField: "", StashRule: `(sexlikereal).com\/[^0-9]*(-\d*)`, StashMatchResultPosition: 2})
			config.Sites[site.ID] = siteConfig
		}
		if strings.HasSuffix(site.Name, "POVR)") {
			siteConfig := config.Sites[site.ID]
			//siteConfig.StashId = studio.Data.Studio.ID
			if len(siteConfig.Rules) == 0 {
				siteConfig.Rules = append(siteConfig.Rules, MatchRule{XbvrMatchType: RegexMatch, XbvrField: "scene_id", XbvrMatch: `-\d+$`, XbvrMatchResultPosition: 0, StashMatchType: RegexGroup, StashField: "", StashRule: `(povr|wankzvr).com\/(.*)(-\d*?)\/?$`, StashMatchResultPosition: 2})
				config.Sites[site.ID] = siteConfig
			}
		}
	}

	jsonData, _ := json.MarshalIndent(config, "", "  ")
	kvs := models.KV{Key: "stashdb", Value: string(jsonData)}
	kvs.Save()

	return config
}

// links an aka group Actor in xbvr to stashdb, based on any links to stashdb by actors in the group
// it then adds links for other actors in the group that don't have links
func LinkOnXbvrAkaGroups() {
	log.Infof("LinkActors based on XBR Aka Groups")
	db, _ := models.GetDB()
	defer db.Close()

	// Link Aka group actors
	var unlinkedAkaActors []models.Actor
	db.Where("name like 'aka:%' and IFNULL(image_url, '') = ''").Find(&unlinkedAkaActors)
	for _, akaActor := range unlinkedAkaActors {
		var akaGroup models.Aka
		db.Preload("Akas").
			Where("aka_actor_id = ?", akaActor.ID).
			First(&akaGroup)

		for _, actor := range akaGroup.Akas {
			var extref models.ExternalReference
			db.
				Table("external_reference_links").
				Joins("JOIN external_references on external_references.id = external_reference_links.external_reference_id").
				Preload("XbvrLinks").
				Where("internal_db_id = ? and external_reference_links.external_source='stashdb performer'", actor.ID).
				Select("external_references.*").
				First(&extref)
			if extref.ID != 0 {
				var data models.StashPerformer
				json.Unmarshal([]byte(extref.ExternalData), &data)
				xbrLink := models.ExternalReferenceLink{InternalTable: "actors", InternalDbId: akaActor.ID, InternalNameId: akaActor.Name,
					ExternalReferenceID: extref.ID, ExternalSource: extref.ExternalSource, ExternalId: extref.ExternalId, MatchType: 60}
				extref.XbvrLinks = append(extref.XbvrLinks, xbrLink)
				extref.Save()
				UpdateXbvrActor(data, akaActor.ID)
				break
			}
		}
	}

	// Link unlinked actors in aka group
	var akaGroup []models.Aka
	db.Preload("Akas").
		Joins("JOIN external_reference_links on external_reference_links.internal_db_id = akas.aka_actor_id and external_reference_links.external_source='stashdb performer'").
		Find(&akaGroup)
	for _, akaActor := range akaGroup {
		var akaActorRef models.ExternalReference
		db.Table("external_reference_links").
			Preload("XbvrLinks").
			Joins("JOIN external_references on external_references.id = external_reference_links.external_reference_id").
			Where("internal_db_id = ? and external_reference_links.external_source='stashdb performer'", akaActor.AkaActorId).
			Select("external_references.*").
			First(&akaActorRef)
		var akaActorStashPerformer models.StashPerformer
		json.Unmarshal([]byte(akaActorRef.ExternalData), &akaActorStashPerformer)

		for _, actor := range akaActor.Akas {
			var extref models.ExternalReference
			db.Table("external_reference_links").
				Joins("JOIN external_references on external_references.id = external_reference_links.external_reference_id").
				Where("internal_db_id = ? and external_reference_links.external_source='stashdb performer'", actor.ID).
				Select("external_references.*").
				First(&extref)
			if extref.ID == 0 {
				xbrLink := models.ExternalReferenceLink{InternalTable: "actors", InternalDbId: actor.ID, InternalNameId: actor.Name,
					ExternalReferenceID: akaActorRef.ID, ExternalSource: akaActorRef.ExternalSource, ExternalId: akaActorRef.ExternalId, MatchType: 70}
				akaActorRef.XbvrLinks = append(akaActorRef.XbvrLinks, xbrLink)
				akaActorRef.Save()
				UpdateXbvrActor(akaActorStashPerformer, actor.ID)
			}

		}
	}
}
