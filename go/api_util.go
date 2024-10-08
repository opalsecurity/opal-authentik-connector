package openapi

import "github.com/gin-gonic/gin"

func buildRespFromErr(err error) gin.H {
	return gin.H{
		"message": err.Error(),
		"code":    500,
	}
}
