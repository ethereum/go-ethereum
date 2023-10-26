package discover

type Storage interface {
	ContentId(contentKey []byte) []byte

	Get(contentKey []byte, contentId []byte) ([]byte, error)

	Put(contentKey []byte, content []byte) error
}
