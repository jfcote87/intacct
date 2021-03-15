// Copyright 2020 James Cote
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package intacct // github.com/intacct/query

import (
	"context"
	"encoding/xml"
	"fmt"
	"time"
)

// Query implements new intacct query and definition functionality that replaces
// the readByQuery and inspect functions with query and lookup. Stmt and Lookup
// structs are intacct functions that may be called by intacct.Service.
// intacct documentation may be found at:
// https://developer.intacct.com/web-services/queries/
type Query struct {
	XMLName         xml.Name      `xml:"query"`
	Object          string        `xml:"object"`
	Select          Select        `xml:"select"`
	Filter          *Filter       `xml:"filter,omitempty"`
	Sort            *QuerySort    `xml:"orderby,omitempty"`
	Options         *QueryOptions `xml:"options,omitempty"`
	PageSz          int           `xml:"pagesize,omitempty"`
	Offset          int           `xml:"offset,omitempty"`
	TransactionType string        `xml:"docparid,omitempty"`
	// ControlID used for transaction marking. Leave blank for
	// defaul behavior
	ControlID string `xml:"-"`
}

// GetControlID fulfills intacct.Function so may be used in
// Service Exec call
func (q Query) GetControlID() string {
	return q.ControlID
}

// GetAll reads all pages and unmarshals them into results.  resultSlice must be a pointer to a slice.
func (q Query) GetAll(ctx context.Context, sv *Service, resultSlice interface{}) error {
	pgsz := q.PageSz
	if pgsz == 0 {
		pgsz = 100
	}
	numRemaining := -1
	for numRemaining != 0 {
		resp, err := sv.Exec(ctx, q)
		if err != nil {
			return err
		}
		if err = resp.Decode(resultSlice); err != nil {
			return err
		}
		if len(resp.Results) == 0 || resp.Results[0].Data == nil {
			return fmt.Errorf("empty result returned")
		}
		numRemaining = resp.Results[0].Data.NumRemaining
		q.Offset += pgsz
	}
	return nil
}

// Select determines fields to return for query
type Select struct {
	Fields []string `xml:"field"`
	Count  string   `xml:"count,omitempty"`
	Avg    string   `xml:"avg,omitempty"`
	Min    string   `xml:"min,omitempty"`
	Max    string   `xml:"max,omitempty"`
	Sum    string   `xml:"sum,omitempty"`
}

type QuerySort struct {
	XMLName xml.Name  `xml:"orderby"`
	Fields  []OrderBy `xml:"order"`
}

// OrderBy describes sort conditions
type OrderBy struct {
	XMLName    xml.Name `xml:"order"`
	Field      string   `xml:"field,omitempty"`
	Descending bool     `xml:"descending,omitempty"`
}

// MarshalXML used to create <descending> tag
func (o OrderBy) MarshalXML(e *xml.Encoder, s xml.StartElement) error {
	e.EncodeToken(s)
	e.EncodeElement(o.Field, xml.StartElement{Name: xml.Name{Local: "field"}})
	if o.Descending {
		e.EncodeElement("", xml.StartElement{Name: xml.Name{Local: "descending"}})
	}
	e.EncodeToken(s.End())
	return nil
}

// QueryOptions set query flags
type QueryOptions struct {
	CaseInsensitive bool `xml:"caseinsensitive,omitempty"`
	ShowPrivate     bool `xml:"showprivate,omitempty"`
}

// NewFilter returns an initialized Filter pointer
func NewFilter() *Filter {
	return &Filter{}
}

// Filter is a heirarchy of criteria. Use function to add criteria
type Filter struct {
	XMLName xml.Name
	Field   string     `xml:"field,omitempty"`
	Value   FilterVals `xml:"value,omitempty"`
	Filters []Filter
}

// And creates a new And filter and adds it to the method receiver's filter list. The
// return value is the new And filter not the method receiver.
func (f *Filter) And() *Filter {
	return f.newFilter("and")
}

// Or creates a new OR filter and adds it to the method receiver's filter list. The
// return value is the new Or filter not the method receiver.
func (f *Filter) Or() *Filter {
	return f.newFilter("or")
}

func (f *Filter) newFilter(nm string) *Filter {
	ret := Filter{XMLName: xml.Name{Local: nm}}
	if f != nil {
		f.Filters = append(f.Filters, ret)
		return &f.Filters[len(f.Filters)-1]
	}
	return &ret
}

// FilterVals handles proper marsheling of empty strings
type FilterVals []string

// MarshalXML output nothing for empty slice, value elements for all others
func (fv FilterVals) MarshalXML(e *xml.Encoder, s xml.StartElement) error {
	for _, val := range fv {
		e.EncodeToken(s)
		e.EncodeToken(xml.CharData(val))
		e.EncodeToken(s.End())
	}
	return nil
}

func (f *Filter) add(nm, field string, values ...string) *Filter {
	if f == nil {
		f = &Filter{}
	}
	f.Filters = append(f.Filters, Filter{
		XMLName: xml.Name{Local: nm},
		Field:   field,
		Value:   values,
	})
	return f
}

// EqualTo adds an equal filter for the field and value to f's list of filters.  The
// receiver value f is returned to allow chaining
func (f *Filter) EqualTo(field string, value string) *Filter {
	return f.add("equalto", field, value)
}

// NotEqualTo adds a not equal filter for the field and value to f's list of filters.  The
// receiver value f is returned to allow chaining
func (f *Filter) NotEqualTo(field string, value string) *Filter {
	return f.add("notequalto", field, value)
}

// LessThan adds a less than filter for the field and value to f's list of filters.  The
// receiver value f is returned to allow chaining
func (f *Filter) LessThan(field string, value string) *Filter {
	return f.add("lessthan", field, value)
}

// LessThanOrEqualTo adds a less than or equal to filter for the field and value to f's list of filters.  The
// receiver value f is returned to allow chaining
func (f *Filter) LessThanOrEqualTo(field string, value string) *Filter {
	return f.add("lessthanorequalto", field, value)
}

// GreaterThan adds a greater than filter for the field and value to f's list of filters.  The
// receiver value f is returned to allow chaining
func (f *Filter) GreaterThan(field string, value string) *Filter {
	return f.add("greaterthan", field, value)
}

// GreaterThanOrEqualTo adds a less than or equal to filter for the field and value to f's list of filters.  The
// receiver value f is returned to allow chaining
func (f *Filter) GreaterThanOrEqualTo(field string, value string) *Filter {
	return f.add("greaterthanorequalto", field, value)
}

// Between adds a date range filter for the field and begin and end dates to f's list of filters.  The
// receiver value f is returned to allow chaining
func (f *Filter) Between(field string, start, end time.Time) *Filter {
	return f.add("between", field, start.Format("01/02/2006"), end.Format("01/02/2006"))
}

// In adds a list filter for the field and values to f's list of filters.  The
// receiver value f is returned to allow chaining
func (f *Filter) In(field string, values ...string) *Filter {
	return f.add("in", field, values...)
}

// NotIn adds a list filter for the field and values to f's list of filters.  The
// receiver value f is returned to allow chaining
func (f *Filter) NotIn(field string, values ...string) *Filter {
	return f.add("notin", field, values...)
}

// Like adds a string filter for the field and values to f's list of filters.  The
// receiver value f is returned to allow chaining
func (f *Filter) Like(field string, value string) *Filter {
	return f.add("like", field, value)
}

// NotLike adds a string filter for the field and values to f's list of filters.  The
// receiver value f is returned to allow chaining
func (f *Filter) NotLike(field string, value string) *Filter {
	return f.add("notlike", field, value)
}

// IsNull adds a null value filter for the field and values to f's list of filters.  The
// receiver value f is returned to allow chaining
func (f *Filter) IsNull(field string) *Filter {
	return f.add("isnull", field)
}

// IsNotNull adds a null value filter for the field and values to f's list of filters.  The
// receiver value f is returned to allow chaining
func (f *Filter) IsNotNull(field string) *Filter {
	return f.add("isnotnull", field)
}

// ObjectRelationship describes relationship between objects
type ObjectRelationship struct {
	Path      string `xml:"OBJECTPATH"`
	Name      string `xml:"OBJECTNAME"`
	Lable     string `xml:"LABEL"`
	Type      string `xml:"RELATIONSHIPTYPE"`
	RelatedBy string `xml:"RELATEDBY"`
}

// ObjectField defines parameters of an intacct object (table)
type ObjectField struct {
	ID          string   `xml:"ID"`
	Label       string   `xml:"LABEL"`
	Description string   `xml:"DESCRIPTION"`
	Required    bool     `xml:"REQUIRED"`
	ReadOnly    bool     `xml:"READONLY"`
	DataType    string   `xml:"DATATYPE"`
	IsCustom    bool     `xml:"ISCUSTOM"`
	ValidValues []string `xml:"VALIDVALUES>VALIDVALUE"`
}

// ObjectType is top level response for lookup function
type ObjectType struct {
	XMLName       xml.Name             `xml:"Type"`
	Name          string               `xml:"Name,attr"`
	Type          string               `xml:"DocumentType,attr"`
	Fields        []ObjectField        `xml:"Fields>Field"`
	Relationships []ObjectRelationship `xml:"Relationships>Relationship"`
}

// Lookup returns an object definition
type Lookup struct {
	XMLName    xml.Name `xml:"lookup"`
	ObjectName string   `xml:"object"`
	ControlID  string   `xml:"-"`
}

// GetControlID fulfills intacct.Function so may be used in
// Service Exec call
func (l Lookup) GetControlID() string {
	return l.ControlID
}
