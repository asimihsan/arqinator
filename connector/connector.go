package connector

type Object interface {
	GetPath() string
}

type Connection interface {
	String() string
	GetCacheDirectory() string
	ListObjectsAsFolders(prefix string) ([]Object, error)
	ListObjectsAsAll(prefix string) ([]Object, error)
	Get(key string) (string, error)
	CachedGet(key string) (string, error)
}
