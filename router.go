package atreugo

import (
	"net/http"
	"strings"
	"time"

	fastrouter "github.com/fasthttp/router"
	logger "github.com/savsgio/go-logger"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

var emptyFilters = Filters{}

func newRouter(log *logger.Logger) *Router {
	r := new(Router)
	r.log = log
	r.router = fastrouter.New()

	return r
}

// NewGroupPath returns a new router to group paths
func (r *Router) NewGroupPath(path string) *Router {
	g := new(Router)
	g.log = r.log
	g.router = r.router.Group(path)

	return g
}

func (r *Router) addRoute(httpMethod, url string, handler fasthttp.RequestHandler) {
	if httpMethod != strings.ToUpper(httpMethod) {
		panic("The http method '" + httpMethod + "' must be in uppercase")
	}

	r.router.Handle(httpMethod, url, handler)
}

func (r *Router) handler(viewFn View, filters Filters) fasthttp.RequestHandler {
	before := append(r.beforeMiddlewares, filters.Before...)
	after := append(filters.After, r.afterMiddlewares...)

	return func(ctx *fasthttp.RequestCtx) {
		actx := acquireRequestCtx(ctx)

		if r.log.DebugEnabled() {
			r.log.Debugf("%s %s", actx.Method(), actx.URI())
		}

		var err error
		var statusCode int

		if statusCode, err = execMiddlewares(actx, before); err == nil {
			if err = viewFn(actx); err != nil {
				statusCode = fasthttp.StatusInternalServerError
			} else {
				statusCode, err = execMiddlewares(actx, after)
			}
		}

		if err != nil {
			r.log.Error(err)
			actx.Error(err.Error(), statusCode)
		}

		releaseRequestCtx(actx)
	}
}

// UseBefore register middleware functions in the order you want to execute them before the view execution
func (r *Router) UseBefore(fns ...Middleware) {
	r.beforeMiddlewares = append(r.beforeMiddlewares, fns...)
}

// UseAfter register middleware functions in the order you want to execute them after the view execution
func (r *Router) UseAfter(fns ...Middleware) {
	r.afterMiddlewares = append(r.afterMiddlewares, fns...)
}

// Path registers a new view with the given path and method.
//
// This function is intended for bulk loading and to allow the usage of less
// frequently used, non-standardized or custom methods (e.g. for internal
// communication with a proxy).
func (r *Router) Path(httpMethod, url string, viewFn View) {
	r.PathWithFilters(httpMethod, url, viewFn, emptyFilters)
}

// PathWithFilters registers a new view with the given path and method,
// and with filters that will execute before and after
//
// This function is intended for bulk loading and to allow the usage of less
// frequently used, non-standardized or custom methods (e.g. for internal
// communication with a proxy).
func (r *Router) PathWithFilters(httpMethod, url string, viewFn View, filters Filters) {
	r.addRoute(httpMethod, url, r.handler(viewFn, filters))
}

// TimeoutPath registers a new view with the given path and method,
// which returns StatusRequestTimeout error with the given msg to the client
// if view didn't return during the given duration.
//
// The returned handler may return StatusTooManyRequests error with the given
// msg to the client if there are more than Server.Concurrency concurrent
// handlers view are running at the moment.
func (r *Router) TimeoutPath(httpMethod, url string, viewFn View, timeout time.Duration, msg string) {
	r.TimeoutPathWithFilters(httpMethod, url, viewFn, emptyFilters, timeout, msg)
}

// TimeoutPathWithFilters registers a new view with the given path and method,
// and with filters that will execute before and after, which returns StatusRequestTimeout
// error with the given msg to the client if view/filters didn't return during the given duration.
//
// The returned handler may return StatusTooManyRequests error with the given
// msg to the client if there are more than Server.Concurrency concurrent
// handlers view/filters are running at the moment.
func (r *Router) TimeoutPathWithFilters(httpMethod, url string, viewFn View, filters Filters,
	timeout time.Duration, msg string) {
	handler := r.handler(viewFn, filters)
	r.addRoute(httpMethod, url, fasthttp.TimeoutHandler(handler, timeout, msg))
}

// TimeoutWithCodePath registers a new view with the given path and method,
// which returns an error with the given msg and status code to the client
// if view/filters didn't return during the given duration.
//
// The returned handler may return StatusTooManyRequests error with the given
// msg to the client if there are more than Server.Concurrency concurrent
// handlers view/filters are running at the moment.
func (r *Router) TimeoutWithCodePath(httpMethod, url string, viewFn View,
	timeout time.Duration, msg string, statusCode int) {
	r.TimeoutWithCodePathWithFilters(httpMethod, url, viewFn, emptyFilters, timeout, msg, statusCode)
}

// TimeoutWithCodePathWithFilters registers a new view with the given path and method,
// and with filters that will execute before and after, which returns an error
// with the given msg and status code to the client if view/filters didn't return during
// the given duration.
//
// The returned handler may return StatusTooManyRequests error with the given
// msg to the client if there are more than Server.Concurrency concurrent
// handlers view/filters are running at the moment.
func (r *Router) TimeoutWithCodePathWithFilters(httpMethod, url string, viewFn View, filters Filters,
	timeout time.Duration, msg string, statusCode int) {
	handler := r.handler(viewFn, filters)
	r.addRoute(httpMethod, url, fasthttp.TimeoutWithCodeHandler(handler, timeout, msg, statusCode))
}

// NetHTTPPath wraps net/http handler to atreugo view and registers it with the given path and method
//
// While this function may be used for easy switching from net/http to fasthttp/atreugo,
// it has the following drawbacks comparing to using manually written fasthttp/atreugo,
// request handler:
//
//     * A lot of useful functionality provided by fasthttp/atreugo is missing
//       from net/http handler.
//     * net/http -> fasthttp/atreugo handler conversion has some overhead,
//       so the returned handler will be always slower than manually written
//       fasthttp/atreugo handler.
//
// So it is advisable using this function only for quick net/http -> fasthttp
// switching. Then manually convert net/http handlers to fasthttp handlers
// according to https://github.com/valyala/fasthttp#switching-from-nethttp-to-fasthttp .
func (r *Router) NetHTTPPath(httpMethod, url string, handler http.Handler) {
	r.NetHTTPPathWithFilters(httpMethod, url, handler, emptyFilters)
}

// NetHTTPPathWithFilters wraps net/http handler to atreugo view and registers it to
// the given path and method, and with filters that will execute before and after
//
// While this function may be used for easy switching from net/http to fasthttp/atreugo,
// it has the following drawbacks comparing to using manually written fasthttp/atreugo,
// request handler:
//
//     * A lot of useful functionality provided by fasthttp/atreugo is missing
//       from net/http handler.
//     * net/http -> fasthttp/atreugo handler conversion has some overhead,
//       so the returned handler will be always slower than manually written
//       fasthttp/atreugo handler.
//
// So it is advisable using this function only for quick net/http -> fasthttp
// switching. Then manually convert net/http handlers to fasthttp handlers
// according to https://github.com/valyala/fasthttp#switching-from-nethttp-to-fasthttp .
func (r *Router) NetHTTPPathWithFilters(httpMethod, url string, handler http.Handler, filters Filters) {
	h := fasthttpadaptor.NewFastHTTPHandler(handler)

	aHandler := func(ctx *RequestCtx) error {
		h(ctx.RequestCtx)
		return nil
	}

	r.addRoute(httpMethod, url, r.handler(aHandler, filters))
}

// Static serves static files from the given file system root.
//
// Make sure your program has enough 'max open files' limit aka
// 'ulimit -n' if root folder contains many files.
func (r *Router) Static(url, rootPath string) {
	if strings.HasSuffix(url, "/") {
		url = url[:len(url)-1]
	}

	r.router.ServeFiles(url+"/*filepath", rootPath)
}

// ServeFile returns HTTP response containing compressed file contents
// from the given path.
//
// HTTP response may contain uncompressed file contents in the following cases:
//
//   * Missing 'Accept-Encoding: gzip' request header.
//   * No write access to directory containing the file.
//
// Directory contents is returned if path points to directory.
func (r *Router) ServeFile(url, filePath string) {
	r.router.GET(url, func(ctx *fasthttp.RequestCtx) {
		fasthttp.ServeFile(ctx, filePath)
	})
}
