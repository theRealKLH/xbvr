package scrape

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/markphelps/optional"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
)

type ActorScraperRulesMap map[string]GenericScraperRuleSet
type ActorScraperRules struct {
	Rules ActorScraperRulesMap
}
type GenericScraperRuleSet struct {
	SiteRules []GenericActorScraperRule `json:"rules"`
	Domain    string                    `json:"domain"`
	isJson    bool                      `json:"isJson"`
}

func (siteActorScrapeRules ActorScraperRules) BuildRules() {
	db, _ := models.GetDB()
	defer db.Close()
	var sites []models.Site

	// To understand the regex used, sign up to chat.openai.com and just ask something like Explain (.*, )?(.*)$
	// To test regex I use https://regex101.com/
	siteDetails := GenericScraperRuleSet{}
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
	siteActorScrapeRules.Rules["zexyvr scrape"] = siteDetails

	siteDetails.Domain = "wankitnowvr.com"
	siteActorScrapeRules.Rules["wankitnowvr scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "www.sexlikereal.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `script[type="application/ld+json"]:contains("\/schema.org\/\",\"@type\": \"Person")`, PostProcessing: []PostProcessing{{Function: "jsonString", Params: []string{"image"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "birth_date", Selector: `script[type="application/ld+json"]:contains("\/schema.org\/\",\"@type\": \"Person")`,
		PostProcessing: []PostProcessing{
			{Function: "jsonString", Params: []string{"birthDate"}},
			{Function: "Parse Date", Params: []string{"January 2, 2006"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "height", Selector: `script[type="application/ld+json"]:contains("\/schema.org\/\",\"@type\": \"Person")`, PostProcessing: []PostProcessing{
		{Function: "jsonString", Params: []string{"height"}},
		{Function: "RegexString", Params: []string{`(\d{3})\s?cm`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "weight", Selector: `script[type="application/ld+json"]:contains("\/schema.org\/\",\"@type\": \"Person")`, PostProcessing: []PostProcessing{
		{Function: "jsonString", Params: []string{"weight"}},
		{Function: "RegexString", Params: []string{`(\d{2,3})\s?kg`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "nationality", Selector: `script[type="application/ld+json"]:contains("\/schema.org\/\",\"@type\": \"Person")`,
		PostProcessing: []PostProcessing{
			{Function: "jsonString", Params: []string{"nationality"}},
			{Function: "RegexString", Params: []string{`^(.*,)?\s?(.*)$`, "2"}},
			{Function: "Lookup Country"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "biography", Selector: `script[type="application/ld+json"]:contains("\/schema.org\/\",\"@type\": \"Person")`, PostProcessing: []PostProcessing{{Function: "jsonString", Params: []string{"description"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "aliases", Selector: `div[data-qa="model-info-aliases"] div.u-wh`})
	siteActorScrapeRules.Rules["slr-originals scrape"] = siteDetails
	siteActorScrapeRules.Rules["slr-jav-originals scrape"] = siteDetails
	db.Where("name like ?", "%SLR)").Find(&sites)
	siteActorScrapeRules.Rules["slr scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "baberoticavr.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "birth_date", Selector: `div[id="model"] div:contains('Birth date:')+div`, PostProcessing: []PostProcessing{{Function: "Parse Date", Params: []string{"January 2, 2006"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "eye_color", Selector: `div[id="model"] div:contains('Eye Color:')+div`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hair_color", Selector: `div[id="model"] div:contains('Hair color:')+div`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "height", Selector: `div[id="model"] div:contains('Height:')+div`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d+`, "0"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "weight", Selector: `div[id="model"] div:contains('Weight:')+div`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d+`, "0"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "ethnicity", Selector: `div[id="model"] div:contains('Ethnicity:')+div`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "nationality", Selector: `div[id="model"] div:contains('Country:')+div`, PostProcessing: []PostProcessing{{Function: "Lookup Country"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "aliases", Selector: `div[id="model"] div:contains('Aliases:')+div`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `div.m5 img`, ResultType: "attr", Attribute: "src", First: optional.NewInt(0)})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "band_size", Selector: `div[id="model"] div:contains('Body:')+div`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(B)(\d{2})`, "2"}}, {Function: "inch to cm"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "waist_size", Selector: `div[id="model"] div:contains('Body:')+div`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(W)(\d{2})`, "2"}}, {Function: "inch to cm"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hip_size", Selector: `div[id="model"] div:contains('Body:')+div`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(H)(\d{2})`, "2"}}, {Function: "inch to cm"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "cup_size", Selector: `div[id="model"] div:contains('Breasts Cup:')+div`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`[A-K]{1,2}`, "0"}}}})
	siteActorScrapeRules.Rules["baberoticavr scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
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
	siteActorScrapeRules.Rules["vrporn scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
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
	siteActorScrapeRules.Rules["virtualrealporn scrape"] = siteDetails

	siteDetails.Domain = "virtualrealtrans.com"
	siteActorScrapeRules.Rules["virtualrealtrans scrape"] = siteDetails

	siteDetails.Domain = "virtualrealgay.com"
	siteActorScrapeRules.Rules["virtualrealgay scrape"] = siteDetails

	siteDetails.Domain = "virtualrealpassion.com"
	siteActorScrapeRules.Rules["virtualrealpassion scrape"] = siteDetails

	siteDetails.Domain = "virtualrealamateurporn.com"
	siteActorScrapeRules.Rules["virtualrealamateurporn scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "www.groobyvr.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `div.model_photo img`, ResultType: "attr", Attribute: "src",
		PostProcessing: []PostProcessing{{Function: "AbsoluteUrl"}}})

	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "biography", Selector: `div[id="bio"] ul`, First: optional.NewInt(1), Last: optional.NewInt(1)})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "ethnicity", Selector: `div[id="bio"] li:contains('Ethnicity:')`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`^(Ethnicity: )(.+)`, "2"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "nationality", Selector: `div[id="bio"] li:contains('Nationality:')`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`^(Nationality: )(.+)`, "2"}}, {Function: "Lookup Country"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "height", Selector: `div[id="bio"] li:contains('Height:')`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`^(Height: )(.+)`, "2"}}, {Function: "Feet+Inches to cm", Params: []string{`(\d+)\'(\d+)\"`, "1", "2"}}}})
	siteActorScrapeRules.Rules["groobyvr scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "www.hologirlsvr.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "height", Selector: `.starBio`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d+\s*ft\s*\d+\s*in`, "0"}},
			{Function: "Replace", Params: []string{" ft ", `'`}},
			{Function: "Replace", Params: []string{" in", `"`}},
			{Function: "Feet+Inches to cm", Params: []string{`(\d+)\'(\d+)\"`, "1", "2"}}}})
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
	siteActorScrapeRules.Rules["hologirlsvr scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "vrbangers.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `div.single-model-profile__image > img`, ResultType: "attr", Attribute: "src"})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "biography", Selector: `div.single-model-biography__content div.toggle-content__text`, First: optional.NewInt(1), Last: optional.NewInt(1)})
	siteActorScrapeRules.Rules["vrbangers scrape"] = siteDetails
	siteDetails.Domain = "vrbtrans.com"
	siteActorScrapeRules.Rules["vrbtrans scrape"] = siteDetails
	siteDetails.Domain = "vrbgay.com"
	siteActorScrapeRules.Rules["vrbgay scrape"] = siteDetails
	siteDetails.Domain = "vrconk.com"
	siteActorScrapeRules.Rules["vrconk scrape"] = siteDetails
	siteDetails.Domain = "blowvr.com"
	siteActorScrapeRules.Rules["blowvr scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "virtualporn.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `div.model__img-wrapper > img`, ResultType: "attr", Attribute: "src"})
	siteActorScrapeRules.Rules["bvr scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
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
	siteActorScrapeRules.Rules["realitylovers scrape"] = siteDetails

	siteDetails.Domain = "tsvirtuallovers.com"
	siteActorScrapeRules.Rules["tsvirtuallovers scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "vrphub.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `.model-thumb img`, ResultType: "attr", Attribute: "src"})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "aliases", Selector: `span.details:contains("Aliases:") + span.details-value`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "ethnicity", Selector: `span.details:contains("Ethnicity:") + span.details-value`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "band_size", Selector: `span.details:contains("Measurements:") + span.details-value`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(\d{2,3}).{1,2}-\d{2,3}-\d{2,3}`, "1"}},
			{Function: "inch to cm"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "cup_size", Selector: `span.details:contains("Measurements:") + span.details-value`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d{2,3}(.{1,2})-\d{2,3}-\d{2,3}`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "waist_size", Selector: `span.details:contains("Measurements:") + span.details-value`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d{2,3}.{1,2}-(\d{2,3})-\d{2,3}`, "1"}},
			{Function: "inch to cm"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hip_size", Selector: `span.details:contains("Measurements:") + span.details-value`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d{2,3}.{1,2}-\d{2,3}-(\d{2,3})`, "1"}},
			{Function: "inch to cm"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "band_size", Selector: `span.details:contains("Bra cup size:") + span.details-value`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(\d{2,3}).{1,2}`, "1"}},
			{Function: "inch to cm"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "cup_size", Selector: `span.details:contains("Bra cup size:") + span.details-value`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d{2,3}(.{1,2})`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "tattoos", Selector: `span.tattoo-value`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(No Tattoos)?(.*)`, "2"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "piercings", Selector: `span.details:contains("Piercings:") + span.details-value`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(No Piercings)?(.*)`, "2"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "biography", Selector: `span.bio-details`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "birth_date", Selector: `span.details:contains("Date of birth:") + span.details-value`,
		PostProcessing: []PostProcessing{{Function: "Parse Date", Params: []string{"January 2, 2006"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "nationality", Selector: `span.details:contains("Birthplace:") + span.details-value`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(.*, )?(.*)$`, "2"}},
			{Function: "Lookup Country"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hair_color", Selector: `span.details:contains("Hair Color:") + span.details-value`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "eye_color", Selector: `span.details:contains("Eye Color:") + span.details-value`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "height", Selector: `span.details:contains("Height:") + span.details-value`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(\d{2,3}) cm`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "weight", Selector: `span.details:contains("Weight:") + span.details-value`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(\d{2,3}) kg`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "urls", Selector: `.model-info-block2 a`, ResultType: "attr", Attribute: "href"})
	siteActorScrapeRules.Rules["vrphub scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "vrhush.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `img[id="model-thumbnail"]`, ResultType: "attr", Attribute: "src", PostProcessing: []PostProcessing{{Function: "AbsoluteUrl"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "biography", Selector: `div[id="model-info-block"] p`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "ethnicity", Selector: `ul.model-attributes li:contains("Ethnicity")`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`Ethnicity (.*)`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "eye_color", Selector: `ul.model-attributes li:contains("Eye Color")`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`Eye Color (.*)`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "height", Selector: `ul.model-attributes li:contains("Height")`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`^(Height )(.+)`, "2"}}, {Function: "Feet+Inches to cm", Params: []string{`(\d+)\'(\d+)\"`, "1", "2"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "gender", Selector: `ul.model-attributes li:contains("Gender")`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`Gender (.*)`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hair_color", Selector: `ul.model-attributes li:contains("Hair Color")`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`Hair Color (.*)`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "weight", Selector: `ul.model-attributes li:contains("Weight")`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`^(Weight )(.+)`, "2"}}, {Function: "lbs to kg"}}})
	siteActorScrapeRules.Rules["vrhush scrape"] = siteDetails

	siteDetails.Domain = "vrallure.com"
	siteActorScrapeRules.Rules["vrallure scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "vrlatina.com"
	// The data-pagespeed-lazy-src attribute holds the URL of the image that should be loaded lazily, the PageSpeed module dynamically replaces the data-pagespeed-lazy-src attribute with the standard src attribute, triggering the actual loading of the image.
	// In my testing sometime, I got the data-pagespeed-lazy-src with a blank image in the src attribute (with a relative url) and other times I just got src with the correct url
	// The following will first load the data-pagespeed-lazy-src then the src attribute.  The check for thehttp prefix, stops the blank image been processed with the relative url
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `div.model-avatar img`, ResultType: "attr", Attribute: "data-pagespeed-lazy-src", PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(http.*)`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `div.model-avatar img`, ResultType: "attr", Attribute: "src", PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(http.*)`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "aliases", Selector: `ul.model-list>li:contains("Aka:")>span+span`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "birth_date", Selector: `ul.model-list>li:contains("Dob:")>span+span`, PostProcessing: []PostProcessing{{Function: "Parse Date", Params: []string{"2006-01-02"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "height", Selector: `ul.model-list>li:contains("Height:")>span+span`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(\d{2,3})`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "weight", Selector: `ul.model-list>li:contains("Weight:")>span+span`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(\d{2,3})`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "band_size", Selector: `ul.model-list>li:contains("Measurements:")>span+span`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(\d{2,3}).{1,2}`, "1"}},
			{Function: "inch to cm"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "cup_size", Selector: `ul.model-list>li:contains("Measurements:")>span+span`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d{2,3}(.{1,2})`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hair_color", Selector: `ul.model-list>li:contains("Hair:")>span+span`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "eye_color", Selector: `ul.model-list>li:contains("Eyes:")>span+span`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "biography", Selector: `ul.model-list>li:contains("Biography:")>span+span`})
	siteActorScrapeRules.Rules["vrlatina scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "badoinkvr.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `img.girl-details-photo`, ResultType: "attr", Attribute: "src"})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "band_size", Selector: `.girl-details-stats-item:contains("Measurements:")>span+span`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(\d{2,3}).{1,2}-\d{2,3}-\d{2,3}`, "1"}},
			{Function: "inch to cm"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "cup_size", Selector: `.girl-details-stats-item:contains("Measurements:")>span+span`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d{2,3}(.{1,2})-\d{2,3}-\d{2,3}`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "waist_size", Selector: `.girl-details-stats-item:contains("Measurements:")>span+span`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d{2,3}.{1,2}-(\d{2,3})-\d{2,3}`, "1"}},
			{Function: "inch to cm"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hip_size", Selector: `.girl-details-stats-item:contains("Measurements:")>span+span`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d{2,3}.{1,2}-\d{2,3}-(\d{2,3})`, "1"}},
			{Function: "inch to cm"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "height", Selector: `.girl-details-stats-item:contains("Height:")>span+span`,
		PostProcessing: []PostProcessing{{Function: "Feet+Inches to cm", Params: []string{`(\d+)\D*(\d{1,2})`, "1", "2"}}}})

	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "weight", Selector: `.girl-details-stats-item:contains("Weight:")>span+span`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(\d{2,3})`, "1"}}, {Function: "lbs to kg"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "aliases", Selector: `.girl-details-stats-item:contains("Aka:")>span+span`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "nationality", Selector: `.girl-details-stats-item:contains("Country:")>span+span`,
		PostProcessing: []PostProcessing{{Function: "Lookup Country"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hair_color", Selector: `.girl-details-stats-item:contains("Hair:")>span+span`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "eye_color", Selector: `.girl-details-stats-item:contains("Eyes:")>span+span`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "ethnicity", Selector: `.girl-details-stats-item:contains("Ethnicity:")>span+span`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "biography", Selector: `div.girl-details-bio p`})
	siteActorScrapeRules.Rules["badoinkvr scrape"] = siteDetails

	siteDetails.Domain = "babevr.com"
	siteActorScrapeRules.Rules["babevr scrape"] = siteDetails
	siteDetails.Domain = "vrcosplayx.com"
	siteActorScrapeRules.Rules["vrcosplayx scrape"] = siteDetails
	siteDetails.Domain = "18vr.com"
	siteActorScrapeRules.Rules["18vr scrape"] = siteDetails
	siteDetails.Domain = "kinkvr.com"
	siteActorScrapeRules.Rules["kinkvr scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "darkroomvr.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `img.pornstar-detail__picture`, ResultType: "attr", Attribute: "src"})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "urls", Selector: `div.pornstar-detail__social a`, ResultType: "attr", Attribute: "href"})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "nationality", Selector: `div.pornstar-detail__info span`, Last: optional.NewInt(1),
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`^(.*?),`, "1"}},
			{Function: "Lookup Country"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "start_year", Selector: `div.pornstar-detail__info span:contains("Career Start")`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`Career Start .*(\d{4})`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "aliases", Selector: `div.pornstar-detail__info span:contains("aka ")`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`aka (.*)`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "birth_date", Selector: `div.pornstar-detail__params:contains("Birthday:")`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`Birthday: (.{3} \d{1,2}, \d{4})`, "1"}},
			{Function: "Parse Date", Params: []string{"Jan 2, 2006"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "band_size", Selector: `div.pornstar-detail__params:contains("Measurements:")`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(\d{2,3}).{1,2}(?:\s?-|\s-\s)\d{2,3}(?:\s?-|\s-\s)\d{2,3}`, "1"}},
			{Function: "inch to cm"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "cup_size", Selector: `div.pornstar-detail__params:contains("Measurements:")`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d{2,3}(.{1,2})(?:\s?-|\s-\s)\d{2,3}(?:\s?-|\s-\s)\d{2,3}`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "waist_size", Selector: `div.pornstar-detail__params:contains("Measurements:")`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d{2,3}.{1,2}(?:\s?-|\s-\s)(\d{2,3})(?:\s?-|\s-\s)\d{2,3}`, "1"}},
			{Function: "inch to cm"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hip_size", Selector: `div.pornstar-detail__params:contains("Measurements:")`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d{2,3}.{1,2}(?:\s?-|\s-\s)\d{2,3}(?:\s?-|\s-\s)(\d{2,3})`, "1"}},
			{Function: "inch to cm"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "height", Selector: `div.pornstar-detail__params:contains("Height:")`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`Height:\s*(\d{2,3})\s*cm`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "weight", Selector: `div.pornstar-detail__params:contains("Weight:")`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`Weight:\s*(\d{2,3})\s*kg`, "1"}}}})
	siteActorScrapeRules.Rules["darkroomvr scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "www.fuckpassvr.com"
	siteDetails.isJson = true

	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "nationality", Selector: `data.seo.porn_star.national`, PostProcessing: []PostProcessing{{Function: "Lookup Country"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "ethnicity", Selector: `data.seo.porn_star.ethnicity`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "eye_color", Selector: `data.seo.porn_star.eye_color`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hair_color", Selector: `data.seo.porn_star.hair_color`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "band_size", Selector: `data.seo.porn_star.measurement`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(\d{2,3}).{1,2}(?:\s?-|\s-\s)\d{2,3}(?:\s?-|\s-\s)\d{2,3}`, "1"}}, {Function: "inch to cm"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "cup_size", Selector: `data.seo.porn_star.measurement`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d{2,3}(.{1,2})(?:\s?-|\s-\s)\d{2,3}(?:\s?-|\s-\s)\d{2,3}`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "waist_size", Selector: `data.seo.porn_star.measurement`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d{2,3}.{1,2}(?:\s?-|\s-\s)(\d{2,3})(?:\s?-|\s-\s)\d{2,3}`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hip_size", Selector: `data.seo.porn_star.measurement`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`\d{2,3}.{1,2}(?:\s?-|\s-\s)\d{2,3}(?:\s?-|\s-\s)(\d{2,3})`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "birth_date", Selector: `data.seo.porn_star.birthday`, PostProcessing: []PostProcessing{{Function: "Parse Date", Params: []string{"2006-01-02"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "height", Selector: `data.seo.porn_star.height`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "weight", Selector: `data.seo.porn_star.weight`, PostProcessing: []PostProcessing{{Function: "lbs to kg"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "biography", Selector: `data.seo.porn_star.write_up`,
		PostProcessing: []PostProcessing{{Function: "Replace", Params: []string{"<p>", ``}},
			{Function: "Replace", Params: []string{"</p>", `
		`}},
			{Function: "Replace", Params: []string{"<br>", `
		`}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "urls", Selector: `data.seo.porn_star.slug`,
		PostProcessing: []PostProcessing{{Function: "RegexReplaceAll", Params: []string{`^(.*)$`, `https://www.fuckpassvr.com/vr-pornstars/$0`}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `data.seo.porn_star.thumbnail_url`}) // image will expiry, hopefully cache will keep it
	siteActorScrapeRules.Rules["fuckpassvr-native scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "realjamvr.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `div.actor-view img`, ResultType: "attr", Attribute: "src"})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "gender", Selector: `div.details div div:contains("Gender:")`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`Gender: (.*)`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "nationality", Selector: `div.details div div:contains("City and Country:")`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`City and Country:\s?(.*,)?(.*)$`, "2"}}, {Function: "Lookup Country"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "birth_date", Selector: `div.details div div:contains("Date of Birth:")`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`Date of Birth: (.*)`, "1"}},
			{Function: "Parse Date", Params: []string{"Jan. 2, 2006"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "height", Selector: `div.details div div:contains("Height:")`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(\d{2,3})\s?cm`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "weight", Selector: `div.details div div:contains("Weight:")`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(\d{2,3})\s?kg`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "eye_color", Selector: `div.details div div:contains("Eyes color:")`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`Eyes color: (.*)`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hair_color", Selector: `div.details div div:contains("Hair color:")`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`Hair color: (.*)`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "piercings", Selector: `div.details div div:contains("Piercing:")`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`Piercing:\s?([v|V]arious)?([t|T]rue)?(.*)`, "3"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "tattoos", Selector: `div.details div div:contains("Tattoo:")`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`Tattoo:\s?([v|V]arious)?([t|T]rue)?(.*)`, "3"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "biography", Selector: `div.details div div:contains("About:")`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`About: (.*)`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "waist_size", Selector: `div.details div div:contains("Waist:")`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(\d{2})`, "1"}}, {Function: "inch to cm"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hip_size", Selector: `div.details div div:contains("Hips:")`,
		PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`(\d{2})`, "1"}}, {Function: "inch to cm"}}})
	siteActorScrapeRules.Rules["realjamvr scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "povr.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `script[type="application/ld+json"]`, PostProcessing: []PostProcessing{{Function: "jsonString", Params: []string{"image"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "gender", Selector: `script[type="application/ld+json"]`, PostProcessing: []PostProcessing{{Function: "jsonString", Params: []string{"gender"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "birth_date", Selector: `script[type="application/ld+json"]`,
		PostProcessing: []PostProcessing{{Function: "jsonString", Params: []string{"birthDate"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "nationality", Selector: `script[type="application/ld+json"]`,
		PostProcessing: []PostProcessing{
			{Function: "jsonString", Params: []string{"birthPlace"}},
			{Function: "RegexString", Params: []string{`^(.*,)?\s?(.*)$`, "2"}},
			{Function: "Lookup Country"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "height", Selector: `script[type="application/ld+json"]`, PostProcessing: []PostProcessing{
		{Function: "jsonString", Params: []string{"height"}},
		{Function: "RegexString", Params: []string{`(\d{3})`, "1"}}}})
	siteActorScrapeRules.Rules["povr scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "tmwvrnet.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `div.model-page__image img`, ResultType: "attr", Attribute: "data-src", PostProcessing: []PostProcessing{{Function: "AbsoluteUrl"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "start_year", Selector: `div.model-page__information span.title:contains("Debut year:") + span.value`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hair_color", Selector: `div.model-page__information span.title:contains("Hair:") + span.value`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "eye_color", Selector: `div.model-page__information span.title:contains("Eyes:") + span.value`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "biography", Selector: `p.about`})
	siteActorScrapeRules.Rules["tmwvrnet scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "xsinsvr.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `.model-header__photo img`, ResultType: "attr", Attribute: "src", PostProcessing: []PostProcessing{{Function: "AbsoluteUrl"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "birth_date", Selector: `time`, PostProcessing: []PostProcessing{{Function: "Parse Date", Params: []string{"02/01/2006"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "band_size", Selector: `h2:contains("Measurements") + p`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`^(\d{2,3})\s?.{1,2}\s?-\s?\d{2,3}\s?-\s?\d{2,3}?`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "cup_size", Selector: `h2:contains("Measurements") + p`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`^\d{2,3}\s?(.{1,2})\s?-\s?\d{2,3}\s?-\s?\d{2,3}?`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "waist_size", Selector: `h2:contains("Measurements") + p`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`^\d{2,3}\s?.{1,2}\s?-\s?(\d{2,3})\s?-\s?\d{2,3}?`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hip_size", Selector: `h2:contains("Measurements") + p`, PostProcessing: []PostProcessing{{Function: "RegexString", Params: []string{`^\d{2,3}\s?.{1,2}\s?-\s?\d{2,3}\s?-\s?(\d{2,3})?`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "nationality", Selector: `h2:contains("Country") + p`, PostProcessing: []PostProcessing{
		{Function: "RegexString", Params: []string{`(.*)\s?(([\(|-]))`, "1"}}, // stops at - or (
		{Function: "Lookup Country"}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "weight", Selector: `h2:contains("Weight") + p`, PostProcessing: []PostProcessing{
		{Function: "RegexString", Params: []string{`(\d{2,3})\s?/`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "height", Selector: `h2:contains("Weight") + p`, PostProcessing: []PostProcessing{
		{Function: "RegexString", Params: []string{`/\s?(\d{2,3})`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "hair_color", Selector: `h2:contains("Hair ") + p`, PostProcessing: []PostProcessing{
		{Function: "RegexString", Params: []string{`^(.*)\s?\/\s?(.*)?`, "1"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "eye_color", Selector: `h2:contains("Hair ") + p`, PostProcessing: []PostProcessing{
		{Function: "RegexString", Params: []string{`^(.*)\s?\/\s?(.*)?`, "2"}}}})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "biography", ResultType: "html", Selector: `div.model-header__intro`, PostProcessing: []PostProcessing{
		{Function: "RegexString", Params: []string{`(?s)<h2>Bio<\/h2>(.*)`, "1"}}, // get everything after the H2 Bio
		{Function: "RegexReplaceAll", Params: []string{`<[^>]*>`, ``}},            // replace html tags with nothing, ie remove them
		{Function: "RegexReplaceAll", Params: []string{`^\s+|\s+$`, ``}}}})        // now remove leading & trailing whitespace
	siteActorScrapeRules.Rules["sinsvr scrape"] = siteDetails

	siteDetails = GenericScraperRuleSet{}
	siteDetails.Domain = "www.naughtyamerica.com"
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "biography", Selector: `p.bio_about_text`})
	siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "image_url", Selector: `img.performer-pic`, ResultType: "attr", Attribute: "data-src", PostProcessing: []PostProcessing{{Function: "AbsoluteUrl"}}})
	siteActorScrapeRules.Rules["naughtyamericavr scrape"] = siteDetails

	siteActorScrapeRules.GetCustomRules()
}

// Loads custom rules from actor_scrapers_examples.json
// Building custom rules for Actor scrapers is an advanced task, requiring developer or scraping skills
// Most likely to be used to post updated rules by developers, prior to an offical release
func (siteActorScrapeRules ActorScraperRules) GetCustomRules() {
	// first see if we have an example file with the builting rules
	//	this is to give examples, it is not loaded
	fName := filepath.Join(common.AppDir, "actor_scrapers_examples.json")
	out, _ := json.MarshalIndent(siteActorScrapeRules, "", "  ")
	ioutil.WriteFile(fName, out, 0644)

	// now check if the user has any custom rules
	fName = filepath.Join(common.AppDir, "actor_scrapers_custom.json")
	if _, err := os.Stat(fName); os.IsNotExist(err) {
		if _, err := os.Stat(fName); os.IsNotExist(err) {
			// create a dummy template
			exampleRules := ActorScraperRules{Rules: ActorScraperRulesMap{}}
			siteDetails := GenericScraperRuleSet{}
			siteDetails.Domain = ".com"
			siteDetails.SiteRules = append(siteDetails.SiteRules, GenericActorScraperRule{XbvrField: "", Selector: ``, ResultType: "", Attribute: "",
				PostProcessing: []PostProcessing{{Function: "", Params: []string{``}}}})
			exampleRules.Rules[" scrape"] = siteDetails
			out, _ := json.MarshalIndent(exampleRules, "", "  ")
			ioutil.WriteFile(fName, out, 0644)
		}
	} else {
		// load any custom rules and update the builtin list
		customSiteActorScrapeRules := ActorScraperRules{Rules: ActorScraperRulesMap{}}
		b, err := ioutil.ReadFile(fName)
		if err != nil {
			return
		}
		json.Unmarshal(b, &customSiteActorScrapeRules)
		for key, rule := range customSiteActorScrapeRules.Rules {
			if key != " scrape" {
				siteActorScrapeRules.Rules[key] = rule
			}
		}
	}
}
