package dao

import (
	"encoding/json"
	"errors"
	"fmt"
	resourcedmaster_storage "github.com/resourced/resourced-master/storage"
	"time"
)

// NewApplication is constructor for Application struct.
func NewApplication(store resourcedmaster_storage.Storer, name string) (*Application, error) {
	a := &Application{}
	a.store = store
	a.Name = name
	a.Enabled = true
	a.CreatedUnixNano = time.Now().UnixNano()
	a.Id = a.CreatedUnixNano

	return a, nil
}

// GetApplicationById returns Application struct with name as key.
func GetApplicationById(store resourcedmaster_storage.Storer, id int64) (*Application, error) {
	jsonBytes, err := store.Get(fmt.Sprintf("/applications/id/%s", id))
	if err != nil {
		return nil, err
	}

	a := &Application{}

	err = json.Unmarshal(jsonBytes, a)
	if err != nil {
		return nil, err
	}

	a.store = store

	return a, nil
}

type Application struct {
	Id              int64
	Name            string
	Enabled         bool
	CreatedUnixNano int64
	store           resourcedmaster_storage.Storer
}

// validateBeforeSave checks various conditions before saving.
func (a *Application) validateBeforeSave() error {
	if a.Id <= 0 {
		return errors.New("Id must not be empty.")
	}
	if a.Name == "" {
		return errors.New("Name must not be empty.")
	}
	return nil
}

// Save application in JSON format.
func (a *Application) Save() error {
	err := a.validateBeforeSave()
	if err != nil {
		return err
	}

	jsonBytes, err := json.Marshal(a)
	if err != nil {
		return err
	}

	err = CommonSaveById(a.store, "applications", a.Id, jsonBytes)
	if err != nil {
		return err
	}

	return nil
}

func (a *Application) Delete() error {
	return CommonDeleteById(a.store, "applications", a.Id)
}
