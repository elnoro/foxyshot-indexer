package captioning

type Dummy struct {
	dummyText string
}

func NewDummy(dummyText string) *Dummy {
	return &Dummy{dummyText: dummyText}
}

func (d *Dummy) Caption(string) (string, error) {
	return "", nil
}
