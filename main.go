package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/run/", runHandler)
	http.ListenAndServe(fmt.Sprintf(":%d", config.ServicePort), nil)
}

func runHandler(responseWriter http.ResponseWriter, request *http.Request) {
	if request.Method != "POST" {
		replyJSONError(responseWriter, "only POST requests allowed",
			http.StatusBadRequest)
		return
	}
	containerName, err := getContainerName(request)
	if err != nil {
		replyJSONError(responseWriter, err.Error(), http.StatusUnauthorized)
		return
	}
	args := getCommandLineFromRequest(request)
	if len(args) == 0 {
		handleUsage(containerName, responseWriter)
		return
	}
	switch args[0] {
	case "list":
		handleZfsList(containerName, args, responseWriter)
	case "create":
		handleZfsCreate(containerName, args, responseWriter)
	case "set":
		handleZfsSet(containerName, args, responseWriter)
	case "clone":
		err := hasPermissionsZfs(containerName, args[len(args)-2])
		if err != nil {
			replyJSONError(responseWriter, err.Error(), http.StatusForbidden)
			return
		}
		handleZfsClone(containerName, args, responseWriter)
	case "rename":
		err := hasPermissionsZfs(containerName, args[len(args)-2])
		if err != nil {
			replyJSONError(responseWriter, err.Error(), http.StatusForbidden)
			return
		}
		handleZfsRename(containerName, args, responseWriter)
	case "snap", "snapshot":
		err := hasPermissionsZfs(containerName, args[len(args)-1])
		if err != nil {
			replyJSONError(responseWriter, err.Error(), http.StatusForbidden)
			return
		}
		handleZfsSnap(containerName, args, responseWriter)
	case "destroy":
		err := hasPermissionsZfs(containerName, args[len(args)-1])
		if err != nil {
			replyJSONError(responseWriter, err.Error(), http.StatusForbidden)
			return
		}
		handleZfsDestroy(containerName, args, responseWriter)
	default:
		replyJSONError(responseWriter, "not implemented", http.StatusNotFound)
		return
	}

}
