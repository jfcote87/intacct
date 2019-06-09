// Copyright 2019 James Cote
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package intacct

import (
	"encoding/xml"
)

// Request is a batch the struct sent to Intacct.
type Request struct {
	XMLName xml.Name  `xml:"request"`
	Control Control   `xml:"control"`
	Op      Operation `xml:"operation"`
}

// Control provides a header to a request
type Control struct {
	SenderID          string `xml:"senderid,omitempty"`
	Password          string `xml:"password,omitemtpy"`
	ControlID         string `xml:"controlid,omitempty"`
	UniqueID          bool   `xml:"uniqueid"`
	DTDVersion        string `xml:"dtdversion"`
	PolicyID          string `xml:"policyid,omitempty"`
	Debug             bool   `xml:"debug,omitempty"`
	Includewhitespace bool   `xml:"includewhitespace"`
	Status            string `xml:"status,omitempty"`
}

// Operation marshals into an intacct request
type Operation struct {
	Transaction  bool              `xml:"transaction,attr,omitempty"`
	Auth         interface{}       `xml:"authentication"`
	CompanyPrefs []Preference      `xml:"preference/companyprefs/companypref"`
	ModulePrefs  []Preference      `xml:"preference/moduleprefs/modulepref"`
	Content      []RequestFunction `xml:"content>function"`
}

// RequestFunction wraps function.
type RequestFunction struct {
	ControlID string `xml:"controlid,attr"`
	Payload   interface{}
}

// Preference add additional data to an operation.
type Preference struct {
	Application string `xml:"application"`
	Name        string `xml:"preference"`
	Value       string `xml:"prefvalue"`
}
