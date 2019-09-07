package xbvr

import (
	"os"
	"time"
)

type Volume struct {
	ID        uint      `gorm:"primary_key" json:"id"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`

	Path           string    `json:"path"`
	LastScan       time.Time `json:"last_scan"`
	IsEnabled      bool      `json:"-"`
	IsAvailable    bool      `json:"is_available"`
	FileCount      int       `gorm:"-" json:"file_count"`
	UnmatchedCount int       `gorm:"-" json:"unmatched_count"`
	TotalSize      int64     `gorm:"-" json:"total_size"`
}

func (o *Volume) IsMounted() bool {
	if _, err := os.Stat(o.Path); os.IsNotExist(err) {
		return false
	}
	return true
}

func (o *Volume) Save() error {
	db, _ := GetDB()
	err := db.Save(o).Error
	db.Close()
	return err
}

func (o *Volume) Files() []File {
	var allFiles []File
	db, _ := GetDB()
	db.Where("path LIKE ?", o.Path+"%").Find(&allFiles)
	db.Close()
	return allFiles
}

func CheckVolumes() {
	db, _ := GetDB()
	defer db.Close()

	var vol []Volume
	db.Find(&vol)

	for i := range vol {
		vol[i].IsAvailable = vol[i].IsMounted()
		vol[i].Save()
	}
}
