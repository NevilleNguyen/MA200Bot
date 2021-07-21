package storage

type KeyValueStorage interface {
	Set(key string, value interface{}) error
	Get(key string, value interface{}) error
}
