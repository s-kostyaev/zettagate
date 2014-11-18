package main

import (
	"github.com/gin-gonic/gin"
	"strings"
)

func authContainer() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Add("Content-Type", "application/json")
		if setContainer(c) {
			c.Next()
			return
		}
		c.JSON(403, gin.H{"error": "access forbidden"})
		c.Abort(403)
	}
}

func checkTarget() gin.HandlerFunc {
	return func(c *gin.Context) {
		args := getArgs(c)
		target := args[len(args)-1]
		if getSubcommand(c) == "clone" {
			target = args[len(args)-2]
		}
		if strings.HasPrefix(target, "/") {
			c.Next()
			return
		}
		if strings.Contains(target, "@") {
			target = strings.Split(target, "@")[0]
		}
		acceptedTargets := []string{}
		datasets := []string{}
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
		stdout, _, err = run(getHost(getContainer(c)),
			"/usr/bin/zfs list -t all")
		if err != nil {
			c.JSON(503, gin.H{"error": err.Error()})
			return
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
		data := filterByRootFs(result, getContainer(c))
		for _, d := range filterByDatasets(result, datasets) {
			data = append(data, d)
		}
		for _, m := range data {
			acceptedTargets = append(acceptedTargets, m["name"])
		}
		for _, aTarget := range acceptedTargets {
			if aTarget == target {
				c.Next()
				return
			}
		}

		c.JSON(403, gin.H{"error": "access forbidden"})
		c.Abort(403)
	}
}
