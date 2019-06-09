// Copyright 2019 James Cote
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package v21_test

import (
	"bytes"
	"context"
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

// Example demonstrates how to execute a
// version 2.1 call using a LegacyFunction struct.
func Example_getList() {
	var ctx context.Context = context.Background()

	sv, err := intacct.ServiceFromConfigJSON(bytes.NewReader([]byte(sessionConfig)))
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	getListOfBillsWithPONumber := &intacct.LegacyFunction{
		Payload: &v21.GetList{
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
		},
	}
	resp, err := sv.Exec(ctx, getListOfBillsWithPONumber)
	if err != nil {
		log.Fatalf("Exec failed %v", err)
	}
	var records []intacct.ResultMap
	if err = resp.Decode(&records); err != nil {
		log.Fatalf("result err: %v", err)
	}
	for _, record := range records {
		log.Printf("%s %s %s",
			record.String("ponumber"),
			record.Date("datecreated").Format("2016 Jan 01"),
			record.String("supdocid"))
	}
}

// Example demonstrates how to read supporting documents
func Example_getList_supdoc() {
	var ctx context.Context = context.Background()

	sv, err := intacct.ServiceFromConfigJSON(bytes.NewReader([]byte(sessionConfig)))
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	getListOfBillsWithSupportingDocs := &intacct.LegacyFunction{
		Payload: &v21.GetList{
			ObjectName: "bill",
			Fields:     &[]string{"key", "vendorid", "ponumber", "datecreated"},
			MaxItems:   10,
			Filter: &v21.Expression{
				Type: v21.ExpressionLogicalAnd,
				Expressions: []v21.Expression{
					{
						Type:  v21.ExpressionGreaterThan,
						Field: "ponumber",
						Value: "",
					},
					{
						Type:  v21.ExpressionEqual,
						Field: "vendorid",
						Value: "12345",
					},
				},
			},
		},
	}
	resp, err := sv.Exec(ctx, getListOfBillsWithSupportingDocs)
	if err != nil {
		log.Fatalf("Exec failed %v", err)
	}
	var records []intacct.ResultMap
	if err = resp.Decode(&records); err != nil {
		log.Fatalf("result err: %v", err)
	}
	for _, record := range records {
		log.Printf("%s %s %s",
			record.String("ponumber"),
			record.Date("datecreated").Format("2016 Jan 01"),
			record.String("supdocid"))
	}
}
