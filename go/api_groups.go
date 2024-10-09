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
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	authentik "goauthentik.io/api/v3"
)

type GroupsAPI struct {
}

// Post /groups/:group_id/resources
func (api *GroupsAPI) AddGroupResource(c *gin.Context) {
	// Your handler implementation
	c.JSON(200, gin.H{"status": "OK"})
}

// Post /groups/:group_id/users
func (api *GroupsAPI) AddGroupUser(c *gin.Context) {
	groupID := c.Param("group_id")

	var addGroupUserRequest AddGroupUserRequest
	err := c.BindJSON(&addGroupUserRequest)
	if err != nil {
		c.JSON(401, buildRespFromErr(err, 401))
		return
	}

	authentik, err := NewAuthentikClient()
	if err != nil {
		c.JSON(500, buildRespFromErr(err, 500))
		return
	}

	err = authentik.AddUserToGroup(c, groupID, addGroupUserRequest.UserId)
	if err != nil {
		var clientErr *ClientError
		if errors.As(err, &clientErr) {
			c.JSON(clientErr.StatusCode, buildRespFromErr(err, clientErr.StatusCode))
		} else {
			c.JSON(500, buildRespFromErr(err, 500))
		}
		return
	}

	c.JSON(200, gin.H{})
}

// Get /groups/:group_id
func (api *GroupsAPI) GetGroup(c *gin.Context) {
	groupID := c.Param("group_id")

	authentik, err := NewAuthentikClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, buildRespFromErr(err, http.StatusInternalServerError))
		return
	}

	authentikGroup, err := authentik.GetGroup(c, groupID)
	if err != nil {
		var clientErr *ClientError
		if errors.As(err, &clientErr) {
			c.JSON(clientErr.StatusCode, buildRespFromErr(err, clientErr.StatusCode))
		} else {
			c.JSON(http.StatusInternalServerError, buildRespFromErr(err, http.StatusInternalServerError))
		}
		return
	}

	opalGroup := toOpalGroup(authentikGroup)

	c.JSON(http.StatusOK, gin.H{"group": *opalGroup})
}

// Get /groups/:group_id/resources
func (api *GroupsAPI) GetGroupResources(c *gin.Context) {
	// Authentik groupresources not supported
	nextCursor := ""
	c.JSON(http.StatusOK, &GroupResourcesResponse{NextCursor: &nextCursor, Resources: []GroupResource{}})
}

// Get /groups/:group_id/users
func (api *GroupsAPI) GetGroupUsers(c *gin.Context) {
	groupID := c.Param("group_id")

	authentik, err := NewAuthentikClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, buildRespFromErr(err, http.StatusInternalServerError))
		return
	}

	groupMemberships, err := authentik.GetGroupUsers(c, groupID)
	if err != nil {
		var clientErr *ClientError
		if errors.As(err, &clientErr) {
			c.JSON(clientErr.StatusCode, buildRespFromErr(err, clientErr.StatusCode))
		} else {
			c.JSON(http.StatusInternalServerError, buildRespFromErr(err, http.StatusInternalServerError))
		}
		return
	}

	groupUsers := make([]GroupUser, 0)
	for _, groupMembership := range groupMemberships {
		groupUsers = append(groupUsers, GroupUser{
			// The group member primary key is the user's primary key, which is the user ID we use throughout Opal
			UserId: strconv.Itoa(int(groupMembership.GetPk())),
			Email:  groupMembership.GetEmail(),
		})
	}

	// Next cursor being "" tells Opal this is the last page. Since GroupUsers are not paginated in Authentik we use this
	nextCursor := ""
	c.JSON(http.StatusOK, GroupUsersResponse{
		NextCursor: &nextCursor,
		Users:      groupUsers,
	})
}

// Get /groups
func (api *GroupsAPI) GetGroups(c *gin.Context) {
	authentik, err := NewAuthentikClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, buildRespFromErr(err, http.StatusInternalServerError))
		return
	}

	authentikGroups, nextCursor, err := authentik.PaginatedListGroups(c)
	if err != nil {
		var clientErr *ClientError
		if errors.As(err, &clientErr) {
			c.JSON(clientErr.StatusCode, buildRespFromErr(err, clientErr.StatusCode))
		} else {
			c.JSON(http.StatusInternalServerError, buildRespFromErr(err, http.StatusInternalServerError))
		}
		return
	}

	groups := make([]Group, 0)
	for _, authentikGroup := range authentikGroups {
		group := toOpalGroup(&authentikGroup)
		groups = append(groups, *group)
	}

	c.JSON(http.StatusOK, GroupsResponse{
		Groups:     groups,
		NextCursor: &nextCursor,
	})
}

// Delete /groups/:group_id/resources/:resource_id
func (api *GroupsAPI) RemoveGroupResource(c *gin.Context) {
	// Your handler implementation
	c.JSON(200, gin.H{"status": "OK"})
}

// Delete /groups/:group_id/users/:user_id
func (api *GroupsAPI) RemoveGroupUser(c *gin.Context) {
	groupID := c.Param("group_id")
	userID := c.Param("user_id")

	authentik, err := NewAuthentikClient()
	if err != nil {
		c.JSON(500, buildRespFromErr(err, 500))
		return
	}

	err = authentik.RemoveUserFromGroup(c, groupID, userID)
	if err != nil {
		var clientErr *ClientError
		if errors.As(err, &clientErr) {
			c.JSON(clientErr.StatusCode, buildRespFromErr(err, clientErr.StatusCode))
		} else {
			c.JSON(500, buildRespFromErr(err, 500))
		}
		return
	}

	c.JSON(200, gin.H{})
}

func toOpalGroup(group *authentik.Group) *Group {
	return &Group{
		// There are multiple available IDs for the groups, UID, UUID and PK
		// PK is the same as UUID, and is guaranteed to be available throughout other Authentik APIs
		// Therefore, we use the group's primary key as its ID in Opal
		Id:   group.GetPk(),
		Name: group.GetName(),
		// Description is not available for authentik groups
	}
}
