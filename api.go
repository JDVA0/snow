package snow

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Resp is an API response value built by the snow.api helpers.
type Resp struct {
	Status  int
	CType   string
	Body    string
	Headers map[string]string
}

// APIServer holds the state of the snow.api HTTP server.
type APIServer struct {
	i           *Interp
	mu          sync.Mutex
	port        int
	routes      []*apiRoute
	middlewares []Val
	statics     []*apiStatic
}

type apiStatic struct {
	prefix string
	dir    string
}

type apiRoute struct {
	method  string
	segs    []string
	handler Val
}

func newAPIModule(i *Interp, name string) *Module {
	s := &APIServer{i: i, port: 8080}
	i.api = s
	m := &Module{Name: name, Dict: NewDict(20)}
	m.Dict.Set("listen", Native(apiListen))
	m.Dict.Set("route", Native(apiRouteAdd))
	m.Dict.Set("get", Native(apiMethodRoute("GET")))
	m.Dict.Set("post", Native(apiMethodRoute("POST")))
	m.Dict.Set("put", Native(apiMethodRoute("PUT")))
	m.Dict.Set("delete", Native(apiMethodRoute("DELETE")))
	m.Dict.Set("patch", Native(apiMethodRoute("PATCH")))
	m.Dict.Set("all", Native(apiMethodRoute("*")))
	m.Dict.Set("use", Native(apiUse))
	m.Dict.Set("static", Native(apiStaticDir))
	m.Dict.Set("cors", Native(apiCORS))
	m.Dict.Set("serve", Native(apiServe))
	m.Dict.Set("response", Native(apiResponse))
	m.Dict.Set("json", Native(apiJSON))
	m.Dict.Set("text", Native(apiText))
	m.Dict.Set("html", Native(apiHTML))
	m.Dict.Set("redirect", Native(apiRedirect))
	m.Dict.Set("status", Native(apiStatusResp))
	return m
}

func apiListen(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "listen", 1); err != nil {
		return nil, err
	}
	v, ok := args[0].(Int)
	if !ok {
		return nil, fmt.Errorf("listen expects a port int, got %s", TypeName(args[0]))
	}
	if i.api == nil {
		newAPIModule(i, "api")
	}
	i.api.port = int(v)
	return []Val{}, nil
}

func apiMethodRoute(method string) Native {
	return func(i *Interp, args []Val) ([]Val, error) {
		if err := want(args, strings.ToLower(method), 2); err != nil {
			return nil, err
		}
		path, err := strArg(args[0], strings.ToLower(method))
		if err != nil {
			return nil, err
		}
		switch args[1].(type) {
		case *Fn, Native:
		default:
			return nil, fmt.Errorf("%s expects a handler function, got %s", strings.ToLower(method), TypeName(args[1]))
		}
		if i.api == nil {
			newAPIModule(i, "api")
		}
		segs := strings.Split(strings.Trim(string(path), "/"), "/")
		i.api.routes = append(i.api.routes, &apiRoute{method: method, segs: segs, handler: args[1]})
		return []Val{}, nil
	}
}

func apiRouteAdd(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "route", 3); err != nil {
		return nil, err
	}
	meth, err := strArg(args[0], "route")
	if err != nil {
		return nil, err
	}
	path, err := strArg(args[1], "route")
	if err != nil {
		return nil, err
	}
	switch args[2].(type) {
	case *Fn, Native:
	default:
		return nil, fmt.Errorf("route expects a handler function, got %s", TypeName(args[2]))
	}
	if i.api == nil {
		newAPIModule(i, "api")
	}
	method := strings.ToUpper(string(meth))
	segs := strings.Split(strings.Trim(string(path), "/"), "/")
	i.api.routes = append(i.api.routes, &apiRoute{method: method, segs: segs, handler: args[2]})
	return []Val{}, nil
}

func apiUse(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "use", 1); err != nil {
		return nil, err
	}
	switch args[0].(type) {
	case *Fn, Native:
	default:
		return nil, fmt.Errorf("use expects a middleware function, got %s", TypeName(args[0]))
	}
	if i.api == nil {
		newAPIModule(i, "api")
	}
	i.api.middlewares = append(i.api.middlewares, args[0])
	return []Val{}, nil
}

func apiStaticDir(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "static", 2); err != nil {
		return nil, err
	}
	prefix, err := strArg(args[0], "static")
	if err != nil {
		return nil, err
	}
	dir, err := strArg(args[1], "static")
	if err != nil {
		return nil, err
	}
	if i.api == nil {
		newAPIModule(i, "api")
	}
	cleanPrefix := "/" + strings.Trim(string(prefix), "/")
	i.api.statics = append(i.api.statics, &apiStatic{prefix: cleanPrefix, dir: string(dir)})
	return []Val{}, nil
}

func apiCORS(i *Interp, args []Val) ([]Val, error) {
	corsMiddleware := Native(func(interp *Interp, mwArgs []Val) ([]Val, error) {
		if len(mwArgs) == 0 {
			return nil, nil
		}
		req, ok := mwArgs[0].(*Dict)
		if !ok {
			return nil, nil
		}
		if meth, ok := req.Get("method"); ok && SnowStr(meth) == "OPTIONS" {
			res := &Resp{
				Status: http.StatusNoContent,
				Headers: map[string]string{
					"Access-Control-Allow-Origin":  "*",
					"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, PATCH, OPTIONS",
					"Access-Control-Allow-Headers": "*",
				},
			}
			return []Val{res}, nil
		}
		return nil, nil
	})
	if i.api == nil {
		newAPIModule(i, "api")
	}
	i.api.middlewares = append(i.api.middlewares, corsMiddleware)
	return []Val{}, nil
}

func apiServe(i *Interp, args []Val) ([]Val, error) {
	if i.api == nil {
		newAPIModule(i, "api")
	}
	if len(args) == 1 {
		if portInt, ok := args[0].(Int); ok {
			i.api.port = int(portInt)
		} else {
			return nil, fmt.Errorf("serve port must be an int, got %s", TypeName(args[0]))
		}
	} else if len(args) > 1 {
		return nil, fmt.Errorf("serve expects at most 1 argument (port)")
	}
	s := i.api
	srv := &http.Server{Addr: fmt.Sprintf(":%d", s.port), Handler: http.HandlerFunc(s.handle)}
	fmt.Fprintf(i.out, "\033[36;1m❄ snow\033[0m listening on \033[4mhttp://0.0.0.0:%d\033[0m\n", s.port)
	return nil, srv.ListenAndServe()
}

func apiResponse(i *Interp, args []Val) ([]Val, error) {
	if len(args) < 3 {
		return nil, fmt.Errorf("response expects at least 3 arguments (status, ctype, body)")
	}
	status, ok := args[0].(Int)
	if !ok {
		return nil, fmt.Errorf("response expects an int status, got %s", TypeName(args[0]))
	}
	ct, err := strArg(args[1], "response")
	if err != nil {
		return nil, err
	}
	body := SnowStr(args[2])
	res := &Resp{Status: int(status), CType: string(ct), Body: body}
	if len(args) >= 4 {
		if hd, ok := args[3].(*Dict); ok {
			res.Headers = dictToHeaders(hd)
		}
	}
	return []Val{res}, nil
}

func apiJSON(i *Interp, args []Val) ([]Val, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("json expects at least 1 argument")
	}
	status := 200
	if len(args) >= 2 {
		if st, ok := args[1].(Int); ok {
			status = int(st)
		}
	}
	s, err := EncodeJSON(args[0])
	if err != nil {
		return nil, err
	}
	res := &Resp{Status: status, CType: "application/json", Body: s}
	if len(args) >= 3 {
		if hd, ok := args[2].(*Dict); ok {
			res.Headers = dictToHeaders(hd)
		}
	}
	return []Val{res}, nil
}

func apiText(i *Interp, args []Val) ([]Val, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("text expects at least 1 argument")
	}
	status := 200
	if len(args) >= 2 {
		if st, ok := args[1].(Int); ok {
			status = int(st)
		}
	}
	res := &Resp{Status: status, CType: "text/plain; charset=utf-8", Body: SnowStr(args[0])}
	if len(args) >= 3 {
		if hd, ok := args[2].(*Dict); ok {
			res.Headers = dictToHeaders(hd)
		}
	}
	return []Val{res}, nil
}

func apiHTML(i *Interp, args []Val) ([]Val, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("html expects at least 1 argument")
	}
	status := 200
	if len(args) >= 2 {
		if st, ok := args[1].(Int); ok {
			status = int(st)
		}
	}
	res := &Resp{Status: status, CType: "text/html; charset=utf-8", Body: SnowStr(args[0])}
	if len(args) >= 3 {
		if hd, ok := args[2].(*Dict); ok {
			res.Headers = dictToHeaders(hd)
		}
	}
	return []Val{res}, nil
}

func apiRedirect(i *Interp, args []Val) ([]Val, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("redirect expects a destination url")
	}
	url, err := strArg(args[0], "redirect")
	if err != nil {
		return nil, err
	}
	status := 302
	if len(args) >= 2 {
		if st, ok := args[1].(Int); ok {
			status = int(st)
		}
	}
	res := &Resp{
		Status: status,
		Headers: map[string]string{
			"Location": string(url),
		},
	}
	return []Val{res}, nil
}

func apiStatusResp(i *Interp, args []Val) ([]Val, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("status expects a status code")
	}
	st, ok := args[0].(Int)
	if !ok {
		return nil, fmt.Errorf("status code must be an int")
	}
	body := ""
	if len(args) >= 2 {
		body = SnowStr(args[1])
	}
	res := &Resp{Status: int(st), CType: "text/plain; charset=utf-8", Body: body}
	return []Val{res}, nil
}

func dictToHeaders(d *Dict) map[string]string {
	h := make(map[string]string, len(d.keys))
	for _, k := range d.keys {
		if v, ok := d.Get(k); ok {
			h[k] = SnowStr(v)
		}
	}
	return h
}

func (s *APIServer) handle(w http.ResponseWriter, r *http.Request) {
	// 1. Check static file routes
	for _, st := range s.statics {
		if strings.HasPrefix(r.URL.Path, st.prefix) {
			rel := strings.TrimPrefix(r.URL.Path, st.prefix)
			filePath := filepath.Join(st.dir, rel)
			if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
				http.ServeFile(w, r, filePath)
				return
			}
		}
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	req := NewDict(9)
	req.Set("method", Str(r.Method))
	req.Set("path", Str(r.URL.Path))
	req.Set("query", queryDict(r.URL.Query()))
	req.Set("headers", headerDict(r.Header))
	req.Set("body", Str(string(body)))
	req.Set("remote", Str(r.RemoteAddr))
	req.Set("content_type", Str(r.Header.Get("Content-Type")))

	// Parse JSON automatically if available
	if len(body) > 0 {
		trimmedBody := strings.TrimSpace(string(body))
		if strings.HasPrefix(trimmedBody, "{") || strings.HasPrefix(trimmedBody, "[") {
			if decoded, jsonErr := DecodeJSON(trimmedBody); jsonErr == nil {
				req.Set("json", decoded)
			} else {
				req.Set("json", Nil)
			}
		} else {
			req.Set("json", Nil)
		}
	} else {
		req.Set("json", Nil)
	}

	// 2. Run middlewares
	for _, mw := range s.middlewares {
		s.mu.Lock()
		res, mwErr := s.i.invoke(mw, []Val{req})
		s.mu.Unlock()
		if mwErr != nil {
			http.Error(w, mwErr.Error(), http.StatusInternalServerError)
			return
		}
		if len(res) > 0 && res[0] != Nil {
			if err := writeReply(w, res); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
	}

	// 3. Match route
	rt, params := s.match(r.Method, r.URL.Path)
	if rt == nil {
		http.Error(w, "404 page not found", http.StatusNotFound)
		return
	}
	req.Set("params", paramsDict(params))

	s.mu.Lock()
	vals, err := s.i.invoke(rt.handler, []Val{req})
	s.mu.Unlock()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := writeReply(w, vals); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *APIServer) match(method, path string) (*apiRoute, map[string]string) {
	ps := strings.Split(strings.Trim(path, "/"), "/")
	if path == "/" {
		ps = []string{""}
	}
	for _, rt := range s.routes {
		if rt.method != "*" && rt.method != method && !(rt.method == "GET" && method == "HEAD") {
			continue
		}
		if len(rt.segs) != len(ps) {
			continue
		}
		params := map[string]string{}
		ok := true
		for k, seg := range rt.segs {
			if strings.HasPrefix(seg, ":") {
				params[strings.TrimPrefix(seg, ":")] = ps[k]
			} else if seg != ps[k] {
				ok = false
				break
			}
		}
		if ok {
			return rt, params
		}
	}
	return nil, nil
}

func queryDict(q map[string][]string) *Dict {
	d := NewDict(len(q))
	for k, vs := range q {
		if len(vs) > 0 {
			d.Set(k, Str(vs[0]))
		}
	}
	return d
}

func headerDict(h http.Header) *Dict {
	d := NewDict(len(h))
	for k, vs := range h {
		if len(vs) > 0 {
			d.Set(strings.ToLower(k), Str(vs[0]))
		}
	}
	return d
}

func paramsDict(p map[string]string) *Dict {
	d := NewDict(len(p))
	for k, v := range p {
		d.Set(k, Str(v))
	}
	return d
}

func writeReply(w http.ResponseWriter, vals []Val) error {
	if len(vals) == 0 {
		w.WriteHeader(http.StatusOK)
		return nil
	}
	switch b := vals[0].(type) {
	case *Resp:
		for hk, hv := range b.Headers {
			w.Header().Set(hk, hv)
		}
		if b.CType != "" {
			w.Header().Set("Content-Type", b.CType)
		}
		w.WriteHeader(b.Status)
		if b.Body != "" && b.Status >= 200 {
			fmt.Fprint(w, b.Body)
		}
	case Str:
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, string(b))
	case NilT:
		w.WriteHeader(http.StatusOK)
	case Int, Float, Bool, *Dict, List:
		s, err := EncodeJSON(b)
		if err != nil {
			return err
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, s)
	default:
		return fmt.Errorf("handler must return a string, list, dict or response; got %s", TypeName(b))
	}
	return nil
}
