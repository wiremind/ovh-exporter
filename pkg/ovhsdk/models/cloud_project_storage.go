package models

type StorageContainer struct {
	Name         string `json:"name"`
	ObjectsCount int64  `json:"objectsCount"`
	ObjectsSize  int64  `json:"objectsSize"`
}
