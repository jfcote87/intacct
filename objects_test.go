// Copyright 2019 James Cote
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package intacct_test

import (
	"bytes"
	"encoding/xml"
	"text/template"
	"time"

	"github.com/jfcote87/intacct"
)

type Vendor struct {
	XMLName          xml.Name
	RecordNumber     intacct.Int `xml:"RECORDNO,omitempty"`
	VendorID         string      `xml:"VENDORID,omitempty"` // Required
	VendorName       string      `xml:"NAME,omitempty"`     // Required
	F1099Name        string      `xml:"NAME1099,omitempty"`
	ParentKey        string      `xml:"PARENTKEY,omitempty"`
	ParentVendor     string      `xml:"PARENTID,omitempty"`
	ParentVendorName string      `xml:"PARENTNAME,omitempty"`
}

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
	Auth         Authentication    `xml:"authentication"`
	CompanyPrefs []Preference      `xml:"preference/companyprefs/companypref"`
	ModulePrefs  []Preference      `xml:"preference/moduleprefs/modulepref"`
	Content      []RequestFunction `xml:"content>function"`
}

type Authentication struct {
	UserID     string `xml:"login>userid" json:"user_id"`
	Company    string `xml:"login>companyid" json:"company"`
	Password   string `xml:"login>password" json:"password"`
	ClientID   string `xml:"login>clientid,omitempty" json:"client_id,omitempty"`
	LocationID string `xml:"login>locationid,omitempty" json:"location_id,omitempty"`
	SessionID  string `xml:"sessionid"`
}

// RequestFunction wraps function.
type RequestFunction struct {
	ControlID string `xml:"controlid,attr"`
	Payload   string `xml:",innerxml"`
}

// Preference add additional data to an operation.
type Preference struct {
	Application string `xml:"application"`
	Name        string `xml:"preference"`
	Value       string `xml:"prefvalue"`
}

const tSessionConfig = `{
	"sender_id": "Your SenderID",
	"sender_pwd": "Your Password",
	"login": {
		"user_id": "xml_gateway",
		"company": "Company Name",
		"password": "User Password",
		"location_id": "XYZ"
	},
	"session": {}
}`

const tmplResponse = `<?xml version="1.0" encoding="UTF-8"?>
<response>
    <control>
        <status>success</status>
        <senderid>SENDERID</senderid>
        <controlid>1559244370</controlid>
        <uniqueid>false</uniqueid>
        <dtdversion>3.0</dtdversion>
    </control>
    <operation>
        <authentication>
            <status>success</status>
            <userid>xml_gateway</userid>
            <companyid>Your Company</companyid>
            <locationid>{{.Loc}}</locationid>
            <sessiontimestamp>{{.TmStamp}}</sessiontimestamp>
            <sessiontimeout>{{.TmOut}}</sessiontimeout>
        </authentication>
        {{ .Result }}
    </operation>
</response>`

const tmplGetApiResult = `<result>
<status>success</status>
<function>getAPISession</function>
<controlid>ac0d0d92-e449-4858-9a0c-720416fdec4b</controlid>
<data>
	<api>
		<sessionid>{{.SessionID}}</sessionid>
		<endpoint>https://test.url</endpoint>
		<locationid>{{.Loc}}</locationid>
	</api>
</data>
</result>`

const tmplGetinspectResult = `
        <result>
            <status>success</status>
            <function>inspect</function>
            <controlid>testFunctionId</controlid>
            <data listtype="All" count="1">
                <type typename="APADJUSTMENT">AP Adjustment</type>
                <type typename="APADJUSTMENTITEM">AP Adjustment Detail</type>
                <type typename="APBILL">AP Bill</type>
            </data>
		</result>
`

func getSessionResult(sessionID string) ([]byte, error) {
	tmstamp := time.Now()
	rtmpl := template.Must(template.New("resp").Parse(tmplResponse))
	stmpl := template.Must(template.New("getAPISessionResonse").Parse(tmplGetApiResult))

	buff := &bytes.Buffer{}
	if err := stmpl.Execute(buff, map[string]string{
		"Loc":       "XYZ",
		"SessionID": "First SESSIONID",
	}); err != nil {
		return nil, err
	}
	result := buff.Bytes()
	buff = &bytes.Buffer{}
	if err := rtmpl.Execute(buff, map[string]string{
		"Loc":     "XYZ",
		"TmOut":   tmstamp.Add(time.Hour).Format(time.RFC3339),
		"TmStamp": tmstamp.Format(time.RFC3339),
		"Result":  string(result),
	}); err != nil {
		return nil, err
	}
	return buff.Bytes(), nil
}

func getInspectResult() ([]byte, error) {
	tmstamp := time.Now()
	rtmpl := template.Must(template.New("resp").Parse(tmplResponse))

	buff := &bytes.Buffer{}
	if err := rtmpl.Execute(buff, map[string]string{
		"Loc":     "XYZ",
		"TmOut":   tmstamp.Add(time.Hour).Format(time.RFC3339),
		"TmStamp": tmstamp.Format(time.RFC3339),
		"Result":  tmplGetinspectResult,
	}); err != nil {
		return nil, err
	}
	return buff.Bytes(), nil
}

var readMore1 = `<?xml version="1.0" encoding="UTF-8"?>
<response>
    <control>
        <status>success</status>
        <senderid>SENDERID</senderid>
        <controlid>1559419337</controlid>
        <uniqueid>false</uniqueid>
        <dtdversion>3.0</dtdversion>
    </control>
    <operation>
        <authentication>
            <status>success</status>
            <userid>xml_gateway</userid>
            <companyid>Company</companyid>
            <sessiontimestamp>2019-06-01T13:02:17-07:00</sessiontimestamp>
            <sessiontimeout>2019-06-01T19:02:17-07:00</sessiontimeout>
        </authentication>
        <result>
            <status>success</status>
            <function>readByQuery</function>
            <controlid>testFunctionId</controlid>
            <data listtype="project" count="10" totalcount="12" numremaining="2" resultId="READMOREID">
                <project>
                    <PROJECTID>P01</PROJECTID>
                    <NAME>Exhibit - DC</NAME>
                </project>
                <project>
                    <PROJECTID>S02</PROJECTID>
                    <NAME>Exhibit DFW</NAME>
                </project>
                <project>
                    <PROJECTID>S03</PROJECTID>
                    <NAME>Exhibit LGA</NAME>
                </project>
                <project>
                    <PROJECTID>S04</PROJECTID>
                    <NAME>Exhibit DEN</NAME>
                </project>
                <project>
                    <PROJECTID>S05</PROJECTID>
                    <NAME>Exhibit SFO</NAME>
                </project>
                <project>
                    <PROJECTID>S06</PROJECTID>
                    <NAME>Exhibit LAX</NAME>
                </project>
                <project>
                    <PROJECTID>S07</PROJECTID>
                    <NAME>Exhibit ORD</NAME>
                </project>
                <project>
                    <PROJECTID>S08</PROJECTID>
                    <NAME>Exhibit ACK</NAME>
                </project>
                <project>
                    <PROJECTID>S09</PROJECTID>
                    <NAME>Exhibit BOS</NAME>
                </project>
                <project>
                    <PROJECTID>S10</PROJECTID>
                    <NAME>Exhibit MCI</NAME>
                </project>
            </data>
        </result>
    </operation>
</response>`

const readMore2 = `<?xml version="1.0" encoding="UTF-8"?>
<response>
    <control>
        <status>success</status>
        <senderid>SENDERID</senderid>
        <controlid>1559419454</controlid>
        <uniqueid>false</uniqueid>
        <dtdversion>3.0</dtdversion>
    </control>
    <operation>
        <authentication>
            <status>success</status>
            <userid>xml_gateway</userid>
            <companyid>Company</companyid>
            <sessiontimestamp>2019-06-01T13:04:14-07:00</sessiontimestamp>
            <sessiontimeout>2019-06-01T19:04:14-07:00</sessiontimeout>
        </authentication>
        <result>
            <status>success</status>
            <function>readMore</function>
            <controlid>testFunctionId</controlid>
            <data listtype="project" count="2" totalcount="12" numremaining="0" resultId="READMOREID">
                <project>
                    <PROJECTID>S11</PROJECTID>
                    <NAME>Exhibit CLE</NAME>
                </project>
                <project>
                    <PROJECTID>S12</PROJECTID>
                    <NAME>Exhibit OKC</NAME>
                </project>
            </data>
        </result>
    </operation>
</response>`

var responseErrControl = `<?xml version="1.0" encoding="UTF-8"?>
<response>
    <control>
        <status>failure</status>
        <senderid>SENDERID</senderid>
        <controlid>1559435393</controlid>
        <uniqueid>false</uniqueid>
        <dtdversion>3.0</dtdversion>
    </control>
    <errormessage>
        <error>
            <errorno>XL03000006</errorno>
            <description></description>
            <description2>Incorrect Intacct XML Partner ID or password.</description2>
            <correction></correction>
        </error>
    </errormessage>
</response>`

var responseErrOperation = `<response>
<control>
	<status>success</status>
	<senderid>test_sender_id</senderid>
	<controlid>hello_world</controlid>
	<uniqueid>false</uniqueid>
	<dtdversion>3.0</dtdversion>
</control>
<operation>
	<authentication>
		<status>failure</status>
		<userid>test_user_id</userid>
		<companyid>test_company_id</companyid>
	</authentication>
	<errormessage>
		<error>
			<errorno>XL03000006</errorno>
			<description>Invalid Web Services Authorization</description>
			<description2>The sender ID &#039;test_sender_id&#039; is not authorized to make Web Services requests to company ID &#039;test_company_id&#039;.</description2>
			<correction>Contact the company administrator to grant Web Services authorization to this sender ID.</correction>
		</error>
	</errormessage>
</operation>
</response>`
