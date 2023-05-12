package scrape

import (
	"encoding/json"
	"fmt"
	"html"
	"math"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/markphelps/optional"
	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/models"
)

func GenericActorScrapers() {
	log.Infof("Scraping Actors Started")
	db, _ := models.GetDB()
	defer db.Close()

	siteRules := BuildRules()

	maxConcurrent := 10                             // limit the number of tasks running at the same time
	semaphore := make(chan struct{}, maxConcurrent) // create a semaphore with capacity `maxConcurrent`
	var wg sync.WaitGroup

	var actors []models.Actor
	db.Preload("Scenes").
		Find(&actors)
	for _, actor := range actors {
		wg.Add(1)
		go func(actor models.Actor) {
			semaphore <- struct{}{} // acquire a semaphore
			var actorLink []models.ActorLink
			json.Unmarshal([]byte(actor.URLs), &actorLink)
			for _, link := range actorLink {
				for _, rule := range siteRules {
					if link.Type == rule.Source {
						var extRefLink models.ExternalReferenceLink
						db.Preload("ExternalReference").
							Where(&models.ExternalReferenceLink{ExternalSource: rule.Source, InternalDbId: actor.ID}).
							First(&extRefLink)
						if extRefLink.ID == 0 {
							applyRules(link.Url, link.Type, rule, &actor)
						} else {
							for _, scene := range actor.Scenes {
								if scene.ReleaseDate.After(extRefLink.ExternalReference.ExternalDate) {
									applyRules(link.Url, link.Type, rule, &actor)
									break
								}
							}

						}
					}
				}
			}
			<-semaphore // release the semaphore
			wg.Done()
		}(actor)
	}
	wg.Wait()
	log.Infof("Scraping Actors Completed")
}
func GenericSingleActorScraper(actorId uint, actorPage string) {
	log.Infof("Scraping Actor Details from %s", actorPage)
	db, _ := models.GetDB()
	defer db.Close()

	var actor models.Actor
	actor.ID = actorId
	db.Find(&actor)
	siteRules := BuildRules()

	var extRefLink models.ExternalReferenceLink
	db.Preload("ExternalReference").
		Where(&models.ExternalReferenceLink{ExternalId: actorPage, InternalDbId: actor.ID}).
		First(&extRefLink)

	for _, rule := range siteRules {
		if extRefLink.ExternalSource == rule.Source {
			applyRules(actorPage, rule.Source, rule, &actor)
		}
	}

	log.Infof("Scraping Actor Details from %s Completed", actorPage)
}

func applyRules(actorPage string, source string, rules GenericActorScraperRules, actor *models.Actor) {
	actorCollector := CreateCollector(rules.Domain)
	data := make(map[string]string)
	actorChanged := false
	actorCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		for _, rule := range rules.SiteRules {
			recordCnt := 1
			e.ForEach(rule.Selector, func(id int, e *colly.HTMLElement) {
				if (rule.First.Present() && rule.First.OrElse(0) > recordCnt) || (rule.Last.Present() && recordCnt > rule.Last.OrElse(0)) {
				} else {
					var result string
					switch rule.ResultType {
					case "text", "":
						result = strings.TrimSpace(e.Text)
					case "attr":
						result = strings.TrimSpace(e.Attr(rule.Attribute))
					}
					if len(rule.PostProcessing) > 0 {
						result = postProcessing(rule, result, e)
					}
					if assignField(rule.XbvrField, result, actor) {
						actorChanged = true
					}
					if data[rule.XbvrField] == "" {
						data[rule.XbvrField] = result
					} else {
						data[rule.XbvrField] = data[rule.XbvrField] + ", " + result
					}
				}
				recordCnt += 1
			})
		}
	})

	actorCollector.Visit(actorPage)
	var extref models.ExternalReference
	var extreflink models.ExternalReferenceLink

	db, _ := models.GetDB()
	defer db.Close()
	db.Preload("ExternalReference").
		Where(&models.ExternalReferenceLink{ExternalSource: source, InternalDbId: actor.ID}).
		First(&extreflink)
	extref = extreflink.ExternalReference

	if actorChanged || extref.ID == 0 {
		actor.Save()
		dataJson, _ := json.Marshal(data)

		extrefLink := []models.ExternalReferenceLink{models.ExternalReferenceLink{InternalTable: "actors", InternalDbId: actor.ID, InternalNameId: actor.Name, ExternalSource: source, ExternalId: actorPage}}
		extref = models.ExternalReference{ID: extref.ID, XbvrLinks: extrefLink, ExternalSource: source, ExternalId: actorPage, ExternalURL: actorPage, ExternalDate: time.Now(), ExternalData: string(dataJson)}
		extref.AddUpdateWithId()
	} else {
		extref.ExternalDate = time.Now()
		extref.AddUpdateWithId()
	}
}
func getSubRuleResult(rule GenericActorScraperRule, e *colly.HTMLElement) string {
	recordCnt := 1
	var result string
	e.ForEach(rule.Selector, func(id int, e *colly.HTMLElement) {
		if (rule.First.Present() && rule.First.OrElse(0) > recordCnt) || (rule.Last.Present() && recordCnt > rule.Last.OrElse(0)) {
		} else {
			switch rule.ResultType {
			case "text", "":
				result = strings.TrimSpace(e.Text)
			case "attr":
				result = strings.TrimSpace(e.Attr(rule.Attribute))
			}
			if len(rule.PostProcessing) > 0 {
				result = postProcessing(rule, result, e)
			}
		}
		recordCnt += 1
	})
	return result
}

func checkActorUpdateRequired(linkUrl string, actor *models.Actor) bool {
	db, _ := models.GetDB()
	defer db.Close()

	var extRefLink models.ExternalReferenceLink
	db.Preload("ExternalReference").
		Where("internal_db_id = ? and external_id = ?", actor.ID, linkUrl).First(&extRefLink)
	if extRefLink.ID != 0 {
		for _, scene := range actor.Scenes {
			if extRefLink.ExternalReference.ExternalDate.Before(scene.CreatedAt) {
				return true
			}
		}
	}

	return true
}
func assignField(field string, value string, actor *models.Actor) bool {
	changed := false
	switch field {
	case "birth_date":
		// check Birth date is not in the last 15 years, some sites just set the BirthDay to the current date when created
		t, err := time.Parse("2006-01-02", value)
		if err == nil && actor.BirthDate.IsZero() && t.Before(time.Now().AddDate(-15, 0, 0)) {
			actor.BirthDate = t
			changed = true
		}
	case "height":
		num, _ := strconv.Atoi(value)
		if actor.Height == 0 && num > 0 {
			actor.Height = num
			changed = true
		}
	case "weight":
		num, _ := strconv.Atoi(value)
		if actor.Weight == 0 && num > 0 {
			actor.Weight = num
			changed = true
		}
	case "nationality":
		if actor.Nationality == "" && value > "" {
			actor.Nationality = value
			changed = true
		}
	case "ethnicity":
		if actor.Ethnicity == "" && value > "" {
			actor.Ethnicity = value
			changed = true
		}
	case "band_size":
		num, _ := strconv.Atoi(value)
		if actor.BandSize == 0 && num > 0 {
			actor.BandSize = num
			changed = true
		}
	case "waist_size":
		num, _ := strconv.Atoi(value)
		if actor.WaistSize == 0 && num > 0 {
			actor.WaistSize = num
			changed = true
		}
	case "hip_size":
		num, _ := strconv.Atoi(value)
		if actor.HipSize == 0 && num > 0 {
			actor.HipSize = num
			changed = true
		}
	case "cup_size":
		if actor.CupSize == "" && value > "" {
			actor.CupSize = value
			changed = true
		}
	case "eye_color":
		if actor.EyeColor == "" && value > "" {
			actor.EyeColor = value
			changed = true
		}
	case "hair_color":
		if actor.HairColor == "" && value > "" {
			actor.HairColor = value
			changed = true
		}
	case "biography":
		if actor.Biography == "" && value > "" {
			actor.Biography = value
			changed = true
		}
	case "image_url":
		if actor.AddToImageArray(value) {
			changed = true
		}
		if actor.ImageUrl == "" && value != "" {
			actor.ImageUrl = value
			changed = true
		}
	case "images":
		if actor.AddToImageArray(value) {
			changed = true
		}
	case "aliases":
		array := strings.Split(value, ",")
		for _, item := range array {
			if actor.AddToAliases(strings.TrimSpace(item)) {
				changed = true
			}
		}
	case "piercings":
		log.Infof("piercings %s", value)
		array := strings.Split(value, ",")
		for _, item := range array {
			if actor.AddToPiercings(strings.TrimSpace(item)) {
				changed = true
			}
		}
	case "tattoos":
		log.Infof("tattoos %s", value)
		array := strings.Split(value, ",")
		for _, item := range array {
			if actor.AddToTattoos(strings.TrimSpace(item)) {
				changed = true
			}
		}
	}
	return changed
}
func getRegexResult(value string, pattern string, pos int) string {
	re := regexp.MustCompile(pattern)
	if pos == 0 {
		return re.FindString(value)
	} else {
		groups := re.FindStringSubmatch(value)
		if len(groups) < pos+1 {
			return ""
		}
		return re.FindStringSubmatch(value)[pos]
	}

}

func postProcessing(rule GenericActorScraperRule, value string, htmlElement *colly.HTMLElement) string {
	for _, postprocessing := range rule.PostProcessing {
		switch postprocessing.Function {
		case "Lookup Country":
			value = getCountryCode(value)
		case "Parse Date":
			t, err := time.Parse(postprocessing.Params[0], strings.Replace(strings.Replace(strings.Replace(strings.Replace(value, "st", "", -1), "nd", "", -1), "rd", "", -1), "th", "", -1))
			if err != nil {
				return ""
			}
			value = t.Format("2006-01-02")
		case "inch to cm":
			num, _ := strconv.ParseFloat(value, 64)
			num = num * 2.54
			value = strconv.Itoa(int(math.Round(num)))
		case "Feet+Inches to cm":
			re := regexp.MustCompile(`(\d+)\'(\d+)\"`)
			matches := re.FindStringSubmatch(value)
			if len(matches) >= 3 {
				feet, _ := strconv.Atoi(matches[1])
				inches, _ := strconv.Atoi(matches[2])
				num := float64(feet*12+inches) * 2.54
				value = strconv.Itoa(int(math.Round(num)))
			}
		case "jsonString":
			value = strings.TrimSpace(html.UnescapeString(gjson.Get(value, postprocessing.Params[0]).String()))
		case "RegexString":
			pos, _ := strconv.Atoi(postprocessing.Params[1])
			value = getRegexResult(value, postprocessing.Params[0], pos)
		case "Replace":
			value = strings.Replace(value, postprocessing.Params[0], postprocessing.Params[1], 1)
		case "AbsoluteUrl":
			value = htmlElement.Request.AbsoluteURL(value)
		case "CollyForEach":
			value = getSubRuleResult(postprocessing.SubRule, htmlElement)
		case "DOMNext":
			value = strings.TrimSpace(htmlElement.DOM.Next().Text())
		}
	}
	return value
}

type GenericActorScraperRule struct {
	XbvrField      string           `json:"xbvr_field"`
	Selector       string           `json:"selector`        // css selector to identify data
	PostProcessing []PostProcessing `json:"post_processing` // call routines for specific handling, eg dates parshing, json extracts, etc, see PostProcessing function
	First          optional.Int     `json:"first"`          // used to limit how many results you want, the start position you want.  First index pos	 is 0
	Last           optional.Int     `json:"last"`           // used to limit how many results you want, the end position you want
	ResultType     string           `json:"result_type"`    // how to treat the result, text, attribute value, json
	Attribute      string           `json:"attribute`       // name of the atribute you want
}
type PostProcessing struct {
	Function string                  `json:"post_processing` // call routines for specific handling, eg dates, json extracts
	Params   []string                `json:"params`          // used to pass params to PostProcessing functions, eg date format
	SubRule  GenericActorScraperRule `json:"sub_rule`        // sub rules allow for a foreach within a foreach, use Function CollyForEach
}
type GenericActorScraperRules struct {
	SiteRules []GenericActorScraperRule `json:"rules"`
	Source    string                    `json:"source"`
	Domain    string                    `json:"domain"`
}

func getCountryCode(countryName string) string {
	switch strings.ToLower(countryName) {
	case "united states", "american":
		return "US"
	case "english", "scottish":
		return "GB"
	default:
		code, err := lookupCountryCode(countryName)
		if err != nil {
			return countryName
		} else {
			return code
		}
	}
}

func lookupCountryCode(countryName string) (string, error) {
	// Construct the API URL with the country name as a query parameter
	url := fmt.Sprintf("https://restcountries.com/v2/name/%s", countryName)

	// Send a GET request to the API and decode the JSON response
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var countries []struct {
		Alpha2Code string `json:"alpha2Code"`
	}
	err = json.NewDecoder(resp.Body).Decode(&countries)
	if err != nil {
		return "", err
	}

	// Check if a country code was found
	if len(countries) == 0 {
		return "", fmt.Errorf("no country code found for %s", countryName)
	}

	return countries[0].Alpha2Code, nil
}

func structToMap(obj interface{}) map[string]interface{} {
	values := reflect.ValueOf(obj)
	typ := values.Type()

	result := make(map[string]interface{})
	for i := 0; i < values.NumField(); i++ {
		key := typ.Field(i).Name
		value := values.Field(i).Interface()
		result[key] = value
	}

	return result
}

func BuildRules() []GenericActorScraperRules {
	var siteActorScrapeRules []GenericActorScraperRules

	// , PostProcessing: []PostProcessing{{Function:"RegexString", Params: []string{`^(Eye color: )(.+),"2"}}}})

	siteDetails := GenericActorScraperRules{}
	siteDetails.Source = "zexyvr scrape"
	siteDetails.Domain = "zexyvr.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "birth_date", Selector: `li:contains("Birth date") > b`, PostProcessing: []PostProcessing{{Function: "Parse Date", Params: []string{"Jan 2, 2006"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "height", Selector: `li:contains("Height") > b:first-of-type`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d+`, "0"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "nationality", Selector: "li:contains(\"Nationality\") > b", PostProcessing: []PostProcessing{{Function: "Lookup Country"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "band_size", Selector: "li:contains(\"Bra size\") > b:first-of-type", PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d+`, "0"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "cup_size", Selector: "li:contains(\"Bra size\") > b:first-of-type", PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`[A-K]{1,2}`, "0"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "eye_color", Selector: "li:contains(\"Eye Color\") > b"})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hair_color", Selector: "li:contains(\"Hair Color\") > b"})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "weight", Selector: "li:contains(\"Weight\") > b:first-of-type", PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d+`, "0"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "images", Selector: `div.col-12.col-lg-5 > img, div.col-12.col-lg-7 img`, ResultType: "attr", Attribute: "src"})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `div.col-12.col-lg-5 > img`, ResultType: "attr", Attribute: "src", First: optional.NewInt(0)})
	siteActorScrapeRules = append(siteActorScrapeRules, siteDetails)

	siteDetails.Source = "wankitnowvr scrape"
	siteDetails.Domain = "wankitnowvr.com"
	siteActorScrapeRules = append(siteActorScrapeRules, siteDetails)

	siteDetails = GenericActorScraperRules{}
	siteDetails.Source = "slr scrape"
	siteDetails.Domain = "www.sexlikereal.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "birth_date", Selector: `div[data-qa="model-info-birth"] div.u-wh`, PostProcessing: []PostProcessing{{Function: "Parse Date", Params: []string{"January 2, 2006"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "height", Selector: `div[data-qa="model-info-height"] div.u-wh`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d+`, "0"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "nationality", Selector: `div[data-qa="model-info-country"] div.u-wh`, PostProcessing: []PostProcessing{{Function: "Lookup Country"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "weight", Selector: `div[data-qa="model-info-weight"] div.u-wh`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d+`, "0"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "aliases", Selector: `div[data-qa="model-info-aliases"] div.u-wh`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "biography", Selector: `div[data-qa="model-info-bio"] div.u-wh`})
	siteActorScrapeRules = append(siteActorScrapeRules, siteDetails)

	siteDetails = GenericActorScraperRules{}
	siteDetails.Source = "baberoticavr scrape"
	siteDetails.Domain = "baberoticavr.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "birth_date", Selector: `div[id="model"] div:contains('Birth date:')+div`, PostProcessing: []PostProcessing{{Function: "Parse Date", Params: []string{"January 2, 2006"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "eye_color", Selector: `div[id="model"] div:contains('Eye Color:')+div`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hair_color", Selector: `div[id="model"] div:contains('Hair color:')+div`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "height", Selector: `div[id="model"] div:contains('Height:')+div`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d+`, "0"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "weight", Selector: `div[id="model"] div:contains('Weight:')+div`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d+`, "0"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "ethnicity", Selector: `div[id="model"] div:contains('Ethnicity:')+div`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "nationality", Selector: `div[id="model"] div:contains('Country:')+div`, PostProcessing: []PostProcessing{{Function: "Lookup Country"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "aliases", Selector: `div[id="model"] div:contains('Aliases:')+div`})
	//siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "piercings", Selector: `div[id="model"] div:contains('Piercings:')+div`, Regex: "^(No)(.+)"})
	//siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "tattoos", Selector: `div[id="model"] div:contains('Tattoos:')+div`, Regex: "^(No)(.+)"})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `div.m5 img`, ResultType: "attr", Attribute: "src", First: optional.NewInt(0)})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "band_size", Selector: `div[id="model"] div:contains('Body:')+div`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(B)(\d{2})`, "2"}}, {Function: "inch to cm"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "waist_size", Selector: `div[id="model"] div:contains('Body:')+div`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(W)(\d{2})`, "2"}}, {Function: "inch to cm"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hip_size", Selector: `div[id="model"] div:contains('Body:')+div`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(H)(\d{2})`, "2"}}, {Function: "inch to cm"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "cup_size", Selector: `div[id="model"] div:contains('Breasts Cup:')+div`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`[A-K]{1,2}`, "0"}}}})
	siteActorScrapeRules = append(siteActorScrapeRules, siteDetails)

	siteDetails = GenericActorScraperRules{}
	siteDetails.Source = "vrporn scrape"
	siteDetails.Domain = "vrporn.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "birth_date", Selector: `li:contains('Birthdate:')`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`^(Birthdate: )(.+)`, "2"}}, {Function: "Parse Date", Params: []string{"02/01/2006"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "nationality", Selector: `li:contains('Country of origin:')`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`^(Country of origin: )(.+)`, "2"}}, {Function: "Lookup Country"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "height", Selector: `li:contains('Height:')`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`^(Height: )(\d{2,3})`, "2"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "weight", Selector: `li:contains('Weight:')`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`^(Weight: )(\d{2,3})`, "2"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "band_size", Selector: `li:contains('Breast Size:')`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`^(Breast Size: )(\d{2,3})`, "2"}}, {Function: "inch to cm"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "cup_size", Selector: `li:contains('Breast Size:')`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`^(Breast Size: )(\d{2,3})(.+)`, "3"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hair_color", Selector: `li:contains('Hair color:')`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`^(Hair color: )(.+)`, "2"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "eye_color", Selector: `li:contains('Eye color:')`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`^(Eye color: )(.+)`, "2"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "aliases", Selector: `div.list_aliases_pornstar li`})
	siteActorScrapeRules = append(siteActorScrapeRules, siteDetails)

	siteDetails = GenericActorScraperRules{}
	siteDetails.Source = "virtualrealporn scrape"
	siteDetails.Domain = "virtualrealporn.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "birth_date", Selector: `script[type="application/ld+json"][class!='yoast-schema-graph']`,
		PostProcessing: []PostProcessing{{Function: "jsonString", Params: []string{"birthDate"}},
			{Function: "Parse Date", Params: []string{"01/02/2006"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "nationality", Selector: `script[type="application/ld+json"][class!='yoast-schema-graph']`,
		PostProcessing: []PostProcessing{{Function: "jsonString", Params: []string{"birthPlace"}}, {Function: "Lookup Country"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `script[type="application/ld+json"][class!='yoast-schema-graph']`,
		PostProcessing: []PostProcessing{{Function: "jsonString", Params: []string{"image"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "eye_color", Selector: `table[id="table_about"] tr th:contains('Eyes Color')+td`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hair_color", Selector: `table[id="table_about"] tr th:contains('Hair Color')+td`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "band_size", Selector: `table[id="table_about"] tr th:contains('Bust')+td`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "waist_size", Selector: `table[id="table_about"] tr th:contains('Waist')+td`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hip_size", Selector: `table[id="table_about"] tr th:contains('Hips')+td`})
	siteActorScrapeRules = append(siteActorScrapeRules, siteDetails)

	siteDetails.Source = "virtualrealtrans scrape"
	siteDetails.Domain = "virtualrealtrans.com"
	siteActorScrapeRules = append(siteActorScrapeRules, siteDetails)

	siteDetails.Source = "virtualrealgay scrape"
	siteDetails.Domain = "virtualrealgay.com"
	siteActorScrapeRules = append(siteActorScrapeRules, siteDetails)

	siteDetails.Source = "virtualrealpassion scrape"
	siteDetails.Domain = "virtualrealpassion.com"
	siteActorScrapeRules = append(siteActorScrapeRules, siteDetails)

	siteDetails.Source = "virtualrealamateurporn scrape"
	siteDetails.Domain = "virtualrealamateurporn.com"
	siteActorScrapeRules = append(siteActorScrapeRules, siteDetails)

	siteDetails = GenericActorScraperRules{}
	siteDetails.Source = "groobyvr scrape"
	siteDetails.Domain = "www.groobyvr.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `div.model_photo img`, ResultType: "attr", Attribute: "src",
		PostProcessing: []PostProcessing{{Function: "AbsoluteUrl"}}})

	//subRule := GenericActorScraperRule{Selector: `li`, First: optional.NewInt(2), Last: optional.NewInt(2)}  // turns out the subquery wasn't needed in this case
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "biography", Selector: `div[id="bio"] ul`, First: optional.NewInt(1), Last: optional.NewInt(1)})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "ethnicity", Selector: `div[id="bio"] li:contains('Ethnicity:')`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`^(Ethnicity: )(.+)`, "2"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "nationality", Selector: `div[id="bio"] li:contains('Nationality:')`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`^(Nationality: )(.+)`, "2"}}, {Function: "Lookup Country"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "height", Selector: `div[id="bio"] li:contains('Height:')`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`^(Height: )(.+)`, "2"}}, {Function: "Feet+Inches to cm"}}})
	siteActorScrapeRules = append(siteActorScrapeRules, siteDetails)

	siteDetails = GenericActorScraperRules{}
	siteDetails.Source = "hologirlsvr scrape"
	siteDetails.Domain = "www.hologirlsvr.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "height", Selector: `.starBio`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d+\s*ft\s*\d+\s*in`, "0"}},
			{Function: "Replace", Params: []string{" ft ", `'`}},
			{Function: "Replace", Params: []string{" in", `"`}},
			{Function: "Feet+Inches to cm"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "band_size", Selector: `.starBio`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(\d{2,3}).{1,2}-\d{2,3}-\d{2,3}`, "1"}},
			{Function: "inch to cm"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "cup_size", Selector: `.starBio`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d{2,3}(.{1,2})-\d{2,3}-\d{2,3}`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "waist_size", Selector: `.starBio`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d{2,3}.{1,2}-(\d{2,3})-\d{2,3}`, "1"}},
			{Function: "inch to cm"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hip_size", Selector: `.starBio`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d{2,3}.{1,2}-\d{2,3}-(\d{2,3})`, "1"}},
			{Function: "inch to cm"}}})

	siteActorScrapeRules = append(siteActorScrapeRules, siteDetails)

	siteDetails = GenericActorScraperRules{}
	siteDetails.Source = "vrbangers scrape"
	siteDetails.Domain = "vrbangers.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `div.single-model-profile__image > img`, ResultType: "attr", Attribute: "src"})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "biography", Selector: `div.single-model-biography__content div.toggle-content__text`, First: optional.NewInt(1), Last: optional.NewInt(1)})
	siteActorScrapeRules = append(siteActorScrapeRules, siteDetails)
	siteDetails.Source = "vrbtrans scrape"
	siteDetails.Domain = "vrbtrans.com"
	siteActorScrapeRules = append(siteActorScrapeRules, siteDetails)
	siteDetails.Source = "vrbgay scrape"
	siteDetails.Domain = "vrbgay.com"
	siteActorScrapeRules = append(siteActorScrapeRules, siteDetails)
	siteDetails.Source = "vrconk scrape"
	siteDetails.Domain = "vrconk.com"
	siteActorScrapeRules = append(siteActorScrapeRules, siteDetails)
	siteDetails.Source = "blowvr scrape"
	siteDetails.Domain = "blowvr.com"
	siteActorScrapeRules = append(siteActorScrapeRules, siteDetails)

	siteDetails = GenericActorScraperRules{}
	siteDetails.Source = "bvr scrape"
	siteDetails.Domain = "virtualporn.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `div.model__img-wrapper > img`, ResultType: "attr", Attribute: "src"})
	siteActorScrapeRules = append(siteActorScrapeRules, siteDetails)

	siteDetails = GenericActorScraperRules{}
	siteDetails.Source = "realitylovers scrape"
	siteDetails.Domain = "realitylovers.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `img.girlDetails-posterImage`, ResultType: "attr", Attribute: "srcset",
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(.*) \dx,`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "birth_date", Selector: `.girlDetails-info`, PostProcessing: []PostProcessing{
		{Function: "RegexString", Params: []string{`(.{3} \d{2}.{2} \d{4})`, "1"}},
		{Function: "Parse Date", Params: []string{"Jan 02 2006"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "nationality", Selector: `.girlDetails-info`, PostProcessing: []PostProcessing{
		{Function: "RegexString", Params: []string{`Country:\s*(.*)`, "1"}},
		{Function: "Lookup Country"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "height", Selector: `.girlDetails-info`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(\d{2,3}) cm`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "weight", Selector: `.girlDetails-info`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(\d{2,3}) kg`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "biography", Selector: `.girlDetails-bio`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`Biography:\s*(.*)`, "1"}}}})
	siteActorScrapeRules = append(siteActorScrapeRules, siteDetails)

	return siteActorScrapeRules
}
