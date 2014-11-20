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

	destructive.POST("/mount/", mount)
	destructive.POST("/mount", mount)
	destructive.POST("/umount/", umount)
	destructive.POST("/unmount/", umount)
	destructive.POST("/umount", umount)
	destructive.POST("/unmount", umount)
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
	router.GET("/mount", mount)
	router.GET("/mount/", mount)
	router.POST("/create", create)
	router.POST("/create/", create)
	router.POST("/set", set)
	router.POST("/set/", set)

	router.NotFound404(notImplemented)

	router.Run(":" + fmt.Sprint(config.ServicePort))

}
