package blizzard

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

func newHTTPModule(i *Interp, name string) *Module {
	m := &Module{Name: name, Dict: NewDict(8)}
	m.Dict.Set("get", Native(httpGet))
	m.Dict.Set("post", Native(httpPost))
	m.Dict.Set("put", Native(httpPut))
	m.Dict.Set("delete", Native(httpDelete))
	m.Dict.Set("patch", Native(httpPatch))
	m.Dict.Set("request", Native(httpRequest))
	return m
}

// httpGet: http.get(url)  or  http.get(url, headers_dict)
func httpGet(i *Interp, args []Val) ([]Val, error) {
	return httpDo(i, "GET", args, false)
}

// httpDelete: http.delete(url) or http.delete(url, headers_dict)
func httpDelete(i *Interp, args []Val) ([]Val, error) {
	return httpDo(i, "DELETE", args, false)
}

// httpPost: http.post(url, body_str_or_dict)  or  http.post(url, body, headers)
func httpPost(i *Interp, args []Val) ([]Val, error) {
	return httpDo(i, "POST", args, true)
}

// httpPut: http.put(url, body)
func httpPut(i *Interp, args []Val) ([]Val, error) {
	return httpDo(i, "PUT", args, true)
}

// httpPatch: http.patch(url, body)
func httpPatch(i *Interp, args []Val) ([]Val, error) {
	return httpDo(i, "PATCH", args, true)
}

// httpRequest: http.request(method, url, body?, headers?)
func httpRequest(i *Interp, args []Val) ([]Val, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("http.request expects (method, url, ...)")
	}
	method, err := strArg(args[0], "http.request")
	if err != nil {
		return nil, err
	}
	return httpDoFull(i, strings.ToUpper(string(method)), args[1:], true)
}

func httpDo(i *Interp, method string, args []Val, hasBody bool) ([]Val, error) {
	return httpDoFull(i, method, args, hasBody)
}

func httpDoFull(i *Interp, method string, args []Val, hasBody bool) ([]Val, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("http.%s expects a URL as first argument", strings.ToLower(method))
	}
	url, err := strArg(args[0], "http."+strings.ToLower(method))
	if err != nil {
		return nil, err
	}

	var bodyReader io.Reader
	bodyStr := ""
	argIdx := 1

	if hasBody && len(args) > argIdx {
		switch b := args[argIdx].(type) {
		case Str:
			bodyStr = string(b)
			bodyReader = strings.NewReader(bodyStr)
		case *Dict:
			enc, encErr := EncodeJSON(b)
			if encErr != nil {
				return nil, encErr
			}
			bodyStr = enc
			bodyReader = strings.NewReader(bodyStr)
		case NilT:
			// no body
		default:
			// Attempt str conversion
			bodyStr = BlizzardStr(args[argIdx])
			bodyReader = strings.NewReader(bodyStr)
		}
		argIdx++
	}

	req, err := http.NewRequest(method, string(url), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("http.%s: %v", strings.ToLower(method), err)
	}

	// Default content-type for bodies
	if bodyStr != "" {
		if strings.HasPrefix(bodyStr, "{") || strings.HasPrefix(bodyStr, "[") {
			req.Header.Set("Content-Type", "application/json")
		} else {
			req.Header.Set("Content-Type", "text/plain")
		}
	}

	// Apply headers dict if provided
	if len(args) > argIdx {
		if hd, ok := args[argIdx].(*Dict); ok {
			hd.ForEach(func(k string, v Val) {
				req.Header.Set(k, BlizzardStr(v))
			})
		}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http.%s %s: %v", strings.ToLower(method), string(url), err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("http.%s: reading response: %v", strings.ToLower(method), err)
	}
	respBody := string(rawBody)

	// Build response headers dict
	hdrs := NewDict(len(resp.Header))
	for k, vs := range resp.Header {
		hdrs.Set(k, Str(strings.Join(vs, ", ")))
	}

	// Build response object
	d := NewDict(6)
	d.Set("status", Int(resp.StatusCode))
	d.Set("ok", Bool(resp.StatusCode >= 200 && resp.StatusCode < 300))
	d.Set("body", Str(respBody))
	d.Set("headers", hdrs)

	// json() method — parses the body as JSON
	capturedBody := respBody
	d.Set("json", Native(func(interp *Interp, jargs []Val) ([]Val, error) {
		v, jerr := DecodeJSON(capturedBody)
		if jerr != nil {
			return nil, fmt.Errorf("resp.json(): %v", jerr)
		}
		return []Val{v}, nil
	}))

	// text() method — returns body as string
	d.Set("text", Native(func(interp *Interp, targs []Val) ([]Val, error) {
		return []Val{Str(capturedBody)}, nil
	}))

	return []Val{d}, nil
}
