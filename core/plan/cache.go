package plan

type Cache struct {
	Directory string `json:"directory,omitempty"`
}

func NewCache(directory string) *Cache {
	return &Cache{
		Directory: directory,
	}
}
