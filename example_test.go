// Copyright 2019 James Cote
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package intacct_test

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"log"

	"github.com/jfcote87/intacct"
	v21 "github.com/jfcote87/intacct/v21"
)

// Example Config file.
var sessionConfig = `{
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

type Project struct {
	RecordNumber      intacct.Int `xml:"RECORDNO,omitempty"`
	ProjectID         string      `xml:"PROJECTID"`
	Projectname       string      `xml:"NAME"`
	Description       string      `xml:"DESCRIPTION,omitempty"`
	ParentProjectKey  intacct.Int `xml:"PARENTKEY,omitempty"`
	ParentProjectID   string      `xml:"PARENTID,omitempty"`
	ParentProjectName string      `xml:"PARENTNAME,omitempty"`
}

// ExampleService_Exec demonstrates using an intacct service to read
// a project record. It then creates a new project record as a child
// of the first record, and finally runs a query
// reading all projects having orignal project as a parent.
func ExampleService_Exec() {
	var projectName = "EX1732"
	var ctx context.Context = context.Background()

	configReader := bytes.NewReader([]byte(sessionConfig))
	sv, err := intacct.ServiceFromConfigJSON(configReader)
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	var projectRecord *Project
	resp, err := sv.Exec(ctx, intacct.ReadByName("PROJECT", projectName))
	if err != nil {
		log.Fatalf("Execution error: %v", err)
	}
	if err = resp.Decode(&projectRecord); err != nil {
		log.Fatalf("result error: %v", err)
	}

	// empty result set
	if projectRecord == nil {
		log.Fatalf("No project named %s exists", projectName)
	}

	// Create new projects
	parentNo := projectRecord.RecordNumber

	resp, err = sv.Exec(ctx, intacct.Create("PROJECT", []Project{
		{
			ProjectID:        "X2017",
			Description:      "New sub-project of E1732",
			ParentProjectKey: parentNo,
		},
		{
			ProjectID:        "X2018",
			Description:      "New sub-project of E1732",
			ParentProjectKey: parentNo,
		},
	}))
	if err == nil {
		// pass nil in decode to only check for errors
		err = resp.Decode(nil)
	}
	if err != nil {
		log.Fatalf("createProject execution: %v", err)
	}

	var projectList []Project
	// ReadByQuery to read all projects having parent of p. Using GetAll() to return all pages of results
	if err = intacct.
		ReadByQuery("PROJECT", fmt.Sprintf("PARENTKEY = '%d'", parentNo)).
		PageSize(10).
		GetAll(ctx, sv, &projectList); err != nil {
		log.Fatalf("query full read error: %v", err)
	}
	fmt.Printf("Total children: %d\n", len(projectList))
}

// Read documents with attachments
// https://developer.intacct.com/api/company-console/attachments/#list-attachments-legacy
func Example_read_supdoc() {
	var ctx context.Context = context.Background()

	configReader := bytes.NewReader([]byte(sessionConfig))
	sv, err := intacct.ServiceFromConfigJSON(configReader)
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}
	// Read documents with attachment data
	resp, err := sv.Exec(ctx, intacct.LegacyFunction{
		Payload: &v21.GetList{
			ObjectName: "supdoc",
			MaxItems:   1,
			Filter: &v21.Expression{
				Type:  v21.ExpressionEqual,
				Field: "supdocid",
				Value: "DocumentID",
			},
		},
	})

	if err != nil {
		log.Fatalf("exec err: %v", err)
	}
	var supDoc = struct {
		XMLName           xml.Name         `xml:"supdoc"`
		SupdocID          string           `xml:"supdocid"`
		SupdocName        string           `xml:"supdocname,omitempty"`
		SupdocfolderName  string           `xml:"supdocfoldername,omitempty"`
		Supdocdescription string           `xml:"supdocdescription,omitempty"`
		Attachments       []v21.Attachment `xml:"attachments>attachment,omitempty"`
	}{}
	if err = resp.Decode(&supDoc); err != nil {
		log.Fatalf("decode error: %v", err)
	}
	for idx, a := range supDoc.Attachments {
		log.Printf("Document %s: attachment %d %s total bytes %d", supDoc.SupdocName, idx, a.AttachmentName, len(a.AttachmentData))
	}
}

// create document with attachments
// https://developer.intacct.com/api/company-console/attachments/#create-attachment-legacy
func Example_create_doc() {
	var ctx context.Context = context.Background()

	configReader := bytes.NewReader([]byte(sessionConfig))
	sv, err := intacct.ServiceFromConfigJSON(configReader)
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}
	var buffer []byte
	// copy file bytes to buffer
	_, err = sv.Exec(ctx, intacct.LegacyFunction{
		Payload: &v21.CreateSupdoc{
			SupdocID:         "NewDocumentID",
			SupdocName:       "Doc Name",
			SupdocfolderName: "FolderName",
			Attachments: []v21.Attachment{
				{
					AttachmentName: "New PDF File",
					AttachmentType: "pdf",
					AttachmentData: buffer,
				},
			},
		},
	})

	if err != nil {
		log.Fatalf("create supdoc failed: %v", err)
	}
	log.Printf("attachment added")
}

// ExampleReader_readall shows how to read
// all responses to a Reader function
func ExampleReader_readall() {
	var ctx context.Context = context.Background()

	configReader := bytes.NewReader([]byte(sessionConfig))
	sv, err := intacct.ServiceFromConfigJSON(configReader)
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}
	var results []Vendor
	if err = intacct.ReadByQuery("VENDOR", "").
		Fields("RECORDNO", "VENDORID").
		GetAll(ctx, sv, &results); err != nil {
		log.Fatalf("getall error: %v", err)
	}
	log.Printf("Total Records: %d", len(results))
}

// ExampleQuery_GetAll demonstrates using an intacct service to query
// records using the new query function. GetAll reads all pages of the
// query.
func ExampleQuery_GetAll() {
	var ctx context.Context = context.Background()

	configReader := bytes.NewReader([]byte(sessionConfig))
	sv, err := intacct.ServiceFromConfigJSON(configReader)
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	var projects []Project

	var filter *intacct.Filter

	//filter.EqualTo("STATUS", "active").In("PARENTID", "ID01", "ID02")
	var stmt = &intacct.Query{
		Object: "PROJECT",
		Select: intacct.Select{
			Fields: []string{"RECORDNO", "PROJECTID", "NAME", "DESCRIPTION", "PARENTNAME"},
		},
		OrderBy: []intacct.OrderBy{{Field: "PROJECTID"}},
		Filter:  filter.EqualTo("STATUS", "active").In("PARENTID", "ID01", "ID02"),
	}

	if err := stmt.GetAll(ctx, sv, &projects); err != nil {
		log.Printf("read error %v", err)
		return
	}
	for _, p := range projects {
		fmt.Printf("%s", p.Projectname)
	}
}
