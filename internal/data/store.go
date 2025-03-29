package data

/*

A Store is a place where we store Documents.

After creation, a Store must be Opened or Created before any other methods are invoked

*/

type DocReference struct {
	ID   int
	Name string
}

type Store interface {
	Open(filepath string) error

	Create(filepath string) error

	ListDocuments(t bool) ([]DocReference, error)

	CreateDocument(name string, text string) (int64, error)

	SaveDocument(key string, text string) error

	LoadDocument(key string) (string, error)

	TrashDocument(key string) error

	DeleteDocument(key string) error

	RestoreDocument(key string) error

	RenameDocument(key string) error

	DuplicateDocument(key string) error

	LastOpened() (string, error)
}
