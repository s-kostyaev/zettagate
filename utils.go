package main

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/theairkit/runcmd"
	"io"
	"io/ioutil"
	"net/http"
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

var (
	cachedRoot = gin.H{}
)

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

func getContainer(c *gin.Context) string {
	return c.MustGet("container").(string)
}

func setContainer(c *gin.Context) bool {
	ip := strings.Split(c.Request.RemoteAddr, ":")[0]
	response, err := http.Get(config.ReportUrl)
	if err != nil {
		logger.Error(err.Error())
		return false
	}
	dec := json.NewDecoder(response.Body)
	report := Report{}
	if err := dec.Decode(&report); err != nil && err != io.EOF {
		logger.Error(err.Error())
	}
	for _, host := range report {
		for cname, container := range host.Containers {
			if container["ip"] == ip {
				c.Set("container", cname)
				return true
			}
		}
	}
	return false
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

func getArgs(c *gin.Context) []string {
	result := []string{}
	values := c.Request.URL.Query()
	for key, value := range values {
		if key != "last" {
			result = append(result, key)
			if value[0] != "" {
				result = append(result, value[0])
			}
		}
	}
	result = append(result, values.Get("last"))
	return result
}

func run(host, command string) (stdout, stderr string, err error) {
	runner, err := runcmd.NewRemoteKeyAuthRunner(config.User,
		host+":"+fmt.Sprint(config.Port),
		config.KeyFile)
	if err != nil {
		logger.Error(err.Error())
		return "", "", err
	}
	cmd, err := runner.Command(command)
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

func getSubcommand(c *gin.Context) string {
	return strings.Split(strings.Trim(c.Request.URL.RequestURI(), "/"), "?")[0]
}
