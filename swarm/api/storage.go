package api

type Response struct {
	MimeType string
	Status   int
	Size     int64
	// Content  []byte
	Content string
}

// implements a service
type Storage struct {
	api *Api
}

func NewStorage(api *Api) *Storage {
	return &Storage{api}
}

// Put uploads the content to the swarm with a simple manifest speficying
// its content type
func (self *Storage) Put(content, contentType string) (string, error) {
	return self.api.Put(content, contentType)
}

// Get retrieves the content from bzzpath and reads the response in full
// It returns the Response object, which serialises containing the
// response body as the value of the Content field
// NOTE: if error is non-nil, sResponse may still have partial content
// the actual size of which is given in len(resp.Content), while the expected
// size is resp.Size
func (self *Storage) Get(bzzpath string) (*Response, error) {
	reader, mimeType, status, err := self.api.Get(bzzpath, true)
	if err != nil {
		return nil, err
	}
	expsize := reader.Size()
	body := make([]byte, expsize)
	size, err := reader.Read(body)
	if int64(size) == expsize {
		err = nil
	}
	return &Response{mimeType, status, expsize, string(body[:size])}, err
}

func (self *Storage) Modify(rootHash, path, contentHash, contentType string) (newRootHash string, err error) {
	return self.api.Modify(rootHash+"/"+path, contentHash, contentType, true)
}
