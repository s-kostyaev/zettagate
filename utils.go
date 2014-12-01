package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/theairkit/runcmd"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"
)

type Report map[string]Host

type Host struct {
	Hostname   string
	NetAddr    []string
	Pools      []string
	Containers map[string]Container
}

type Container map[string]string

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

func getArgs(r *http.Request) []string {
	cmdStr, err := url.QueryUnescape(strings.TrimPrefix(r.URL.Path, "/run/"))
	if err != nil {
		logger.Error(err.Error())
	}
	return strings.Fields(cmdStr)
}

func getContainer(r *http.Request) (string, error) {
	ip := strings.Split(r.RemoteAddr, ":")[0]
	response, err := http.Get(config.ReportUrl)
	if err != nil {
		return "", err
	}
	dec := json.NewDecoder(response.Body)
	report := Report{}
	if err := dec.Decode(&report); err != nil && err != io.EOF {
		logger.Error(err.Error())
	}
	for _, host := range report {
		for cname, container := range host.Containers {
			if container["ip"] == ip {
				return cname, nil
			}
		}
	}
	return "", errors.New("access forbidden")
}

func returnError(w http.ResponseWriter, error string, code int) {
	tmp, _ := json.Marshal(&ErrorResponse{Error: error})
	http.Error(w, string(tmp), code)
}

func returnTable(w http.ResponseWriter, stderr string, header []string,
	data []map[string]string) {
	tmp, _ := json.Marshal(&TableMessage{
		Stderr: strings.Split(stderr, "\n"),
		Stdout: TableStdout{
			Format: "table",
			Header: header,
			Data:   data,
		}})
	fmt.Fprintln(w, string(tmp))
}

func returnPlain(w http.ResponseWriter, stdout, stderr string) {
	tmp, _ := json.Marshal(&PlainMessage{
		Stdout: PlainData{Data: strings.Split(stdout, "\n")},
		Stderr: strings.Split(stderr, "\n"),
	})
	fmt.Fprintln(w, string(tmp))
}

func checkTarget(container, target string) (bool, error) {
	if strings.HasPrefix(target, "/") {
		return true, nil
	}
	if strings.Contains(target, "@") {
		target = strings.Split(target, "@")[0]
	}
	acceptedTargets := []string{}
	datasets := []string{}
	stdout, _, err := run(getHost(container), "list")
	if err != nil {
		return false, err
	}
	out := strings.Split(strings.Trim(stdout, "\n"), "\n")
	result := []map[string]string{}
	for _, str := range out {
		m := make(map[string]string)
		rows := strings.Fields(str)
		for i, row := range strings.Fields(out[0]) {
			m[strings.ToLower(row)] = rows[i]
		}
		result = append(result, m)
	}
	matches := filterByRootFs(result, container)
	for _, match := range matches {
		datasets = append(datasets, match["name"])
	}
	stdout, _, err = run(getHost(container), "list -t all")
	if err != nil {
		return false, err
	}
	out = strings.Split(strings.Trim(stdout, "\n"), "\n")
	result = []map[string]string{}
	for _, str := range out {
		m := make(map[string]string)
		rows := strings.Fields(str)
		for i, row := range strings.Fields(out[0]) {
			m[strings.ToLower(row)] = rows[i]
		}
		result = append(result, m)
	}
	data := filterByRootFs(result, container)
	for _, d := range filterByDatasets(result, datasets) {
		data = append(data, d)
	}
	for _, m := range data {
		acceptedTargets = append(acceptedTargets, m["name"])
	}
	for _, aTarget := range acceptedTargets {
		if aTarget == target {
			return true, nil
		}
	}

	return false, errors.New("access forbidden")
}

func filterByRootFs(src []map[string]string,
	container string) []map[string]string {
	result := []map[string]string{}
	rootfs := getRootFS(container)
	for _, m := range src {
		if strings.Contains(m["mountpoint"], rootfs) {
			m["mountpoint"] = path.Join("/", strings.Replace(m["mountpoint"],
				rootfs, "", -1))
			result = append(result, m)
		}
	}
	return result
}

func filterByDatasets(src []map[string]string,
	datasets []string) []map[string]string {
	result := []map[string]string{}
	for _, m := range src {
		for _, name := range datasets {
			if strings.Contains(m["name"], name+"@") {
				result = append(result, m)
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
	c, err := runner.Command("/usr/bin/grep -e lxc.rootfs /var/lib/lxc/" +
		container + "/config")
	if err != nil {
		logger.Error(err.Error())
		return ""
	}
	out, err := c.Run()
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
	dec := json.NewDecoder(response.Body)
	report := Report{}
	if err := dec.Decode(&report); err != nil && err == io.EOF {
		logger.Error(err.Error())
	}
	for name, host := range report {
		if _, ok := host.Containers[container]; ok {
			return name
		}
	}
	return ""
}

func run(host, argStr string) (stdout, stderr string, err error) {
	runner, err := runcmd.NewRemoteKeyAuthRunner(config.User,
		host+":"+fmt.Sprint(config.Port),
		config.KeyFile)
	if err != nil {
		logger.Error(err.Error())
		return "", "", err
	}
	cmd, err := runner.Command("/usr/bin/zfs " + argStr)
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
	stdout, stderr, err := run(getHost(container), "list")
	if err != nil {
		return err, stderr
	}
	out := strings.Split(strings.Trim(stdout, "\n"), "\n")
	result := []map[string]string{}
	for _, str := range out {
		m := make(map[string]string)
		rows := strings.Fields(str)
		for i, row := range strings.Fields(out[0]) {
			m[strings.ToLower(row)] = rows[i]
		}
		result = append(result, m)
	}
	data := filterByRootFs(result, container)
	argStr := "umount " + args[len(args)-1]
	_, stderr, err = run(getHost(container), argStr)
	if err != nil {
		return err, stderr
	}
	mountpoint := ""
	for _, m := range data {
		if m["name"] == args[len(args)-1] {
			mountpoint = m["mountpoint"]
			break
		}
	}
	cmdStr := "lxc-attach -e -n " + container + " -- /bin/mount -t zfs " +
		args[len(args)-1] + " " + mountpoint
	_, stderr, err = runCmd(getHost(container), cmdStr)
	return err, stderr
}
