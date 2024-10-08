/*
 * Opal Custom App Connector API
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: 1.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

type User struct {

	// The id of the user in your system. Opal will provide this id when making requests for the user to your connector.
	Id string `json:"id"`

	// The email of the user. Opal will use this to associate the user with the corresponding Opal user.
	Email string `json:"email"`
}
