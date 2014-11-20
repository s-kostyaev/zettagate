package main

import (
	"github.com/gin-gonic/gin"
	"path"
	"strings"
)

func notImplemented(c *gin.Context) {
	c.JSON(404, gin.H{"error": "not implemented"})
}

func list(c *gin.Context) {
	datasets := []string{}
	byDatasets := false
	args := " " + strings.Join(getArgs(c), " ")
	stdout, stderr, err := run(getHost(getContainer(c)), "/usr/bin/zfs list"+
		args)
	if err != nil {
		c.JSON(503, gin.H{"error": err.Error()})
		return
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
	if strings.Contains(args, "t all") || strings.Contains(args, "t snap") {
		stdout, _, err := run(getHost(getContainer(c)), "/usr/bin/zfs list")
		if err != nil {
			c.JSON(503, gin.H{"error": err.Error()})
			return
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
		matches := filterByRootFs(result, getContainer(c))
		for _, match := range matches {
			datasets = append(datasets, match["name"])
		}
		byDatasets = true
	}
	data := filterByRootFs(result, getContainer(c))
	if byDatasets {
		for _, d := range filterByDatasets(result, datasets) {
			data = append(data, d)
		}
	}
	c.JSON(200, gin.H{"stdout": gin.H{"format": "table",
		"data": data},
		"stderr": strings.Split(stderr, "\n")})
}

func mount(c *gin.Context) {
	args := " " + strings.Join(getArgs(c), " ")
	stdout, stderr, err := run(getHost(getContainer(c)), "/usr/bin/zfs mount"+
		args)
	if err != nil {
		c.JSON(503, gin.H{"error": err.Error()})
		return
	}
	out := strings.Split(strings.Trim(stdout, "\n"), "\n")
	result := []map[string]string{}
	for _, str := range out {
		m := make(map[string]string)
		rows := strings.Fields(str)
		if len(rows) == 0 {
			c.JSON(200, gin.H{"stdout": gin.H{"data": strings.Split(string(
				stdout), "\n")}, "stderr": strings.Split(stderr, "\n")})
			return
		}
		for i, row := range []string{"dataset", "mountpoint"} {
			m[strings.ToLower(row)] = rows[i]
		}
		result = append(result, m)
	}
	c.JSON(200, gin.H{"stdout": gin.H{"format": "table",
		"data": filterByRootFs(result, getContainer(c))},
		"stderr": strings.Split(stderr, "\n")})
}

func umount(c *gin.Context) {
	args := " " + strings.Join(func() []string {
		args := getArgs(c)
		if strings.HasPrefix(args[len(args)-1], "/") {
			args[len(args)-1] = path.Join("/", getRootFS(getContainer(c)),
				args[len(args)-1])
		}
		return args
	}(), " ")
	stdout, stderr, err := run(getHost(getContainer(c)), "/usr/bin/zfs umount"+
		args)
	if err != nil {
		c.JSON(503, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"stdout": gin.H{"data": strings.Split(string(stdout),
		"\n")}, "stderr": strings.Split(stderr, "\n")})
}

func snap(c *gin.Context) {
	args := " " + strings.Join(getArgs(c), " ")
	stdout, stderr, err := run(getHost(getContainer(c)), "/usr/bin/zfs snap"+
		args)
	if err != nil {
		c.JSON(503, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"stdout": gin.H{"data": strings.Split(string(stdout),
		"\n")}, "stderr": strings.Split(stderr, "\n")})
}

func create(c *gin.Context) {
	args := " " + strings.Join(setMountpoint(getArgs(c), getContainer(c)), " ")
	stdout, stderr, err := run(getHost(getContainer(c)), "/usr/bin/zfs create"+
		args)
	if err != nil {
		c.JSON(503, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"stdout": gin.H{"data": strings.Split(string(stdout),
		"\n")}, "stderr": strings.Split(stderr, "\n")})
}

func destroy(c *gin.Context) {
	args := " " + strings.Join(getArgs(c), " ")
	stdout, stderr, err := run(getHost(getContainer(c)), "/usr/bin/zfs destroy"+
		args)
	if err != nil {
		c.JSON(503, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"stdout": gin.H{"data": strings.Split(string(stdout),
		"\n")}, "stderr": strings.Split(stderr, "\n")})
}

func set(c *gin.Context) {
	args := getArgs(c)
	option := strings.Split(args[0], "=")
	if option[0] != "mountpoint" {
		c.JSON(403, gin.H{"error": "Setting option " + option[0] + " forbidden"})
		return
	}
	stdout, stderr, err := run(getHost(getContainer(c)),
		"lxc-attach -e -n "+getContainer(c)+" -- /bin/mount -t zfs "+args[1]+
			" "+option[1])
	if err != nil {
		c.JSON(503, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"stdout": gin.H{"data": strings.Split(string(stdout),
		"\n")}, "stderr": strings.Split(stderr, "\n")})
}

func clone(c *gin.Context) {
	args := " " + strings.Join(setMountpoint(getArgs(c), getContainer(c)), " ")
	stdout, stderr, err := run(getHost(getContainer(c)), "/usr/bin/zfs clone"+
		args)
	if err != nil {
		c.JSON(503, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"stdout": gin.H{"data": strings.Split(string(stdout),
		"\n")}, "stderr": strings.Split(stderr, "\n")})
}

func root(c *gin.Context) {
	if _, ok := cachedRoot["stdout"]; ok {
		c.JSON(200, cachedRoot)
		return
	}
	stdout, stderr, err := run(getHost(getContainer(c)), "/usr/bin/zfs")
	if err != nil {
		c.JSON(503, gin.H{"error": err.Error()})
		return
	}
	cachedRoot = gin.H{"stdout": gin.H{"data": strings.Split(string(stdout),
		"\n")}, "stderr": strings.Split(string(stderr), "\n")}
	c.JSON(200, cachedRoot)
}
