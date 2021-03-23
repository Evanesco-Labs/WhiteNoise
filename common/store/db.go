package store

type Store interface {
	Put(key string, value interface{}) error
	Get(key string) (interface{}, error)
	Delete(key string) error
}
