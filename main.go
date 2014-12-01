package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/run/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			returnError(w, "bad request method", http.StatusBadRequest)
			return
		}
		containerName, err := getContainer(r)
		if err != nil {
			returnError(w, err.Error(), http.StatusUnauthorized)
			return
		}
		args := getArgs(r)
		if len(args) == 0 {
			root(containerName, w)
			return
		}
		switch args[0] {
		case "list":
			list(containerName, args, w)
		case "create":
			create(containerName, args, w)
		case "set":
			set(containerName, args, w)
		case "clone":
			ok, err := checkTarget(containerName, args[len(args)-2])
			if !ok {
				returnError(w, err.Error(), 403)
				return
			}
			clone(containerName, args, w)
		case "rename":
			ok, err := checkTarget(containerName, args[len(args)-2])
			if !ok {
				returnError(w, err.Error(), 403)
				return
			}
			rename(containerName, args, w)
		case "snap", "snapshot":
			ok, err := checkTarget(containerName, args[len(args)-1])
			if !ok {
				returnError(w, err.Error(), 403)
				return
			}
			snap(containerName, args, w)
		case "destroy":
			ok, err := checkTarget(containerName, args[len(args)-1])
			if !ok {
				returnError(w, err.Error(), 403)
				return
			}
			destroy(containerName, args, w)
		default:
			returnError(w, "not implemented", 404)
			return
		}

	})
	http.ListenAndServe(fmt.Sprintf(":%d", config.ServicePort), nil)
}
