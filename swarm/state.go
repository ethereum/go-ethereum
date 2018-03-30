package swarm

type Voidstore struct {
}

func (self Voidstore) Load(string) ([]byte, error) {
	return nil, nil
}

func (self Voidstore) Save(string, []byte) error {
	return nil
}
