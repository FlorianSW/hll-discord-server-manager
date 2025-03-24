package resources

func NewTemplates(d string) *fileBackedStore[Template] {
	return &fileBackedStore[Template]{directory: d}
}
