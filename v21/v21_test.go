// Copyright 2019 James Cote
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package v21_test

import (
	"encoding/xml"
	"testing"
	"time"

	"github.com/jfcote87/intacct"
	v21 "github.com/jfcote87/intacct/v21"
)

func TestGetList(t *testing.T) {
	var payload = &v21.GetList{
		ObjectName: "bill",
		Fields:     &[]string{"key", "vendorid", "ponumber", "datecreated"},
		MaxItems:   10,
		Filter: &v21.Expression{
			Type: v21.ExpressionLogicalAnd,
			Expressions: []v21.Expression{
				{
					Type:  v21.ExpressionGreaterThan,
					Field: "ponumber",
					Value: "0",
				},
				{
					Type:  v21.ExpressionEqual,
					Field: "vendorid",
					Value: "12345",
				},
			},
		},
	}
	b, err := xml.Marshal(payload)
	if err != nil {
		t.Errorf("err encoding function: %v", err)
		return
	}
	// tested this string in Postman
	if string(b) != `<get_list object="bill" maxitems="10"><filter><logical logical_operator="and"><expression><field>ponumber</field><operator>&gt;</operator><value>0</value></expression><expression><field>vendorid</field><operator>=</operator><value>12345</value></expression></logical></filter><fields><field>key</field><field>vendorid</field><field>ponumber</field><field>datecreated</field></fields></get_list>` {
		t.Errorf("invalid encoding: %s", b)
		return
	}
}

func TestDateYMD(t *testing.T) {
	var xmlcode = `<a><datecreated><year>2018</year><month>02</month><day>10</day></datecreated><datecreated></datecreated></a>`

	var tx = struct {
		XMLName     xml.Name      `xml:"a"`
		DateCreated []v21.DateYMD `xml:"datecreated"`
	}{}

	if err := xml.Unmarshal([]byte(xmlcode), &tx); err != nil {
		t.Errorf("unmarshal error: %v", err)
		return
	}
	tm1 := time.Date(2018, time.Month(2), 10, 0, 0, 0, 0, time.UTC)
	var expectedValues = []*time.Time{&tm1, nil}
	if len(tx.DateCreated) != 2 {
		t.Errorf("expected 2 return values; got %d", len(tx.DateCreated))
	}
	for idx, dt := range tx.DateCreated {
		v1, v2 := dt.Val(), expectedValues[idx]
		if v1 == nil {
			if v2 != nil {
				t.Errorf("test %d expected %v; got nil", idx, *v2)
			}
		} else if v2 == nil {
			t.Errorf("test %d expected nil; got %v", idx, *v1)
		} else if *v1 != *v2 {
			t.Errorf("test %d expected %v; got %v", idx, *v2, *v1)
		}
	}

	var expectedXML = []string{
		"",
		"<DateYMD><year>2018</year><month>2</month><day>10</day></DateYMD>",
		"<DateYMD><year>2019</year><month>12</month><day>31</day></DateYMD>",
	}
	var tmzero time.Time
	for idx, val := range []time.Time{tmzero, tm1, time.Date(2019, time.Month(12), 31, 0, 0, 0, 0, time.UTC)} {

		dx := v21.TimeToDateYMD(val)
		b, _ := xml.Marshal(dx)
		if expectedXML[idx] != string(b) {
			t.Errorf("test %d expected %s; got %s", idx, expectedXML[idx], string(b))

		}
	}
}

func TestGetSupdoc(t *testing.T) {
	createDoc := &intacct.LegacyFunction{
		Payload: &v21.CreateSupdoc{
			SupdocID:          "NameID of Doc",
			SupdocName:        "docName",
			Supdocdescription: "",
			SupdocfolderName:  "Folder",
			Attachments: []v21.Attachment{
				v21.Attachment{
					AttachmentName: "doc1",
					AttachmentType: "txt",
					AttachmentData: []byte("file bytes"),
				},
			},
		},
	}
	b, err := xml.Marshal(createDoc)
	if err != nil {
		t.Errorf("unable to marshal createsupdoc: %v", err)
	}

	var testStruct = struct {
		XMLName xml.Name `xml:"create_supdoc"`
		Name    string   `xml:"attachments>attachment>attachmentname"`
		Type    string   `xml:"attachments>attachment>attachmenttype"`
		Data    string   `xml:"attachments>attachment>attachmentdata"`
	}{}

	if err = xml.Unmarshal(b, &testStruct); err != nil {
		t.Errorf("unmarshal create_supdoc: %v", err)
		return
	}
	// wanting base64 standard encoding`
	if testStruct.Name != "doc1" || testStruct.Type != "txt" || testStruct.Data != "ZmlsZSBieXRlcw==" {
		t.Errorf("expected name: doc1, type: txt, data: ZmlsZSBieXRlcw==; got %s %s %s", testStruct.Name, testStruct.Type, testStruct.Data)
	}

	var resp *intacct.Response

	if err = xml.Unmarshal([]byte(responseGetSupdoc), &resp); err != nil {
		t.Errorf("responseGetSupdoc xml decode failed: %v", err)
		return
	}
	var supDoc = struct {
		XMLName           xml.Name         `xml:"supdoc"`
		SupdocID          string           `xml:"supdocid"`
		SupdocName        string           `xml:"supdocname,omitempty"`
		SupdocfolderName  string           `xml:"supdocfoldername,omitempty"`
		Supdocdescription string           `xml:"supdocdescription,omitempty"`
		Attachments       []v21.Attachment `xml:"attachments>attachment,omitempty"`
	}{}
	if err = xml.Unmarshal(resp.Results[0].Data.Payload, &supDoc); err != nil {
		t.Errorf("responseGetSupdoc result xml decode failed: %v", err)
		return
	}

	if string(supDoc.Attachments[0].AttachmentData) != "Hello World!" || string(supDoc.Attachments[1].AttachmentData) != "File Text" {
		t.Errorf("expect attachments to read \"Hello World!\" and \"File Text\"; got %s, %s", supDoc.Attachments[0].AttachmentData, supDoc.Attachments[1].AttachmentData)
	}

}

const responseGetSupdoc = `<?xml version="1.0" encoding="UTF-8"?>
<response>
    <control>
        <status>success</status>
        <senderid>LibertyFund</senderid>
        <controlid>1559492683</controlid>
        <uniqueid>false</uniqueid>
        <dtdversion>3.0</dtdversion>
    </control>
    <operation>
        <authentication>
            <status>success</status>
            <userid>xml_gateway</userid>
            <companyid>SENDERID</companyid>
            <locationid>None</locationid>
            <sessiontimestamp>2019-06-02T09:24:43-07:00</sessiontimestamp>
            <sessiontimeout>2019-06-02T15:24:43-07:00</sessiontimeout>
        </authentication>
        <result>
            <status>success</status>
            <function>get_list</function>
            <controlid>abcdef</controlid>
            <listtype start="0" end="0" total="1">supdoc</listtype>
            <data>
                <supdoc>
                    <recordno>14043</recordno>
                    <supdocid>HotelContract10704</supdocid>
                    <supdocname>Contract Detail</supdocname>
                    <folder>Contracts</folder>
                    <description>invoice number : A04724</description>
                    <creationdate>04/22/2019</creationdate>
                    <createdby>Anybody</createdby>
                    <attachments>
                        <attachment>
                            <attachmentname>TestDoc1</attachmentname>
                            <attachmenttype>txt</attachmenttype>
                            <attachmentdata>SGVsbG8gV29ybGQh</attachmentdata>
                        </attachment>
                        <attachment>
                            <attachmentname>TestDoc2</attachmentname>
                            <attachmenttype>txt</attachmenttype>
                            <attachmentdata>RmlsZSBUZXh0</attachmentdata>
                        </attachment>
                    </attachments>
                </supdoc>
            </data>
        </result>
    </operation>
</response>`
