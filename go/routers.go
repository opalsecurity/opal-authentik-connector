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
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// Route is the information for every URI.
type Route struct {
	// Name is the name of this Route.
	Name string
	// Method is the string for the HTTP method. ex) GET, POST etc..
	Method string
	// Pattern is the pattern of the URI.
	Pattern string
	// HandlerFunc is the handler function of this route.
	HandlerFunc gin.HandlerFunc
}

// NewRouter returns a new router.
func NewRouter(handleFunctions ApiHandleFunctions) *gin.Engine {
	return NewRouterWithGinEngine(gin.Default(), handleFunctions)
}

// NewRouter add routes to existing gin engine.
func NewRouterWithGinEngine(router *gin.Engine, handleFunctions ApiHandleFunctions) *gin.Engine {
	router.Use(validateOpalSignature(os.Getenv("OPAL_SIGNING_SECRET")))

	for _, route := range getRoutes(handleFunctions) {
		if route.HandlerFunc == nil {
			route.HandlerFunc = DefaultHandleFunc
		}
		switch route.Method {
		case http.MethodGet:
			router.GET(route.Pattern, route.HandlerFunc)
		case http.MethodPost:
			router.POST(route.Pattern, route.HandlerFunc)
		case http.MethodPut:
			router.PUT(route.Pattern, route.HandlerFunc)
		case http.MethodPatch:
			router.PATCH(route.Pattern, route.HandlerFunc)
		case http.MethodDelete:
			router.DELETE(route.Pattern, route.HandlerFunc)
		}
	}

	return router
}

// Default handler for not yet implemented routes
func DefaultHandleFunc(c *gin.Context) {
	c.String(http.StatusNotImplemented, "501 not implemented")
}

type ApiHandleFunctions struct {

	// Routes for the GroupsAPI part of the API
	GroupsAPI GroupsAPI
	// Routes for the ResourcesAPI part of the API
	ResourcesAPI ResourcesAPI
	// Routes for the StatusAPI part of the API
	StatusAPI StatusAPI
	// Routes for the UsersAPI part of the API
	UsersAPI UsersAPI
}

func getRoutes(handleFunctions ApiHandleFunctions) []Route {
	return []Route{
		{
			"AddGroupMemberGroup",
			http.MethodPost,
			"/groups/:group_id/member-groups",
			handleFunctions.GroupsAPI.AddGroupMemberGroup,
		},
		{
			"AddGroupResource",
			http.MethodPost,
			"/groups/:group_id/resources",
			handleFunctions.GroupsAPI.AddGroupResource,
		},
		{
			"AddGroupUser",
			http.MethodPost,
			"/groups/:group_id/users",
			handleFunctions.GroupsAPI.AddGroupUser,
		},
		{
			"GetGroup",
			http.MethodGet,
			"/groups/:group_id",
			handleFunctions.GroupsAPI.GetGroup,
		},
		{
			"RemoveGroupMemberGroup",
			http.MethodDelete,
			"/groups/:group_id/member-groups/:member_group_id",
			handleFunctions.GroupsAPI.RemoveGroupMemberGroup,
		},
		{
			"GetGroupResources",
			http.MethodGet,
			"/groups/:group_id/resources",
			handleFunctions.GroupsAPI.GetGroupResources,
		},
		{
			"GetGroupUsers",
			http.MethodGet,
			"/groups/:group_id/users",
			handleFunctions.GroupsAPI.GetGroupUsers,
		},
		{
			"GetGroups",
			http.MethodGet,
			"/groups",
			handleFunctions.GroupsAPI.GetGroups,
		},
		{
			"GetGroupMemberGroups",
			http.MethodGet,
			"/groups/:group_id/member-groups",
			handleFunctions.GroupsAPI.GetGroupMemberGroups,
		},
		{
			"RemoveGroupResource",
			http.MethodDelete,
			"/groups/:group_id/resources/:resource_id",
			handleFunctions.GroupsAPI.RemoveGroupResource,
		},
		{
			"RemoveGroupUser",
			http.MethodDelete,
			"/groups/:group_id/users/:user_id",
			handleFunctions.GroupsAPI.RemoveGroupUser,
		},
		{
			"AddResourceUser",
			http.MethodPost,
			"/resources/:resource_id/users",
			handleFunctions.ResourcesAPI.AddResourceUser,
		},
		{
			"GetResource",
			http.MethodGet,
			"/resources/:resource_id",
			handleFunctions.ResourcesAPI.GetResource,
		},
		{
			"GetResourceAccessLevels",
			http.MethodGet,
			"/resources/:resource_id/access_levels",
			handleFunctions.ResourcesAPI.GetResourceAccessLevels,
		},
		{
			"GetResourceUsers",
			http.MethodGet,
			"/resources/:resource_id/users",
			handleFunctions.ResourcesAPI.GetResourceUsers,
		},
		{
			"GetResources",
			http.MethodGet,
			"/resources",
			handleFunctions.ResourcesAPI.GetResources,
		},
		{
			"RemoveResourceUser",
			http.MethodDelete,
			"/resources/:resource_id/users/:user_id",
			handleFunctions.ResourcesAPI.RemoveResourceUser,
		},
		{
			"GetStatus",
			http.MethodGet,
			"/status",
			handleFunctions.StatusAPI.GetStatus,
		},
		{
			"GetUsers",
			http.MethodGet,
			"/users",
			handleFunctions.UsersAPI.GetUsers,
		},
	}
}

// GenerateSignature generates a signature for a given payload in a HTTP request
func GenerateSignature(
	signingSecret string,
	timestamp string,
	serializedBlob []byte,
) (string, error) {
	// Concatenate base string
	sigBaseString := "v0:" + timestamp + ":" + string(serializedBlob)

	// Hash base string to get signature
	hash := hmac.New(sha256.New, []byte(signingSecret))
	_, err := hash.Write([]byte(sigBaseString))
	if err != nil {
		return "", errors.Wrap(err, "error writing hash")
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func validateOpalSignature(signingSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		opalSignature := c.GetHeader("X-Opal-Signature")
		if opalSignature == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, &Error{
				Code:    http.StatusUnauthorized,
				Message: "X-Opal-Signature header is missing",
			})
			return
		}
		opalRequestTimestamp := c.GetHeader("X-Opal-Request-Timestamp")
		if opalRequestTimestamp == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, &Error{
				Code:    http.StatusUnauthorized,
				Message: "X-Opal-Request-Timestamp header is missing",
			})
			return
		}

		var bodyStr string
		// Read request body, once the request body is read, it cannot be read again
		// so we need to save it in a variable and then reassign it to the Request.Body
		var bodyBytes []byte
		var err error
		if c.Request.Body != nil {
			bodyBytes, err = ioutil.ReadAll(c.Request.Body)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, &Error{
					Code:    http.StatusInternalServerError,
					Message: "Unable to read request body",
				})
				return
			}
			c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
			bodyStr = strings.TrimSpace(string(bodyBytes))
		}
		if bodyStr == "" {
			bodyStr = "{}"
		}

		signature, err := GenerateSignature(signingSecret, opalRequestTimestamp, []byte(bodyStr))
		if signature != opalSignature || err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, &Error{
				Code:    http.StatusUnauthorized,
				Message: "Invalid signature",
			})
			return
		}

		c.Next()
	}
}
