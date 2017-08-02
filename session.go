package cartsess

type Options struct {
	Path   string
	Domain string
	MaxAge   int
	Secure   bool
	HttpOnly bool
}

type Session struct {
	ID string
	Values  map[interface{}]interface{}
	Options *Options
	IsNew   bool
	store   Store
	name    string
}
