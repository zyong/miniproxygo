package http

type request struct {
	proto, method string
	path, query   string
	head, body    string
	remoteAddr    string
}
