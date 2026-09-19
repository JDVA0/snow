//go:build client

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
)

func main() {
	// Start the LSP server
	cmd := exec.Command("./snow-lsp")
	
	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Printf("Failed to create stdin pipe: %v\n", err)
		os.Exit(1)
	}
	
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("Failed to create stdout pipe: %v\n", err)
		os.Exit(1)
	}
	
	if err := cmd.Start(); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		os.Exit(1)
	}
	defer cmd.Process.Kill()
	
	fmt.Println("LSP Server started")
	
	// Send initialize request
	initReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"processId": os.Getpid(),
		},
	}
	
	sendRequest(stdin, initReq)
	response := readResponse(stdout)
	fmt.Printf("Initialize response: %s\n", prettyJSON(response))
	
	// Send initialized notification
	initializedNotif := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "initialized",
	}
	sendRequest(stdin, initializedNotif)
	
	// Simulate opening a document
	didOpenParams := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri":        "file:///home/julian/Escritorio/Snow/examples/api.snow",
			"languageId": "snow",
			"version":    1,
			"text":       `using api

api.cors()

api.get("/", fn(req): api.text("Servidor Snow activo"))

fn ver_usuario(req):
    return api.json({id: req.params.id, activo: true})

api.get("/usuarios/:id", ver_usuario)

fn crear(req):
    return api.json({recibido: req.json}, 201)

api.post("/datos", crear)

api.serve(8080)`,
		},
	}
	
	didOpenReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "textDocument/didOpen",
		"params":  didOpenParams,
	}
	
	sendRequest(stdin, didOpenReq)
	fmt.Println("Document opened")
	
	// Simulate document change with invalid code
	didChangeParams := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri":     "file:///home/julian/Escritorio/Snow/examples/api.snow",
			"version": 2,
		},
		"contentChanges": []map[string]interface{}{
			{
				"text": `using api

api.cors()

api.get("/", fn(req): api.text("Servidor Snow activo"))

fn ver_usuario(req):
    return api.json({id: req.params.id, activo: true})

api.get("/usuarios/:id", ver_usuario)

fn crear(req):
    return api.json({recibido: req.json}, 201)

api.post("/datos", crear)

# This is a comment with invalid syntax
print(undefined_variable)

api.serve(8080)`,
			},
		},
	}
	
	didChangeReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "textDocument/didChange",
		"params":  didChangeParams,
	}
	
	sendRequest(stdin, didChangeReq)
	fmt.Println("Document changed")
	
	// Test completion request
	completionParams := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": "file:///home/julian/Escritorio/Snow/examples/api.snow",
		},
		"position": map[string]interface{}{
			"line":      2,
			"character": 5,
		},
		"context": map[string]interface{}{
			"triggerKind": 1,
		},
	}
	
	completionReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "textDocument/completion",
		"params":  completionParams,
	}
	
	sendRequest(stdin, completionReq)
	response = readResponse(stdout)
	fmt.Printf("Completion response: %s\n", prettyJSON(response))
	
	// Send shutdown request
	shutdownReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "shutdown",
	}
	
	sendRequest(stdin, shutdownReq)
	response = readResponse(stdout)
	fmt.Printf("Shutdown response: %s\n", prettyJSON(response))
	
	// Send exit notification
	exitNotif := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "exit",
	}
	sendRequest(stdin, exitNotif)
	
	fmt.Println("LSP Server communication completed")
}

func sendRequest(stdin io.WriteCloser, req map[string]interface{}) {
	data, err := json.Marshal(req)
	if err != nil {
		fmt.Printf("Failed to marshal request: %v\n", err)
		return
	}
	
	fmt.Printf("Sending: %s\n", prettyJSON(req))
	if _, err := stdin.Write(data); err != nil {
		fmt.Printf("Failed to write request: %v\n", err)
	}
}

func readResponse(stdout io.Reader) map[string]interface{} {
	decoder := json.NewDecoder(stdout)
	var response map[string]interface{}
	if err := decoder.Decode(&response); err != nil {
		fmt.Printf("Failed to read response: %v\n", err)
		return nil
	}
	
	return response
}

func prettyJSON(v interface{}) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(data)
}