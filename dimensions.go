// Copyright 2019 James Cote
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package intacct

import "encoding/xml"

// GetDimensions Lists all standard dimensions and UDDs in a company
// along with helpful integration information about the object.
// https://developer.intacct.com/api/platform-services/dimensions/#list-dimensions
func GetDimensions() Function {
	return &Writer{Cmd: "getDimensions"}
}

// GetDimensionRelationships Lists all standard dimensions and UDDs and
// provides information about their to-one and to-many relationships to other dimensions.
// https://developer.intacct.com/api/platform-services/dimensions/#list-dimension-relationships
func GetDimensionRelationships() Function {
	return &Writer{Cmd: "getDimensionRelationships"}
}

// GetDimensionAutofillDetails provides information about auto-fill settings for to-one
// relationships between dimensions. (If the relationship is to-many, auto-fill is not available.)
// https://developer.intacct.com/api/platform-services/dimensions/#list-dimension-auto-fill-details
func GetDimensionAutofillDetails() Function {
	return &Writer{Cmd: "getDimensionAutofillDetails"}
}

// GetDimensionRestrictedData lists the IDs of related dimensions that
// are the target(s) of to-many relationships from a single source dimension.
// https://developer.intacct.com/api/platform-services/dimensions/#list-dimensions-restricted-data
func GetDimensionRestrictedData(dimension, value string) Function {
	var payload = struct {
		XMLName   xml.Name `xml:"DimensionValue"`
		Dimension string   `xml:"dimension"`
		Value     string   `xml:"value"`
	}{
		Dimension: dimension,
		Value:     value,
	}
	return &Writer{Cmd: "getDimensionRestrictedData", Payload: payload}
}

func getCmd(name string, payload interface{}) Function {
	return &Writer{
		Cmd:     "name",
		Payload: payload,
	}
}

// Relationship describes an Intacct dimension
type Relationship struct {
	Dimension       string     `xml:"dimension"`
	ObjectID        string     `xml:"object_id"`
	AutoFillRelated Bool       `xml:"autofillrelated"`
	EnableOverride  Bool       `xml:"enableoverride"`
	Related         []Related  `xml:"related"`
	AutoFill        []AutoFill `xml:"autofil"`
}

// Related describes relationships between dimensions
type Related struct {
	DimensionRelationship
	SourceSide  string `xml:"source_side"`
	RelatedSide string `xml:"related_side"`
}

// AutoFill describes auto fill relationships for a dimension
type AutoFill struct {
	DimensionRelationship
	Type string `xml:"type"`
}

// DimensionRelationship identifies a child relationship
type DimensionRelationship struct {
	Dimension      string `xml:"dimension"`
	ObjectID       string `xml:"object_id"`
	RelationshipID string `xml:"relationship_id"`
}

// Restriction lists the to-one restrictions for each dimension
type Restriction struct {
	Dimension    string                  `xml:"dimension"`
	ObjectID     string                  `xml:"object_id"`
	Restrictions []DimensionRelationship `xml:"restrictedby"`
}
