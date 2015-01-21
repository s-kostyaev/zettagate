package main

import (
	"fmt"
	"net/http"
)

func main() {
	setupLogger()
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
		handleZfsClone(containerName, args, responseWriter)
	case "rename":
		handleZfsRename(containerName, args, responseWriter)
	case "snap", "snapshot":
		handleZfsSnap(containerName, args, responseWriter)
	case "destroy":
		handleZfsDestroy(containerName, args, responseWriter)
	default:
		replyJSONError(responseWriter, "not implemented", http.StatusNotFound)
		return
	}

}
