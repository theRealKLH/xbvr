package models

import "github.com/avast/retry-go"

type Action struct {
	ID            uint   `gorm:"primary_key" json:"id"`

	SceneID       string `json:"scene_id"`
	ActionType    string `json:"action_type"`
	ChangedColumn string `json:"changed_column"`
	NewValue      string `json:"new_value"`
}

func (a *Action) GetIfExist(id uint) error {
	db, _ := GetDB()
	defer db.Close()

	return db.Where(&Action{ID: id}).First(a).Error
}

func (a *Action) Save() {
	db, _ := GetDB()
	defer db.Close()

	var err error
	err = retry.Do(
		func() error {
			err := db.Save(&a).Error
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

func AddAction(sceneID string, actionType string, changedColumn string, newValue string) {
	action := Action{
		SceneID: sceneID,
		ActionType: actionType,
		ChangedColumn: changedColumn,
		NewValue: newValue,
	}

	action.Save()
}
