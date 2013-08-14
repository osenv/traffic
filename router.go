package traffic

import (
  "net/http"
  "fmt"
)

type RequestLogFunc func(int, *http.Request)

type HttpMethod string

type BeforeFilterFunc func(http.ResponseWriter, *http.Request) bool

type Router struct {
  routes map[HttpMethod][]*Route
  NotFoundHandler HttpHandleFunc
  beforeFilters []BeforeFilterFunc
  RequestLogFunc RequestLogFunc
}

func (router *Router) Add(method HttpMethod, path string, handler HttpHandleFunc) *Route {
  route := NewRoute(path, handler)
  router.addRoute(method, route)

  return route
}

func (router *Router) addRoute(method HttpMethod, route *Route) {
  router.routes[method] = append(router.routes[method], route)
}

func (router *Router) Get(path string, handler HttpHandleFunc) *Route {
  route := router.Add(HttpMethod("GET"), path, handler)
  router.addRoute(HttpMethod("HEAD"), route)

  return route
}

func (router *Router) Post(path string, handler HttpHandleFunc) *Route {
  return router.Add(HttpMethod("POST"), path, handler)
}

func (router *Router) Delete(path string, handler HttpHandleFunc) *Route {
  return router.Add(HttpMethod("DELETE"), path, handler)
}

func (router *Router) Put(path string, handler HttpHandleFunc) *Route {
  return router.Add(HttpMethod("PUT"), path, handler)
}

func (router *Router) Patch(path string, handler HttpHandleFunc) *Route {
  return router.Add(HttpMethod("PATCH"), path, handler)
}

func (router *Router) AddBeforeFilter(beforeFilter BeforeFilterFunc) {
  router.beforeFilters = append(router.beforeFilters, beforeFilter)
}

type LoggedResponseWriter struct {
  http.ResponseWriter
  request *http.Request
  log RequestLogFunc
  statusCode int
}

func (loggedResponseWriter *LoggedResponseWriter) WriteHeader(statusCode int) {
  loggedResponseWriter.statusCode = statusCode
  loggedResponseWriter.ResponseWriter.WriteHeader(statusCode)
}

func (loggedResponseWriter LoggedResponseWriter) Flush() {
  statusCode := loggedResponseWriter.statusCode
  if statusCode == 0 {
    statusCode = http.StatusOK
  }
  loggedResponseWriter.log(statusCode, loggedResponseWriter.request)
}

func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
  lrw := &LoggedResponseWriter{
    ResponseWriter: w,
    request: r,
    log: router.RequestLogFunc,
  }

  for _, route := range router.routes[HttpMethod(r.Method)] {
    values, ok := route.Match(r.URL.Path)
    if ok {
      newValues := r.URL.Query()
      for k, v := range values {
        newValues[k] = v
      }

      r.URL.RawQuery = newValues.Encode()

      continueAfterBeforeFilter := true

      for _, beforeFilter := range router.beforeFilters {
        continueAfterBeforeFilter = beforeFilter(lrw, r)
        if !continueAfterBeforeFilter {
          break
        }
      }

      if continueAfterBeforeFilter {
        route.Handler(lrw, r)
      }

      lrw.Flush()
      return
    }
  }

  if router.NotFoundHandler != nil {
    router.NotFoundHandler(lrw, r)
  } else {
    http.Error(lrw, "404 page not found", http.StatusNotFound)
  }

  lrw.Flush()
}

func (router Router) defaultRequestLogFunc(statusCode int, r *http.Request) {
  fmt.Printf("%d - %s\n", statusCode, r.URL)
}

func New() *Router {
  router := &Router{}
  router.routes = make(map[HttpMethod][]*Route)
  router.beforeFilters = make([]BeforeFilterFunc, 0)
  router.RequestLogFunc = router.defaultRequestLogFunc
  return router
}

