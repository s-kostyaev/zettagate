package main

import (
	"net/http"
	"strings"
)

func list(container string, args []string, w http.ResponseWriter) {
	datasets := []string{}
	byDatasets := false
	argStr := strings.Join(args, " ")
	stdout, stderr, err := run(getHost(container), argStr)
	if err != nil {
		returnError(w, err.Error(), 503)
		return
	}
	out := strings.Split(strings.Trim(stdout, "\n"), "\n")
	header := strings.Fields(strings.ToLower(out[0]))
	result := []map[string]string{}
	for _, str := range out {
		m := make(map[string]string)
		rows := strings.Fields(str)
		for i, row := range header {
			m[row] = rows[i]
		}
		result = append(result, m)
	}
	if strings.Contains(argStr, "t all") || strings.Contains(argStr, "t snap") {
		stdout, _, err := run(getHost(container), "list")
		if err != nil {
			returnError(w, err.Error(), 503)
			return
		}
		out := strings.Split(strings.Trim(stdout, "\n"), "\n")
		result := []map[string]string{}
		for _, str := range out {
			m := make(map[string]string)
			rows := strings.Fields(str)
			for i, row := range header {
				m[row] = rows[i]
			}
			result = append(result, m)
		}
		matches := filterByRootFs(result, container)
		for _, match := range matches {
			datasets = append(datasets, match["name"])
		}
		byDatasets = true
	}
	data := filterByRootFs(result, container)
	if byDatasets {
		for _, d := range filterByDatasets(result, datasets) {
			data = append(data, d)
		}
	}
	returnTable(w, stderr, header, data)
}

func snap(container string, args []string, w http.ResponseWriter) {
	argStr := strings.Join(args, " ")
	stdout, stderr, err := run(getHost(container), argStr)
	if err != nil {
		returnError(w, err.Error(), 503)
		return
	}
	returnPlain(w, stdout, stderr)
}

func create(container string, args []string, w http.ResponseWriter) {
	argStr := strings.Join(setMountpoint(args, container), " ")
	stdout, stderr, err := run(getHost(container), argStr)
	if err != nil {
		returnError(w, err.Error(), 503)
		return
	}
	err, stderr = remountToContainer(container, args)
	if err != nil {
		returnError(w, "Created, but not mounted: "+err.Error()+" "+stderr, 503)
		return
	}
	returnPlain(w, stdout, stderr)
}

func destroy(container string, args []string, w http.ResponseWriter) {
	argStr := strings.Join(args, " ")
	stdout, stderr, err := run(getHost(container), argStr)
	if err != nil {
		returnError(w, err.Error(), 503)
		return
	}
	returnPlain(w, stdout, stderr)
}

func set(container string, args []string, w http.ResponseWriter) {
	option := strings.Split(args[1], "=")
	if option[0] != "mountpoint" {
		returnError(w, "Setting option "+option[0]+" forbidden", 403)
		return
	}
	cmd := ""
	if option[1] == "none" {
		cmd = "lxc-attach -e -n " + container + " -- /bin/umount " +
			args[2]
	} else {
		cmd = "/usr/bin/zfs set mountpoint=" + getRootFS(container) +
			option[1] + " " + args[2] + "; lxc-attach -e -n " +
			container + " -- /bin/mount -t zfs " + args[2] + " " +
			option[1]
	}
	stdout, stderr, err := runCmd(getHost(container), cmd)
	if err != nil {
		returnError(w, err.Error()+" "+stderr, 503)
		return
	}
	returnPlain(w, stdout, stderr)
}

func clone(container string, args []string, w http.ResponseWriter) {
	argStr := strings.Join(setMountpoint(args, container), " ")
	stdout, stderr, err := run(getHost(container), argStr)
	if err != nil {
		returnError(w, err.Error()+" "+stderr, 503)
		return
	}
	err, stderr = remountToContainer(container, args)
	if err != nil {
		returnError(w, "Created, but not mounted: "+err.Error()+" "+stderr, 503)
		return
	}
	returnPlain(w, stdout, stderr)
}

func root(container string, w http.ResponseWriter) {
	stdout, stderr, err := run(getHost(container), "")
	if err != nil {
		returnError(w, err.Error()+" "+stderr, 503)
		return
	}
	returnPlain(w, stdout, stderr)
}

func rename(container string, args []string, w http.ResponseWriter) {
	argStr := strings.Join(args, " ")
	stdout, stderr, err := run(getHost(container), argStr)
	if err != nil {
		returnError(w, err.Error()+" "+stderr, 503)
		return
	}
	returnPlain(w, stdout, stderr)
}
