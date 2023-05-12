package models

import (
	"encoding/json"
	"time"

	"github.com/avast/retry-go/v4"
)

type Actor struct {
	ID        uint      `gorm:"primary_key" json:"id" xbvrbackup:"-"`
	CreatedAt time.Time `json:"-" xbvrbackup:"-"`
	UpdatedAt time.Time `json:"-" xbvrbackup:"-"`

	Name   string  `gorm:"unique_index" json:"name" xbvrbackup:"name"`
	Scenes []Scene `gorm:"many2many:scene_cast;" json:"-" xbvrbackup:"-"`
	Count  int     `json:"count" xbvrbackup:"-"`

	AvailCount int `json:"avail_count" xbvrbackup:"-"`

	ImageUrl   string  `json:"image_url" xbvrbackup:"image_url"`
	ImageArr   string  `json:"image_arr" sql:"type:text;" xbvrbackup:"image_arr"`
	StarRating float64 `json:"star_rating" xbvrbackup:"star_rating"`
	Favourite  bool    `json:"favourite" gorm:"default:false" xbvrbackup:"favourite"`
	Watchlist  bool    `json:"watchlist" gorm:"default:false" xbvrbackup:"watchlist"`

	BirthDate   time.Time `json:"birth_date" xbvrbackup:"birth_date"`
	Nationality string    `json:"nationality" xbvrbackup:"nationality"`
	Ethnicity   string    `json:"ethnicity" xbvrbackup:"ethnicity"`
	EyeColor    string    `json:"eye_color" xbvrbackup:"eyeColor"`
	HairColor   string    `json:"hair_color" xbvrbackup:"hairColor"`
	Height      int       `json:"height" xbvrbackup:"height"`
	Weight      int       `json:"weight" xbvrbackup:"weight"`
	CupSize     string    `json:"cup_size" xbvrbackup:"cup_size"`
	BandSize    int       `json:"band_size" xbvrbackup:"band_size"`
	WaistSize   int       `json:"waist_size" xbvrbackup:"waist_size"`
	HipSize     int       `json:"hip_size" xbvrbackup:"hip_size"`
	BreastType  string    `json:"breast_type" xbvrbackup:"breast_type"`
	StartYear   int       `json:"start_year" xbvrbackup:"start_year"`
	EndYear     int       `json:"end_year" xbvrbackup:"end_year"`
	Tattoos     string    `json:"tattoos" sql:"type:text;"  xbvrbackup:"tattoos"`
	Piercings   string    `json:"piercings" sql:"type:text;" xbvrbackup:"biercings"`

	Biography string `json:"biography" sql:"type:text;" xbvrbackup:"biography"`
	Aliases   string `json:"aliases" gorm:"size:1000"  xbvrbackup:"aliases"`
	Gender    string `json:"gender" xbvrbackup:"gender"`
	URLs      string `json:"urls" sql:"type:text;" xbvrbackup:"urls"`
}

type ActorLink struct {
	Url  string `json:"url"`
	Type string `json:"type"`
}

func (i *Actor) Save() error {
	db, _ := GetDB()
	defer db.Close()

	var err error
	err = retry.Do(
		func() error {
			err := db.Save(&i).Error
			if err != nil {
				return err
			}
			return nil
		},
	)

	if err != nil {
		log.Fatal("Failed to save ", err)
	}

	return nil
}

func (i *Actor) CountActorTags() {
	db, _ := GetDB()
	defer db.Close()

	type CountResults struct {
		ID            int
		Cnt           int
		Existingcnt   int
		IsAvailable   int
		Existingavail int
	}

	var results []CountResults

	db.Model(&Actor{}).
		Select("actors.id, count as existingcnt, count(*) cnt, sum(scenes.is_available ) is_available, avail_count as existingavail").
		Group("actors.id").
		Joins("join scene_cast on scene_cast.actor_id = actors.id").
		Joins("join scenes on scenes.id=scene_cast.scene_id and scenes.deleted_at is null").
		Scan(&results)

	for i := range results {
		var actor Actor
		if results[i].Cnt != results[i].Existingcnt || results[i].IsAvailable != results[i].Existingavail {
			db.First(&actor, results[i].ID)
			actor.Count = results[i].Cnt
			actor.AvailCount = results[i].IsAvailable
			actor.Save()
		}
	}
}

func (i *Actor) AddToImageArray(newValue string) bool {
	var array []string
	if newValue == "" {
		return false
	}
	if i.ImageArr == "" {
		i.ImageArr = "[]"
	}

	json.Unmarshal([]byte(i.ImageArr), &array)
	for idx, item := range array {
		if item == newValue {
			// if we are adding an image that is the main actor image, put it at the begining
			if newValue == i.ImageUrl && idx > 0 {
				array = append(array[:idx], array[idx+1:]...)
				array = append([]string{item}, array...)
				jsonString, _ := json.Marshal(array)
				i.ImageArr = string(jsonString)
			}
			return false
		}
	}
	if newValue == i.ImageUrl {
		array = append([]string{newValue}, array...)
	} else {
		array = append(array, newValue)
	}
	jsonString, _ := json.Marshal(array)
	i.ImageArr = string(jsonString)
	return true
}
func (i *Actor) AddToActorUrlArray(newValue ActorLink) bool {
	if newValue.Url == "" {
		return false
	}
	var array []ActorLink
	if i.URLs == "" {
		i.URLs = "[]"
	}
	json.Unmarshal([]byte(i.URLs), &array)
	for _, item := range array {
		if item.Url == newValue.Url {
			return false
		}
	}
	array = append(array, newValue)
	jsonString, _ := json.Marshal(array)
	i.URLs = string(jsonString)
	return true
}
func (i *Actor) AddToTattoos(newValue string) bool {
	updated := false
	if newValue != "" {
		i.Tattoos, updated = addToStringArray(i.Tattoos, newValue)
	}
	return updated
}
func (i *Actor) AddToPiercings(newValue string) bool {
	updated := false
	if newValue != "" {
		i.Piercings, updated = addToStringArray(i.Piercings, newValue)
	}
	return updated
}
func (i *Actor) AddToAliases(newValue string) bool {
	updated := false
	if newValue != "" {
		i.Aliases, updated = addToStringArray(i.Aliases, newValue)
	}
	return updated
}
func addToStringArray(inputArray string, newValue string) (string, bool) {
	var array []string
	if inputArray == "" {
		inputArray = "[]"
	}
	json.Unmarshal([]byte(inputArray), &array)
	for _, item := range array {
		if item == newValue {
			return inputArray, false
		}
	}
	array = append(array, newValue)
	jsonString, _ := json.Marshal(array)
	return string(jsonString), true
}
