package plan

type Cache struct {
	Key       string `json:"key,omitempty"`
	Directory string `json:"directory,omitempty"`
}

func NewCache(key string, directory string) *Cache {
	return &Cache{
		Key:       key,
		Directory: directory,
	}
}
