package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()
	router.Use(authContainer())

	destructive := router.Group("/")
	destructive.Use(checkTarget())

	destructive.POST("/run/snap/*args", snap)
	destructive.POST("/run/snapshot/*args", snap)
	destructive.POST("/run/clone/*args", clone)
	destructive.POST("/run/rename/*args", rename)
	destructive.POST("/run/destroy/*args", destroy)

	router.POST("/run/", root)
	router.POST("/run/list/*args", list)
	router.POST("/run/create/*args", create)
	router.POST("/run/set/*args", set)

	router.NotFound404(notImplemented)

	router.Run(":" + fmt.Sprint(config.ServicePort))

}
