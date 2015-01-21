package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/theairkit/runcmd"
)

type Report map[string]Host

type Host struct {
	Hostname   string
	NetAddr    []string
	Pools      []string
	Containers map[string]Container
}

type Container struct {
	Name   string
	Host   string
	Status string
	Image  string
	Ips    map[string][]string
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type TableMessage struct {
	Stderr []string    `json:"stderr"`
	Stdout TableStdout `json:"stdout"`
}

type TableStdout struct {
	Format string              `json:"format"`
	Header []string            `json:"header"`
	Data   []map[string]string `json:"data"`
}

type PlainMessage struct {
	Stderr []string  `json:"stderr"`
	Stdout PlainData `json:"stdout"`
}

type PlainData struct {
	Data []string `json:"data"`
}

func handleZfsList(container string, args []string,
	responseWriter http.ResponseWriter) {
	stdout, stderr, err := runZfsCmd(getHost(container), []string{"list", "-t",
		"all"})
	if err != nil {
		replyJSONError(responseWriter, err.Error(),
			http.StatusInternalServerError)
		return
	}

	header, table := createTableFromString(stdout)
	data := filterByRootFs(table, container)

	switch getListType(args) {
	case "all":
		datasets := []string{}
		for _, match := range data {
			datasets = append(datasets, match["name"])
		}

		for _, d := range filterByDatasets(table, datasets) {
			data = append(data, d)
		}

		replyTable(responseWriter, stderr, header, data)

	case "snap", "snapshot":
		datasets := []string{}
		for _, match := range data {
			datasets = append(datasets, match["name"])
		}

		data = []map[string]string{}
		for _, d := range filterByDatasets(table, datasets) {
			data = append(data, d)
		}

		replyTable(responseWriter, stderr, header, data)

	case "filesystem":
		replyTable(responseWriter, stderr, header, data)

	default:
		replyJSONError(responseWriter, "this type is not implemented",
			http.StatusBadRequest)

	}

}

func handleZfsSnap(container string, args []string,
	responseWriter http.ResponseWriter) {
	err := hasPermissionsZfs(container, args[len(args)-1])
	if err != nil {
		replyJSONError(responseWriter, err.Error(), http.StatusForbidden)
		return
	}

	stdout, stderr, err := runZfsCmd(getHost(container), args)
	if err != nil {
		replyJSONError(responseWriter, err.Error(),
			http.StatusInternalServerError)
		return
	}

	replyPlain(responseWriter, stdout, stderr)
}

func handleZfsCreate(container string, args []string,
	responseWriter http.ResponseWriter) {
	stdout, stderr, err := runZfsCmd(getHost(container), setMountpoint(args,
		container))
	if err != nil {
		replyJSONError(responseWriter, err.Error(),
			http.StatusInternalServerError)
		return
	}

	err, stderr = remountToContainer(container, args)
	if err != nil {
		replyJSONError(responseWriter, "Created, but not mounted: "+err.Error()+
			" "+stderr, http.StatusInternalServerError)
		return
	}

	replyPlain(responseWriter, stdout, stderr)
}

func handleZfsDestroy(container string, args []string,
	responseWriter http.ResponseWriter) {
	err := hasPermissionsZfs(container, args[len(args)-1])
	if err != nil {
		replyJSONError(responseWriter, err.Error(), http.StatusForbidden)
		return
	}

	stdout, stderr, err := runZfsCmd(getHost(container), args)
	if err != nil {
		replyJSONError(responseWriter, err.Error(),
			http.StatusInternalServerError)
		return
	}

	replyPlain(responseWriter, stdout, stderr)
}

func handleZfsSet(container string, args []string,
	responseWriter http.ResponseWriter) {
	option := strings.Split(args[1], "=")
	if option[0] != "mountpoint" {
		replyJSONError(responseWriter, "Setting option "+option[0]+" forbidden",
			http.StatusForbidden)
		return
	}

	cmd := ""
	if option[1] == "none" {
		cmd = "lxc-attach -e -n " + container + " -- /bin/umount " +
			args[2]
	} else {
		cmd = fmt.Sprintf("/usr/bin/zfs set mountpoint=%s%s %s; lxc-attach -e "+
			"-n %s -- /bin/mount -t zfs %s %s", getRootFS(container), option[1],
			args[2], container, args[2], option[1])
	}

	stdout, stderr, err := runCmd(getHost(container), cmd)
	if err != nil {
		replyJSONError(responseWriter, err.Error()+" "+stderr,
			http.StatusInternalServerError)
		return
	}

	replyPlain(responseWriter, stdout, stderr)
}

func handleZfsClone(container string, args []string,
	responseWriter http.ResponseWriter) {
	err := hasPermissionsZfs(container, args[len(args)-2])
	if err != nil {
		replyJSONError(responseWriter, err.Error(), http.StatusForbidden)
		return
	}

	stdout, stderr, err := runZfsCmd(getHost(container), setMountpoint(args,
		container))
	if err != nil {
		replyJSONError(responseWriter, err.Error()+" "+stderr,
			http.StatusInternalServerError)
		return
	}

	err, stderr = remountToContainer(container, args)
	if err != nil {
		replyJSONError(responseWriter, "Created, but not mounted: "+err.Error()+
			" "+stderr, http.StatusInternalServerError)
		return
	}

	replyPlain(responseWriter, stdout, stderr)
}

func handleUsage(container string, responseWriter http.ResponseWriter) {
	stdout, stderr, err := runZfsCmd(getHost(container), []string{})
	if err != nil {
		replyJSONError(responseWriter, err.Error()+" "+stderr,
			http.StatusInternalServerError)
		return
	}

	replyPlain(responseWriter, stdout, stderr)
}

func handleZfsRename(container string, args []string,
	responseWriter http.ResponseWriter) {
	err := hasPermissionsZfs(container, args[len(args)-2])
	if err != nil {
		replyJSONError(responseWriter, err.Error(), http.StatusForbidden)
		return
	}

	stdout, stderr, err := runZfsCmd(getHost(container), args)
	if err != nil {
		replyJSONError(responseWriter, err.Error()+" "+stderr,
			http.StatusInternalServerError)
		return
	}

	replyPlain(responseWriter, stdout, stderr)
}

func handleZfsUnmount(container string, args []string,
	responseWriter http.ResponseWriter) {
	err := hasPermissionsZfs(container, args[len(args)-1])
	if err != nil {
		replyJSONError(responseWriter, err.Error(), http.StatusForbidden)
	}

	stdout, stder, err := runZfsCmd(getHost(container), args)
	if err != nil {
		replyJSONError(responseWriter, err.Error(), http.StatusForbidden)
		return
	}

	replyPlain(responseWriter, stdout, stder)
}

func getCommandLineFromRequest(request *http.Request) []string {
	cmdStr, err := url.QueryUnescape(strings.TrimPrefix(request.URL.Path,
		"/run/"))
	if err != nil {
		logger.Error(err.Error())
	}

	return strings.Fields(cmdStr)
}

func getContainerName(request *http.Request) (string, error) {
	ip := strings.Split(request.RemoteAddr, ":")[0]
	response, err := http.Get(config.ReportUrl)
	if err != nil {
		return "", err
	}

	decoder := json.NewDecoder(response.Body)
	report := Report{}
	if err := decoder.Decode(&report); err != nil {
		logger.Error(err.Error())
	}

	for _, host := range report {
		for containerName, container := range host.Containers {
			for _, ips := range container.Ips {
				for _, contIp := range ips {
					contIp = strings.Split(contIp, "/")[0]
					if contIp == ip {
						return containerName, nil
					}
				}
			}
		}
	}

	return "", errors.New("container not found")
}

func replyJSONError(responseWriter http.ResponseWriter, error string, code int) {
	logger.Error(error)
	tmp, _ := json.Marshal(&ErrorResponse{Error: error})
	http.Error(responseWriter, string(tmp), code)
}

func replyTable(responseWriter http.ResponseWriter, stderr string,
	header []string, data []map[string]string) {
	json.NewEncoder(responseWriter).Encode(&TableMessage{
		Stderr: strings.Split(stderr, "\n"),
		Stdout: TableStdout{
			Format: "table",
			Header: header,
			Data:   data,
		}})
}

func replyPlain(responseWriter http.ResponseWriter, stdout, stderr string) {
	json.NewEncoder(responseWriter).Encode(&PlainMessage{
		Stdout: PlainData{Data: strings.Split(stdout, "\n")},
		Stderr: strings.Split(stderr, "\n"),
	})
}

func hasPermissionsZfs(container, target string) error {
	if strings.HasPrefix(target, "/") {
		return nil
	}

	if strings.Contains(target, "@") {
		target = strings.Split(target, "@")[0]
	}

	acceptedTargets := []string{}
	datasets := []string{}
	stdout, _, err := runZfsCmd(getHost(container), []string{"list", "-t", "all"})
	if err != nil {
		return err
	}

	_, table := createTableFromString(stdout)
	matches := filterByRootFs(table, container)
	for _, match := range matches {
		datasets = append(datasets, match["name"])
	}

	for _, newRow := range filterByDatasets(table, datasets) {
		matches = append(matches, newRow)
	}

	for _, row := range matches {
		acceptedTargets = append(acceptedTargets, row["name"])
	}

	for _, acceptedTarget := range acceptedTargets {
		if acceptedTarget == target {
			return nil
		}

	}

	return errors.New("access forbidden")
}

func createTableFromString(src string) ([]string, []map[string]string) {
	out := strings.Split(strings.Trim(src, "\n"), "\n")
	header := strings.Fields(strings.ToLower(out[0]))
	result := []map[string]string{}
	for _, str := range out {
		newRow := make(map[string]string)
		rows := strings.Fields(str)
		for i, row := range strings.Fields(out[0]) {
			newRow[strings.ToLower(row)] = rows[i]
		}

		result = append(result, newRow)
	}

	return header, result
}

func filterByRootFs(tableData []map[string]string, container string,
) []map[string]string {
	result := []map[string]string{}
	rootfs := getRootFS(container)
	for _, row := range tableData {
		if strings.Contains(row["mountpoint"], rootfs) {
			row["mountpoint"] = path.Join("/", strings.Replace(row["mountpoint"],
				rootfs, "", -1))
			result = append(result, row)
		}

	}

	return result
}

func filterByDatasets(tableData []map[string]string, datasets []string,
) []map[string]string {
	result := []map[string]string{}
	for _, row := range tableData {
		for _, name := range datasets {
			if strings.Contains(row["name"], name+"@") {
				result = append(result, row)
			}

		}

	}

	return result
}

func getRootFS(container string) string {
	host := getHost(container) + ":" + fmt.Sprint(config.Port)
	if host == "" {
		return ""
	}

	runner, err := runcmd.NewRemoteKeyAuthRunner(config.User, host,
		config.KeyFile)
	if err != nil {
		logger.Error(err.Error())
		return ""
	}

	cmd, err := runner.Command("/usr/bin/grep -e lxc.rootfs /var/lib/lxc/" +
		container + "/config")
	if err != nil {
		logger.Error(err.Error())
		return ""
	}

	out, err := cmd.Run()
	if err != nil {
		logger.Error(err.Error())
		return ""
	}

	result := strings.Fields(out[0])[2]

	return result
}

func getHost(container string) string {
	response, err := http.Get(config.ReportUrl)
	if err != nil {
		logger.Error(err.Error())
		return ""
	}

	if response.StatusCode != 200 {
		return ""
	}

	decoder := json.NewDecoder(response.Body)
	report := Report{}
	if err := decoder.Decode(&report); err != nil {
		logger.Error(err.Error())
	}

	for name, host := range report {
		if _, ok := host.Containers[container]; ok {
			return name
		}

	}

	return ""
}

func runZfsCmd(host string, args []string) (stdout, stderr string, err error) {
	argStr := strings.Join(args, " ")
	return runCmd(host, "/usr/bin/zfs "+argStr)
}

func runCmd(host, cmdStr string) (stdout, stderr string, err error) {
	runner, err := runcmd.NewRemoteKeyAuthRunner(config.User,
		host+":"+fmt.Sprint(config.Port),
		config.KeyFile)
	if err != nil {
		logger.Error(err.Error())
		return "", "", err
	}

	cmd, err := runner.Command(cmdStr)
	if err != nil {
		logger.Error(err.Error())
		return "", "", err
	}

	err = cmd.Start()
	if err != nil {
		logger.Error(err.Error())
	}

	bOut, err := ioutil.ReadAll(cmd.StdoutPipe())
	if err != nil {
		logger.Error(err.Error())
		return "", "", err
	}

	bErr, err := ioutil.ReadAll(cmd.StderrPipe())
	if err != nil {
		logger.Error(err.Error())
		return "", "", err
	}

	return string(bOut), string(bErr), nil
}

func setMountpoint(args []string, container string) []string {
	result := []string{}
	optionsFlag := false
	done := false
	for i, arg := range args {
		if optionsFlag {
			if strings.Contains(arg, "mountpoint=") {
				arg = strings.Replace(arg, "mountpoint=", "mountpoint="+
					getRootFS(container), 1)
				done = true
			} else {
				arg = arg + ",mountpoint=" + getRootFS(container) + "/" +
					args[len(args)-1]
				done = true
			}

			optionsFlag = false
		}

		if i == len(args)-1 && !done {
			result = append(result, "-o")
			result = append(result, "mountpoint="+getRootFS(container)+"/"+arg)
		}

		if strings.HasPrefix(arg, "-") && strings.Contains(arg, "o") {
			optionsFlag = true
		}

		result = append(result, arg)
	}

	return result
}

func remountToContainer(container string, args []string) (error, string) {
	stdout, stderr, err := runZfsCmd(getHost(container), []string{"list"})
	if err != nil {
		return err, stderr
	}
	_, table := createTableFromString(stdout)
	data := filterByRootFs(table, container)
	_, stderr, err = runZfsCmd(getHost(container), []string{"umount ",
		args[len(args)-1]})
	if err != nil {
		return err, stderr
	}

	mountpoint := ""
	for _, row := range data {
		if row["name"] == args[len(args)-1] {
			mountpoint = row["mountpoint"]
			break
		}

	}

	cmdStr := "lxc-attach -e -n " + container + " -- /bin/mount -t zfs " +
		args[len(args)-1] + " " + mountpoint
	_, stderr, err = runCmd(getHost(container), cmdStr)
	return err, stderr
}

func getListType(args []string) string {
	for i, arg := range args {
		if strings.HasPrefix(arg, "-") && strings.Contains(arg, "t") {
			return args[i+1]
		}

	}

	return "filesystem"
}
