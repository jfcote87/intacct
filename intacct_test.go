// Copyright 2019 James Cote
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package intacct_test

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/jfcote87/ctxclient"
	"github.com/jfcote87/intacct"
	"github.com/jfcote87/testutils"
)

type ResponseErrorTest struct {
	FN        string
	IsSuccess bool
	IsOpErr   bool
	ErrorNo   string
}

func TestResponse(t *testing.T) {
	for _, re := range []ResponseErrorTest{
		{"execXMLError.xml", false, false, "XL03000003"},
		{"execLoginInvalid.xml", false, true, "XL03000006"},
		{"execFuncError.xml", true, false, "BL34000061"},
		{"execProjectSuccess.xml", true, false, ""},
	} {
		rx, err := decodeFile("testfiles/" + re.FN)
		if err != nil {
			t.Fatalf("unable to open: testfiles/%s", re.FN)
		}
		var isOpError bool
		var isSuccess bool
		var details intacct.ErrorDetail

		switch e2 := rx.Error().(type) {
		case nil:
			isSuccess = true
			details = intacct.ErrorDetail{ErrorNo: ""}
			var pList []Project
			var p Project
			if err = rx.Decode(&pList, &p); err != nil {
				t.Fatalf("%s expected no error; got %v", re.FN, err)
			}
			if len(pList) != 5 {
				t.Fatalf("%s expected list of 5 projects; got %v", re.FN, pList)
			}
			if p.RecordNumber != 1423 {
				t.Errorf("%s expected Result[1] to have RecordNumber = 1423; got %v", re.FN, p.RecordNumber)
			}
		case *intacct.OperationError:
			isOpError = true
			details = (*e2)[0]
		case *intacct.ControlError:
			details = (*e2)[0]
		case intacct.ResultsError:
			isSuccess = true
			_ = e2[0][0]
			details = e2[0][0]
		default:
			t.Errorf("%v", reflect.ValueOf(e2).Type())
			t.Errorf("%s expected success = %v and Errorno \"%s\"; got error %v", re.FN, re.IsSuccess, re.ErrorNo, e2)
			continue
		}
		if re.IsSuccess != isSuccess || re.ErrorNo != details.ErrorNo {
			t.Errorf("%s expected success = %v and Errorno \"%s\"; got success = %v and ErrorNo = %q", re.FN, re.IsSuccess, re.ErrorNo, isSuccess, details.ErrorNo)
		}
		if isOpError != re.IsOpErr {
			t.Errorf("%s expeced isOpError = %v; got %v", re.FN, re.IsOpErr, isOpError)
		}
	}
}

func decodeFile(fn string) (*intacct.Response, error) {
	b, err := ioutil.ReadFile(fn)
	if err != nil {
		return nil, err
	}
	var rx *intacct.Response
	return rx, xml.Unmarshal(b, &rx)
}

type TX struct{}

func (t TX) RoundTrip(rq *http.Request) (*http.Response, error) {
	b, err := ioutil.ReadAll(rq.Body)
	if err != nil {
		return nil, err
	}
	return nil, errors.New(string(b))
}

func TestService(t *testing.T) {
	var tests = []struct {
		sv  *intacct.Service
		msg string
		ctx context.Context
	}{
		{sv: nil, msg: "nil Service"},
		{sv: &intacct.Service{}, msg: "nil Authenticator"},
		{sv: &intacct.Service{Authenticator: intacct.SessionID("")}, msg: "SendorID/Passowrd is empty"},
		{sv: &intacct.Service{SenderID: "A", Password: "P", Authenticator: intacct.SessionID("")}, msg: "nil context"},
		{sv: &intacct.Service{SenderID: "A", Password: "P", Authenticator: intacct.SessionID("")}, msg: "no functions specified", ctx: context.TODO()},
	}
	for _, tt := range tests {
		if _, err := tt.sv.Exec(tt.ctx); err == nil || err.Error() != tt.msg {
			t.Errorf("expected %s; got %v", tt.msg, err)
		}
	}
}

func TestExec(t *testing.T) {
	invalidSessionPayload, _ := ioutil.ReadFile("testfiles/sessionInvalid.xml")
	testTransport := &testutils.Transport{}
	testTransport.Add(
		&testutils.RequestTester{
			ResponseFunc: func(r *http.Request) (*http.Response, error) {
				return nil, errors.New("local server error")
			},
		},
		&testutils.RequestTester{
			Method:   "POST",
			Response: testutils.MakeResponse(500, []byte("Server Error"), nil),
		},
		&testutils.RequestTester{
			Method:   "POST",
			Response: testutils.MakeResponse(200, invalidSessionPayload, nil),
		},
	)
	sv := &intacct.Service{
		SenderID: "AAAA",
		Password: "BBBB",

		Authenticator: &intacct.Login{
			UserID:     "UID",
			Password:   "PWD",
			Company:    "Company",
			LocationID: "None",
		},
		HTTPClientFunc: func(ctx context.Context) (*http.Client, error) {
			return &http.Client{
				Transport: testTransport,
			}, nil
		},
	}

	ctx := context.Background()
	var f intacct.Function = &intacct.Inspector{}
	_, err := sv.Exec(ctx, f)
	if ex, ok := err.(*url.Error); !ok || ex.Err.Error() != "local server error" {
		t.Fatalf("expected &url.Error of local server error; got %v", err)
	}
	_, err = sv.Exec(ctx, f)
	if _, ok := err.(*ctxclient.NotSuccess); !ok {
		t.Fatalf("expected &ctxclient.NotSuccess{StatusCode:500...; got %v", err)
	}
	_, err = sv.Exec(ctx, f)
	if _, ok := err.(*intacct.OperationError); !ok {
		t.Errorf("expected &intacct.ResponseError{intacct.ErrorDetail{ErrorNo:\"XL03000006\",... ; got %v", err)
	}
}

func TestExec_MultiFunc(t *testing.T) {
	functionErrorSuccessPayload, _ := ioutil.ReadFile("testfiles/functionErrorSuccess.xml")
	vendorResponsePayload, _ := ioutil.ReadFile("testfiles/vendorResponse.xml")
	testTransport := &testutils.Transport{}
	testTransport.Add(
		&testutils.RequestTester{
			Method:   "POST",
			Response: testutils.MakeResponse(200, vendorResponsePayload, nil),
		},
		&testutils.RequestTester{
			Method:   "POST",
			Response: testutils.MakeResponse(200, functionErrorSuccessPayload, nil),
		},
	)

	sv := &intacct.Service{
		SenderID: "AAAA",
		Password: "BBBB",

		Authenticator: &intacct.Login{
			UserID:     "UID",
			Password:   "PWD",
			Company:    "Company",
			LocationID: "None",
		},
		HTTPClientFunc: func(ctx context.Context) (*http.Client, error) {
			return &http.Client{
				Transport: testTransport,
			}, nil
		},
	}

	ctx := context.Background()
	f1 := intacct.Read("VENDOR")
	f2 := intacct.ReadByQuery("VENDOR", "PARENTKEY = '1234'")
	resp, err := sv.Exec(ctx, f1, f2)
	if err != nil {
		t.Fatalf("expected success; got %v", err)
	}
	var vendors []Vendor
	var vendTest *Vendor
	err = resp.Decode(&vendors, &vendTest)
	if err != nil {
		t.Fatalf("vendor response: err %v", err)
	}
	if len(vendors) != 2 {
		t.Errorf("expected 2 vendors; got %d", len(vendors))
	}
	if vendTest == nil {
		t.Fatalf("nil decode")
	}
	if vendTest.RecordNumber.Val() != 29181 {
		t.Fatalf("expected RECORD 29181; got %d", vendTest.RecordNumber.Val())
	}

	f1 = intacct.Read("GLDETAIL").Fields("X")
	f2 = intacct.Read("APBILL", "16747")
	f3 := intacct.Read("GLDETAIL").Fields("Y")
	resp, err = sv.Exec(ctx, f1, f2, f3)
	if err != nil {
		t.Fatalf("expected success; got %v", err)
	}
	err = resp.Error()
	switch ex := err.(type) {
	case nil:
		t.Errorf("expected ResultsError; got nil")
	case intacct.ResultsError:
		if len(ex) != 3 {
			t.Errorf("expected 3 errors; got %d", len(ex))
			return
		}
		if ex[1] != nil || ex[0] == nil || ex[2] == nil {
			t.Errorf("expected 1st result with error, 2nd with nil err; 3rd with error; got %v", ex)
		}
		msgs := strings.Split(ex.Error(), "|")
		if len(msgs) != 2 || !strings.HasPrefix(msgs[0], "result[0] Error") || !strings.HasPrefix(strings.Trim(msgs[1], " "), "result[2] Error") {
			t.Errorf("expected error msg to show 2 errors; got %s", ex.Error())
		}

	default:
		t.Errorf("expected ResultsError; got %#v", ex)
	}
}

var xmlHeader = http.Header{"Content-Type": {"application/xml"}}

func TestSession(t *testing.T) {
	ctx := context.TODO()
	cx := ctxclient.Func(func(ctx context.Context) (*http.Client, error) {
		return &http.Client{Transport: getTestSessionTransport()}, nil
	})

	sv, err := intacct.ServiceFromConfigJSON(bytes.NewReader([]byte(tSessionConfig)),
		intacct.ConfigHTTPClientFunc(cx))
	if err != nil {
		t.Fatalf("unable to build service from json: %v", err)
	}

	f := &intacct.Inspector{}
	resp, err := sv.Exec(ctx, f)
	if err != nil {
		t.Fatalf("session refresh: %v", err)
	}
	var ix []intacct.InspectName
	if err = resp.Decode(&ix); err != nil {
		t.Fatalf("session refresh decode: %v", err)
	}
	if len(ix) != 3 {
		t.Errorf("expected 3 inspectNames; got %d", len(ix))
	}
}

var testSessionCounter = 0

func getTestSessionTransport() *testutils.Transport {
	var makeResponse = func(r *http.Request) (*http.Response, error) {
		testSessionCounter++
		var iReq *Request
		defer r.Body.Close()
		if err := xml.NewDecoder(r.Body).Decode(&iReq); err != nil {
			return testutils.MakeResponse(http.StatusBadRequest, []byte(err.Error()), nil), nil
		}
		var respBytes []byte
		var err error
		auth := iReq.Op.Auth
		if testSessionCounter == 1 {
			// expect login not sessionID
			if auth.UserID != "xml_gateway" || auth.Password != "User Password" || auth.SessionID > "" {
				return testutils.MakeResponse(http.StatusBadRequest,
					[]byte(fmt.Sprintf("expected Login: xml_gateway and Password: User Password; got %s: %s", auth.UserID, auth.Password)),
					nil), nil
			}
			if respBytes, err = getSessionResult("First SESSIONID"); err != nil {
				return nil, err
			}
		} else {
			if auth.UserID > "" || auth.Password > "" || auth.SessionID != "First SESSIONID" {
				return testutils.MakeResponse(http.StatusBadRequest,
					[]byte(fmt.Sprintf("expected First SESSIONID; got %s %s %s", auth.SessionID, auth.UserID, auth.Password)),
					nil), nil
			}
			if respBytes, err = getInspectResult(); err != nil {
				return nil, err
			}
		}
		return testutils.MakeResponse(200, respBytes, xmlHeader), nil
	}
	testTransport := &testutils.Transport{}
	testTransport.Add(
		&testutils.RequestTester{
			ResponseFunc: makeResponse,
		},
		&testutils.RequestTester{
			ResponseFunc: makeResponse,
		},
	)
	return testTransport
}

func TestControlConfig(t *testing.T) {
	const fctrlID = "ABCDEFGH"
	testTransport := &testutils.Transport{}
	// "CTXCtrlID", fctrlID
	var makeResponse = func(str1, str2 string) func(r *http.Request) (*http.Response, error) {
		return func(r *http.Request) (*http.Response, error) {
			var iReq *Request
			defer r.Body.Close()
			if err := xml.NewDecoder(r.Body).Decode(&iReq); err != nil {
				return testutils.MakeResponse(http.StatusBadRequest, []byte(err.Error()), nil), nil
			}
			if iReq.Control.ControlID != str1 {
				return testutils.MakeResponse(http.StatusBadRequest, []byte(fmt.Sprintf("expected ControlID = %s; got %s", str1, iReq.Control.ControlID)), nil), nil
			}
			cid := iReq.Op.Content[0].ControlID
			if cid != str2 {
				return testutils.MakeResponse(http.StatusBadRequest, []byte(fmt.Sprintf("expected Function ControlID = %s; got %s", str2, cid)), nil), nil
			}
			respBytes, err := getInspectResult()
			if err != nil {
				return nil, err
			}
			return testutils.MakeResponse(200, respBytes, xmlHeader), nil
		}
	}

	testTransport.Add(
		&testutils.RequestTester{
			ResponseFunc: makeResponse("CTXCtrlID", fctrlID),
		},
		&testutils.RequestTester{
			ResponseFunc: makeResponse("CONFIG_CTRLID", "CONFIG_CTRLID"),
		})

	sv := &intacct.Service{
		SenderID:      "MYID",
		Password:      "*****",
		Authenticator: intacct.SessionID("SESSIONID"),
		HTTPClientFunc: func(ctx context.Context) (*http.Client, error) {
			return &http.Client{
				Transport: testTransport,
			}, nil
		},
		ControlIDFunc: func(ctx context.Context) string {
			s, _ := ctx.Value("CTRLKEY").(string)
			return s
		},
	}
	ctx := context.WithValue(context.Background(), "CTRLKEY", "CTXCtrlID")
	f := intacct.ObjectList()
	f.SetControlID(fctrlID)
	if _, err := sv.Exec(ctx, f); err != nil {
		t.Errorf("ControlID function test: %v", err)
	}
	cc := &intacct.ControlConfig{
		IsTransaction: true,
		IsUnique:      true,
		Debug:         true,
		ControlID:     "CONFIG_CTRLID",
		PolicyID:      "POLICY",
	}
	f.SetControlID("")
	if _, err := sv.ExecWithControl(ctx, cc, f); err != nil {
		t.Errorf("ExecWithControl: %v", err)
	}

}

func TestErr(t *testing.T) {
	var resp *intacct.Response
	err := xml.Unmarshal([]byte(responseErrOperation), &resp)
	if err != nil {
		t.Errorf("response parse: %v", err)
	}
	err = resp.Error()
	if err == nil || !strings.HasPrefix(err.Error(), "operation") {
		t.Errorf("expected operation error; got %#v", err)
	}

	err = xml.Unmarshal([]byte(responseErrControl), &resp)
	if err != nil {
		t.Errorf("response parse: %v", err)
	}
	err = resp.Error()
	if err == nil || !strings.HasPrefix(err.Error(), "control") {
		t.Errorf("expected control error; got %#v", err)
	}
}
