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

	destructive.POST("/snap/", snap)
	destructive.POST("/snap", snap)
	destructive.POST("/snapshot/", snap)
	destructive.POST("/snapshot", snap)
	destructive.POST("/clone", clone)
	destructive.POST("/clone/", clone)
	destructive.POST("/rename", rename)
	destructive.POST("/rename/", rename)
	destructive.DELETE("/destroy", destroy)
	destructive.DELETE("/destroy/", destroy)

	router.GET("/", root)
	router.GET("/list/", list)
	router.GET("/list", list)
	router.POST("/create", create)
	router.POST("/create/", create)
	router.POST("/set", set)
	router.POST("/set/", set)

	router.NotFound404(notImplemented)

	router.Run(":" + fmt.Sprint(config.ServicePort))

}
