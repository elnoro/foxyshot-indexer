package captioning

type PlaceholderCaption struct {
	text string
}

func NewPlaceholder(placeholderText string) *PlaceholderCaption {
	return &PlaceholderCaption{text: placeholderText}
}

func (d *PlaceholderCaption) Caption(string) (string, error) {
	return d.text, nil
}
