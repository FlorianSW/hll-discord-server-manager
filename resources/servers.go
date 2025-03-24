package resources

func NewServers(d string) *fileBackedStore[Server] {
	return &fileBackedStore[Server]{directory: d}
}
