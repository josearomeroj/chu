package chu

func (r *Router) Method(method, pattern string, h Handler) {
	r.chi.Method(method, pattern, r.adapt(h))
}

func (r *Router) Get(pattern string, h Handler) {
	r.chi.Get(pattern, r.adapt(h))
}

func (r *Router) Post(pattern string, h Handler) {
	r.chi.Post(pattern, r.adapt(h))
}

func (r *Router) Put(pattern string, h Handler) {
	r.chi.Put(pattern, r.adapt(h))
}

func (r *Router) Delete(pattern string, h Handler) {
	r.chi.Delete(pattern, r.adapt(h))
}

func (r *Router) Patch(pattern string, h Handler) {
	r.chi.Patch(pattern, r.adapt(h))
}

func (r *Router) Head(pattern string, h Handler) {
	r.chi.Head(pattern, r.adapt(h))
}

func (r *Router) Options(pattern string, h Handler) {
	r.chi.Options(pattern, r.adapt(h))
}

func (r *Router) Connect(pattern string, h Handler) {
	r.chi.Connect(pattern, r.adapt(h))
}

func (r *Router) Trace(pattern string, h Handler) {
	r.chi.Trace(pattern, r.adapt(h))
}
