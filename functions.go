// Copyright 2019 James Cote
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package intacct

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
)

// Function defines an action.  See
// https://developer.intacct.com/web-services/functions/
type Function interface {
	GetControlID() string
}

// Reader may be a read, readByName, readQuery or readMore function.
// Use the Read, ReadByName, ReadByQuery or ReadMore functions to
// create function rather than creating directly.
type Reader struct {
	XMLName      xml.Name
	Object       string  `xml:"object,omitempty"`       // Object name
	Keys         *string `xml:"keys,omitempty"`         // comma sep list of keys for read and readByName
	Query        *string `xml:"query,omitempty"`        // query statement for readQuery
	FieldList    string  `xml:"fields,omitempty"`       // field list
	MaxRecs      int     `xml:"pagesize,omitempty"`     // max items returned
	ReturnFormat string  `xml:"returnFormat,omitempty"` // xml for now
	Docparid     string  `xml:"docparid,omitempty"`     // don't know what this is
	Relationship string  `xml:"relationship_id,omitempty"`
	ResultID     string  `xml:"resultId,omitempty"`

	controlID string
}

var (
	readXMLName        = xml.Name{Local: "read"}
	readByQueryXMLName = xml.Name{Local: "readByQuery"}
	readByNameXMLName  = xml.Name{Local: "readByName"}
	readMoreXMLName    = xml.Name{Local: "readMore"}
	readRelatedXMLName = xml.Name{Local: "readRelated"}
	readReturnFormat   = "xml"
	readAllFields      = "*"
)

// Read returns a Reader to read specific keys.  If no keys
// are passed, the first 100 records are returned in an
// unspecified order.
func Read(objectName string, keys ...string) *Reader {
	var keyvals = strings.Join(keys, ",")
	return &Reader{
		XMLName:      readXMLName,
		Object:       objectName,
		Keys:         &keyvals,
		FieldList:    readAllFields,
		ReturnFormat: readReturnFormat,
	}
}

// ReadByName returns a Reader to read specific name keys.  If no keys
// are passed, the first 100 records are returned in an
// unspecified order.
func ReadByName(objectName string, keys ...string) *Reader {
	var keyvals = strings.Join(keys, ",")
	return &Reader{
		XMLName:      readByNameXMLName,
		Object:       objectName,
		Keys:         &keyvals,
		FieldList:    readAllFields,
		ReturnFormat: readReturnFormat,
	}
}

// ReadByQuery returns a Reader based upon the passed query string which is an
// SQL-like query based on fields on the object. Illegal XML characters must be
// properly encoded. The following SQL operators are supported: <, >, >=, <=, =,
// like, not like, in, not in. When doing NULL comparisons: IS NOT NULL, IS NULL.
// Multiple fields may be matched using the AND and OR operators. Joins are not
// supported. Single quotes in any operands must be escaped with a backslash -
// For example, the value Erik's Deli would become 'Erik\'s Deli'
func ReadByQuery(objectName string, qry string) *Reader {
	return &Reader{
		XMLName:      readByQueryXMLName,
		Object:       objectName,
		Query:        &qry,
		FieldList:    readAllFields,
		ReturnFormat: readReturnFormat,
	}
}

// ReadMore returns a Reader to retrieve remaining records of
// a ReadByQuery
func ReadMore(resultID string) *Reader {
	return &Reader{
		XMLName:  readMoreXMLName,
		ResultID: resultID,
	}
}

// ReadRelated retrieves records related to on or more records by a
// given relationship.  Note: this only works on custom objects.
// see https://developer.intacct.com/api/platform-services/records/#get-related-records
func ReadRelated(objectName string, relationshipName string, keys ...string) *Reader {
	var keyvals = strings.Join(keys, ",")
	return &Reader{
		XMLName:      readRelatedXMLName,
		Object:       objectName,
		Keys:         &keyvals,
		Relationship: relationshipName,
		FieldList:    readAllFields,
		ReturnFormat: readReturnFormat,
	}
}

// Fields sets the fields to return.  If not set all
// fields are returned.  Do not use with ReadMore type.
func (r *Reader) Fields(fields ...string) *Reader {
	if r != nil {
		if r.XMLName.Local != readMoreXMLName.Local {
			r.FieldList = strings.Join(fields, ",")
		}
	}
	return r
}

// PageSize sets the max number of records returned
//
// if pageSize is not set, 100 is assumed
func (r *Reader) PageSize(numOfRecs int) *Reader {
	if r.XMLName.Local == readByQueryXMLName.Local {
		r.MaxRecs = numOfRecs
	}
	return r
}

// SetControlID sets a unique identifier for the call
func (r *Reader) SetControlID(controlID string) *Reader {
	r.controlID = controlID
	return r
}

// GetControlID returns the unique identifier for the call
func (r Reader) GetControlID() string {
	return r.controlID
}

// GetAll reads all records for a query.  The reader must be a readByQuery
// or readMore type.  resultSlice should be of type *[]<Object>.
func (r Reader) GetAll(ctx context.Context, sv *Service, resultSlice interface{}) error {
	if r.XMLName.Local != readByQueryXMLName.Local && r.XMLName.Local != readMoreXMLName.Local {
		return fmt.Errorf("GetAll not allowed on %s", r.XMLName.Local)
	}
	rptr := &r
	for rptr != nil {
		resp, err := sv.Exec(ctx, rptr)
		if err != nil {
			return err
		}
		if err = resp.Decode(resultSlice); err != nil {
			return err
		}
		if len(resp.Results) > 0 && resp.Results[0].Data != nil && resp.Results[0].Data.NumRemaining > 0 {
			rptr = ReadMore(resp.Results[0].Data.ResultID)
		} else {
			rptr = nil
		}
	}
	return nil
}

// Writer is used to create functions such as create, update, and deleted.
// For these Intacct functions, use the Create, Update and Delete funcs.  See
// CmdGetApiSession definition for an example of how to use Write to implement
// other functions.
type Writer struct {
	// Cmd names the top level element.  If empty, Payload is marshalled directly
	Cmd string `xml:"-"`
	// Payload may not be nil if Cmd is empty
	Payload interface{}
	// If not empty, ObjectName will override Payload's XMLName
	ObjectName string `xml:"-"`
	controlID  string
}

// MarshalXML customizes xml output for Writer
func (w Writer) MarshalXML(e *xml.Encoder, s xml.StartElement) error {
	s.Name.Local = w.Cmd
	s.Name.Space = ""
	s.Attr = nil
	err := e.EncodeToken(s) // encode Cmd element
	if err != nil {
		return err
	}
	if w.Payload != nil {
		if err = w.encodePayload(e); err != nil {
			return err
		}
	}
	return e.EncodeToken(xml.EndElement{Name: s.Name})
}

// only call if w.Payload != nil
func (w *Writer) encodePayload(e *xml.Encoder) error {
	if w.ObjectName == "" {
		return e.Encode(w.Payload)
	}
	return e.EncodeElement(w.Payload, xml.StartElement{Name: xml.Name{Local: w.ObjectName}})
}

// Create returns a Writer function to create object(s) in payload
func Create(objectName string, payload interface{}) *Writer {
	return &Writer{
		Cmd:        "create",
		ObjectName: objectName,
		Payload:    payload,
	}
}

// Update returns a Writer function to update object(s) in payload. Payload
// must contain a key for the intacct object
func Update(objectName string, payload interface{}) *Writer {
	return &Writer{
		Cmd:        "update",
		ObjectName: objectName,
		Payload:    payload,
	}
}

// Delete returns a Writer function to delete object(s) in payload. Payload
// must contain a key for the intacct object
func Delete(objectName string, payload interface{}) *Writer {
	return &Writer{
		Cmd:        "delete",
		ObjectName: objectName,
		Payload:    payload,
	}
}

// SetControlID sets a unique identifier for the call
func (w *Writer) SetControlID(controlID string) *Writer {
	w.controlID = controlID
	return w
}

// GetControlID returns the unique identifier for the call
func (w Writer) GetControlID() string {
	return w.controlID
}

// GetAPISession returns Intacct Function for obtaining a SessionID for
// for the passed location (blank location is the top-level company). Decode
// the Response into a SessionResult struct.
// https://developer.intacct.com/api/company-console/api-sessions/#get-api-session
func GetAPISession(location string) Function {
	var loc = struct {
		XMLName xml.Name `xml:"location_id"`
		Loc     string   `xml:",innerxml"`
	}{
		Loc: location,
	}
	return &Writer{Cmd: "getAPISession", Payload: loc}
}

// InstallApp installs a Platform Services definition
// https://developer.intacct.com/api/platform-services/applications/#install-application
func InstallApp(definition string) Function {
	var def = struct {
		XMLName xml.Name `xml:"appxml"`
		Def     string   `xml:",cdata"`
	}{
		Def: definition,
	}
	return &Writer{Cmd: "installApp", Payload: def}
}

// GetFinancialSetup provides basic financial setup information for a
// company such as its base currency, first fiscal month,
// multi-currency setting, and so forth.
// https://developer.intacct.com/api/company-console/financial-setup/#get-financial-setup
func GetFinancialSetup() Function {
	return &Writer{Cmd: "etFinancialSetup"}
}

// Inspector performs a inspection macro returning the definition
// of the named Object.  For a list of all objects, set Object to "*".
type Inspector struct {
	XMLName   xml.Name `xml:"inspect"`
	IsDetail  int      `xml:"detail,attr,omitempty"` // set to 1 for detail
	Object    string   `xml:"object"`
	controlID string
}

// ObjectFields returns function to list an objects fields.  If showDetail,
// intacct returns an InspectDetailResult else an InspectResult.
func ObjectFields(objectName string, showDetail bool) *Inspector {
	var detVal = 0
	if showDetail {
		detVal = 1
	}
	return &Inspector{
		IsDetail: detVal,
		Object:   objectName,
	}
}

// ObjectList returns an Inspector function that
// return a []InspectName of all objects.
func ObjectList() *Inspector {
	return &Inspector{Object: "*"}
}

// GetControlID returns ControlID for function.
func (i *Inspector) GetControlID() string {
	return i.controlID
}

// SetControlID set the ControlID for function.
func (i *Inspector) SetControlID(id string) {
	i.controlID = id
}

// InspectName is the name listing from a full inspect listing.
type InspectName struct {
	TypeName string `xml:"typename,attr"`
	Name     string `xml:",chardata"`
}

// InspectDetailResult is the full description of an Intactt object via an
// Inspect function call.
type InspectDetailResult struct {
	XMLName      xml.Name      `xml:"Type"`
	Name         string        `xml:"Name,attr"`
	SingularName string        `xml:"Attributes>SingularName"`
	PluralName   string        `xml:"Attributes>PluralName"`
	Description  string        `xml:"Attributes>Description"`
	Fields       []FieldDetail `xml:"Fields>Field"`
}

// FieldDetail is the description of each field of an Intacct object
type FieldDetail struct {
	Name             string `xml:"Name"`
	GroupName        string `xml:"GroupName"`
	DataName         string `xml:"dataName"`
	ExternalDataName string `xml:"externalDataName"`
	IsRequired       bool   `xml:"isRequired"`
	IsReadOnly       bool   `xml:"isReadOnly"`
	MaxLen           string `xml:"maxLength"`
	DisplayLabel     string `xml:"DisplayLabel"`
	Description      string `xml:"Description"`
	ID               string `xml:"id"`
	Relationship     string `xml:"relationship"`
	RelatedObject    string `xml:"relatedObject"`
}

// InspectResult lists all fields for an object (name only).
type InspectResult struct {
	XMLName xml.Name `xml:"Type"`
	Name    string   `xml:"Name,attr"`
	Field   []string `xml:"Fields>Field"`
}

// LegacyFunction is used for v2.1 style calls (get_list, get, etc.).  See examples
// in v21 package.
type LegacyFunction struct {
	Payload   interface{}
	controlID string
}

// MarshalXML only encodes the Payload field
func (n LegacyFunction) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if n.Payload == nil {
		return errors.New("no payload specified")
	}
	return e.Encode(n.Payload)
}

// GetControlID is needed to fulfill the Function interface
func (n LegacyFunction) GetControlID() string {
	return n.controlID
}
