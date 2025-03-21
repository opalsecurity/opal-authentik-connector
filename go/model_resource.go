/*
 * Opal Custom App Connector API
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: 1.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

type Resource struct {

	// The id of the resource in your system. Opal will provide this id when making requests for the resource to your connector.
	Id string `json:"id"`

	// The name of the resource
	Name string `json:"name"`

	// The description of the resource. If provided, it will be imported into Opal.
	Description string `json:"description,omitempty"`
}
