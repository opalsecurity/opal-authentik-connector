package openapi

import "github.com/gin-gonic/gin"

func buildRespFromErr(err error, code int) gin.H {
	return gin.H{
		"message": err.Error(),
		"code":    code,
	}
}
