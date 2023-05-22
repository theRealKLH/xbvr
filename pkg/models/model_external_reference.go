package models

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
)

type ExternalReference struct {
	ID        uint      `gorm:"primary_key" json:"id" xbvrbackup:"-"`
	CreatedAt time.Time `json:"-" xbvrbackup:"created_at-"`
	UpdatedAt time.Time `json:"-" xbvrbackup:"updated_at"`

	XbvrLinks      []ExternalReferenceLink `json:"xbvr_links" xbvrbackup:"xbvr_links"`
	ExternalSource string                  `json:"external_source" xbvrbackup:"external_source"`
	ExternalId     string                  `json:"external_id" gorm:"index" xbvrbackup:"external_id"`
	ExternalURL    string                  `json:"external_url" gorm:"size:1000" xbvrbackup:"external_url"`
	ExternalDate   time.Time               `json:"external_date" xbvrbackup:"external_date"`
	ExternalData   string                  `json:"external_data" sql:"type:text;" xbvrbackup:"external_data"`
}
type ExternalReferenceLink struct {
	ID             uint      `gorm:"primary_key" json:"id" xbvrbackup:"-"`
	CreatedAt      time.Time `json:"-" xbvrbackup:"created_at-"`
	UpdatedAt      time.Time `json:"-" xbvrbackup:"updated_at"`
	InternalTable  string    `json:"internal_table" xbvrbackup:"internal_table"`
	InternalDbId   uint      `json:"internal_db_id" gorm:"index" xbvrbackup:"-"`
	InternalNameId string    `json:"internal_name_id" gorm:"index" xbvrbackup:"internal_name_id"`

	ExternalReferenceID uint   `json:"external_reference_id" gorm:"index" xbvrbackup:"-"`
	ExternalSource      string `json:"external_source" xbvrbackup:"-"`
	ExternalId          string `json:"external_id" gorm:"index" xbvrbackup:"-"`
	MatchType           int    `json:"match_type" xbvrbackup:"match_type"`

	ExternalReference ExternalReference `json:"external_reference" gorm:"foreignKey:ExternalReferenceId" xbvrbackup:"-"`
}

func (o *ExternalReference) GetIfExist(id uint) error {
	db, _ := GetDB()
	defer db.Close()

	return db.Preload("XbvrLinks").Where(&ExternalReference{ID: id}).First(o).Error
}

func (o *ExternalReference) FindExternalUrl(externalSource string, externalUrl string) error {
	db, _ := GetDB()
	defer db.Close()

	return db.Preload("XbvrLinks").Where(&ExternalReference{ExternalSource: externalSource, ExternalURL: externalUrl}).First(o).Error
}

func (o *ExternalReference) FindExternalId(externalSource string, externalId string) error {
	db, _ := GetDB()
	defer db.Close()

	return db.Preload("XbvrLinks").Where(&ExternalReference{ExternalSource: externalSource, ExternalId: externalId}).First(o).Error
}

func (o *ExternalReference) Save() {
	db, _ := GetDB()
	defer db.Close()

	err := retry.Do(
		func() error {
			err := db.Save(&o).Error
			if err != nil {
				return err
			}
			return nil
		},
	)

	if err != nil {
		log.Fatal("Failed to save ", err)
	}
}

func (o *ExternalReference) Delete() {
	db, _ := GetDB()
	db.Delete(&o)
	db.Close()
}

func (o *ExternalReference) AddUpdateWithUrl() {
	db, _ := GetDB()
	defer db.Close()

	existingRef := ExternalReference{ExternalSource: o.ExternalSource, ExternalURL: o.ExternalURL}
	existingRef.FindExternalUrl(o.ExternalSource, o.ExternalURL)
	if existingRef.ID > 0 {
		o.ID = existingRef.ID
		for _, oldlink := range existingRef.XbvrLinks {
			for idx, newLink := range o.XbvrLinks {
				if newLink.InternalDbId == oldlink.InternalDbId {
					o.XbvrLinks[idx].ID = oldlink.ID
				}
			}
		}
	}

	err := retry.Do(
		func() error {
			err := db.Save(&o).Error
			if err != nil {
				return err
			}
			return nil
		},
	)

	if err != nil {
		log.Fatal("Failed to save ", err)
	}
}
func (o *ExternalReference) AddUpdateWithId() {
	db, _ := GetDB()
	defer db.Close()

	existingRef := ExternalReference{ExternalSource: o.ExternalSource, ExternalId: o.ExternalId}
	existingRef.FindExternalId(o.ExternalSource, o.ExternalId)
	if existingRef.ID > 0 {
		o.ID = existingRef.ID
		for _, oldlink := range existingRef.XbvrLinks {
			for idx, newLink := range o.XbvrLinks {
				if newLink.InternalDbId == oldlink.InternalDbId {
					o.XbvrLinks[idx].ID = oldlink.ID
				}
			}
		}
	}

	err := retry.Do(
		func() error {
			err := db.Save(&o).Error
			if err != nil {
				return err
			}
			return nil
		},
	)

	if err != nil {
		log.Fatal("Failed to save ", err)
	}
}

func (o *ExternalReferenceLink) Save() {
	db, _ := GetDB()
	defer db.Close()

	err := retry.Do(
		func() error {
			err := db.Save(&o).Error
			if err != nil {
				return err
			}
			return nil
		},
	)

	if err != nil {
		log.Fatal("Failed to save ", err)
	}
}

func FormatInternalDbId(input uint) string {
	if input == 0 {
		return ""
	}
	return strconv.FormatUint(uint64(input), 10)
}
func InternalDbId2Uint(input string) uint {
	if input == "" {
		return 0
	}
	val, _ := strconv.Atoi(input)
	return uint(val)
}

func (o *ExternalReference) DetermineActorScraperByUrl(url string) string {
	url = strings.ToLower(url)
	site := url
	re := regexp.MustCompile(`^(https?:\/\/)?(www\.)?([a-zA-Z0-9\-]+)\.[a-zA-Z]{2,}(\/.*)?`)
	match := re.FindStringSubmatch(url)
	if len(match) >= 3 {
		site = match[3]
	}

	switch site {
	case "stashdb":
		return "stashdb performer"
	case "sexlikereal":
		return "slr scrape"
	case "xsinsvr":
		return "sinsvr scrape"
	case "naughtyamerica":
		return "naughtyamericavr scrape"
	case "virtualporn":
		return "bvr scrape"
	case "fuckpassvr":
		return "fuckpassvr-native scrape"
	default:
		return site + " scrape"
	}
}

func (o *ExternalReference) DetermineActorScraperBySiteId(siteId string) string {
	db, _ := GetDB()
	defer db.Close()

	var site Site
	db.Where("id = ?", siteId).First(&site)
	if site.Name == "" {
		return siteId
	}

	if strings.HasSuffix(site.Name, "POVR)") {
		return "povr scrape"
	}
	if strings.HasSuffix(site.Name, "SLR)") {
		return "slr scrape"
	}
	if strings.HasSuffix(site.Name, "VRP Hub)") {
		return "vrphub scrape"
	}
	if strings.HasSuffix(site.Name, "VRPorn)") {
		return "slr scrape"
	}
	return siteId + " scrape"
}
