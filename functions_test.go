// Copyright 2019 James Cote
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package intacct_test

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/jfcote87/testutils"

	"github.com/jfcote87/intacct"
)

func TestReader(t *testing.T) {

	var rdrTests = []struct {
		Rdr  *intacct.Reader
		Name string
		Flds []intacct.CustomField
	}{
		{
			Rdr:  intacct.Read("VENDOR", "1", "2").Fields("A", "B", "C"),
			Name: "read",
			Flds: []intacct.CustomField{
				{Name: "object", Value: "VENDOR"},
				{Name: "keys", Value: "1,2"},
				{Name: "fields", Value: "A,B,C"},
				{Name: "returnFormat", Value: "xml"},
			},
		},
		{
			Rdr:  intacct.Read("VENDOR", ""),
			Name: "read",
			Flds: []intacct.CustomField{
				{Name: "object", Value: "VENDOR"},
				{Name: "keys", Value: ""},
				{Name: "fields", Value: "*"},
				{Name: "returnFormat", Value: "xml"},
			},
		},
		{
			Rdr:  intacct.ReadByName("VENDOR", "A").PageSize(100),
			Name: "readByName",
			Flds: []intacct.CustomField{
				{Name: "object", Value: "VENDOR"},
				{Name: "keys", Value: "A"},
				{Name: "fields", Value: "*"},
				{Name: "returnFormat", Value: "xml"},
			},
		},
		{
			Rdr:  intacct.ReadByQuery("VENDOR", "").Fields("fld1", "fld2"),
			Name: "readByQuery",
			Flds: []intacct.CustomField{
				{Name: "object", Value: "VENDOR"},
				{Name: "query", Value: ""},
				{Name: "fields", Value: "fld1,fld2"},
				{Name: "returnFormat", Value: "xml"},
			},
		},
		{
			Rdr:  intacct.ReadByQuery("VENDOR", "A > B").PageSize(100),
			Name: "readByQuery",
			Flds: []intacct.CustomField{
				{Name: "object", Value: "VENDOR"},
				{Name: "query", Value: "A > B"},
				{Name: "fields", Value: "*"},
				{Name: "returnFormat", Value: "xml"},
				{Name: "pagesize", Value: "100"},
			},
		},
		{
			Rdr:  intacct.ReadRelated("asset", "Rasset_class"),
			Name: "readRelated",
			Flds: []intacct.CustomField{
				{Name: "object", Value: "asset"},
				{Name: "keys", Value: ""},
				{Name: "fields", Value: "*"},
				{Name: "returnFormat", Value: "xml"},
				{Name: "relationship_id", Value: "Rasset_class"},
			},
		},
	}
	for idx, tt := range rdrTests {
		var keyRdr = struct {
			CustomFlds []intacct.CustomField `xml:",any"`
		}{}
		b, _ := xml.Marshal(tt.Rdr)
		if !strings.HasPrefix(string(b), "<"+tt.Name) {
			t.Errorf("test #%d expected element name <%s>; got %s", idx, tt.Name, strings.Split(string(b), ">")[0]+">")
			continue
		}
		if err := xml.Unmarshal(b, &keyRdr); err != nil {
			t.Errorf("unable to unmarshal test %d", idx)
			return
		}
		if !cmpCustomFields(tt.Flds, keyRdr.CustomFlds) {
			t.Errorf("test %d expected %#v; got %#v", idx, tt.Flds, keyRdr.CustomFlds)
		}
	}
}

func TestReadAll(t *testing.T) {
	testTransport := &testutils.Transport{}
	testTransport.Add(
		&testutils.RequestTester{
			ResponseFunc: func(req *http.Request) (*http.Response, error) {
				return testutils.MakeResponse(200, []byte(readMore1), xmlHeader), nil
			},
		},
		&testutils.RequestTester{
			ResponseFunc: func(r *http.Request) (*http.Response, error) {
				var iReq *Request
				defer r.Body.Close()
				if err := xml.NewDecoder(r.Body).Decode(&iReq); err != nil {
					return testutils.MakeResponse(http.StatusBadRequest, []byte(err.Error()), nil), nil
				}
				var res = struct {
					ResultID string `xml:"resultId"`
				}{}
				xml.Unmarshal([]byte(iReq.Op.Content[0].Payload), &res)
				if res.ResultID != "READMOREID" {
					return nil, fmt.Errorf("expected resultId = READMOREID; got %s", res.ResultID)
				}
				return testutils.MakeResponse(200, []byte(readMore2), xmlHeader), nil
			},
		},
	)
	testClient := &http.Client{
		Transport: testTransport,
	}
	ctx := context.Background()
	sv := &intacct.Service{
		SenderID:      "SENDERID",
		Password:      "*******",
		Authenticator: intacct.SessionID("SESSIONID"),
		HTTPClientFunc: func(ctx context.Context) (*http.Client, error) {
			return testClient, nil
		},
	}
	var projects []Project
	if err := intacct.ReadByQuery("PROJECT", "PROJECTID LIKE 'P%'").PageSize(10).GetAll(ctx, sv, &projects); err != nil {
		t.Errorf("readAll failed: %v", err)
	}
	if len(projects) != 12 {
		t.Errorf("expected 12 Project records; got %d", len(projects))
	}
}

func cmpCustomFields(a, b []intacct.CustomField) bool {
	if len(a) != len(b) {
		return false
	}
	var tMap = make(map[string]string)
	for _, cf := range a {
		tMap[cf.Name] = cf.Value
	}
	for _, cf := range b {
		v, ok := tMap[cf.Name]
		if !ok || v != cf.Value {
			return false
		}
	}
	return true
}

func TestWriter(t *testing.T) {
	var wTests = []struct {
		w  *intacct.Writer
		nm string
	}{
		{
			w:  intacct.Create("PROJECT", Project{ProjectID: "PX", Projectname: "Name1"}),
			nm: "create",
		},
		{
			w:  intacct.Update("", []Project{{ProjectID: "PX", Projectname: "Name1"}, {ProjectID: "PY", Projectname: "Name2"}}),
			nm: "update",
		},
		{
			w:  intacct.Delete("NOTPROJECT", Project{RecordNumber: 1}),
			nm: "delete",
		},
		{
			w:  intacct.GetAPISession("Loc1").(*intacct.Writer),
			nm: "getAPISession",
		},
		{
			w:  intacct.GetDimensions().(*intacct.Writer),
			nm: "getDimensions",
		},
		{
			w:  intacct.GetDimensionAutofillDetails().(*intacct.Writer),
			nm: "getDimensionAutofillDetails",
		},
		{
			w:  intacct.GetDimensionRelationships().(*intacct.Writer),
			nm: "getDimensionRelationships",
		},
		{
			w:  intacct.GetDimensionRestrictedData("dim", "val").(*intacct.Writer),
			nm: "getDimensionRestrictedData",
		},
		{
			w:  intacct.InstallApp("App Definition").(*intacct.Writer),
			nm: "installApp",
		},
	}
	for idx, tt := range wTests {
		b, _ := xml.Marshal(tt.w)
		if !strings.HasPrefix(string(b), "<"+tt.nm) {
			t.Errorf("test #%d expected element name <%s>; got %s", idx, tt.nm, strings.Split(string(b), ">")[0]+">")
			continue
		}
	}
}
