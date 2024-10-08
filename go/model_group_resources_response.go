/*
 * Opal Custom App Connector API
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: 1.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

type GroupResourcesResponse struct {

	// The cursor with which to continue pagination if additional result pages exist.
	NextCursor *string `json:"next_cursor,omitempty"`

	Resources []GroupResource `json:"resources"`
}
