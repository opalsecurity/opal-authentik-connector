/*
 * Opal Custom App Connector API
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: 1.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

import (
	"github.com/GIT_USER_ID/GIT_REPO_ID/go/authentik"
	"github.com/gin-gonic/gin"
)

type UsersAPI struct {
}

// Get /users
func (api *UsersAPI) GetUsers(c *gin.Context) {
	authentik, err := authentik.NewAuthentikClient()
	if err != nil {
		c.JSON(500, buildRespFromErr(err))
		return
	}

	users, nextCursor, err := authentik.PaginatedListUsers(c)
	if err != nil {
		c.JSON(500, buildRespFromErr(err))
		return
	}

	c.JSON(200, gin.H{
		"users":       users,
		"next_cursor": nextCursor,
	})
}
