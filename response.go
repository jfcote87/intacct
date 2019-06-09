// Copyright 2019 James Cote
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package intacct

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"reflect"
	"time"
)

// Response contains function results from a Request
type Response struct {
	Ack      *Ack            `xml:"acknowledgement"` // used in asyc only
	Control  Control         `xml:"control"`
	Auth     *ResponseAuth   `xml:"operation>authentication"`
	ErrorMsg *ControlError   `xml:"errormessage>error"`
	OpError  *OperationError `xml:"operation>errormessage>error"`
	Results  []Result        `xml:"operation>result"`
}

// execErr returns the top level errors to indicate Exec error
func (r *Response) execErr() error {
	if r.ErrorMsg != nil {
		return r.ErrorMsg
	}
	if r.OpError != nil {
		return r.OpError
	}
	return nil
}

// Error returns error from a response object.
func (r *Response) Error() error {
	err := r.execErr()
	if err != nil {
		return err
	}
	var errResult = make([][]ErrorDetail, len(r.Results), len(r.Results))
	for idx, result := range r.Results {
		if len(result.Errors) > 0 {
			err = ResultsError(errResult)
		}
		errResult[idx] = result.Errors
	}
	return err
}

// Decode interrogates the Response returning errors encoded in the
// top section and Operation section.  Each Result is decoded into the
// corresponding returnValues interface.  Errors are tracked within a
// ResultsError struct.  If no errors are found/occur, nil is returned.
//
// returnValues must be a *[]Struct or *Struct type.
func (r *Response) Decode(returnValues ...interface{}) error {
	if err := r.execErr(); err != nil {
		return err
	}
	var errResult = make(ResultsError, len(r.Results), len(r.Results))
	hasError := false
	for idx, result := range r.Results {
		if len(result.Errors) > 0 {
			hasError = true
			errResult[idx] = result.Errors
			continue
		}
		// don't decode nil values
		if len(returnValues) < idx+1 || returnValues[idx] == nil {
			continue
		}
		if err := result.Decode(returnValues[idx]); err != nil {
			hasError = true
			errResult[idx] = []ErrorDetail{{ErrorNo: "", Description: "Decode Error", Err: err}}
		}
	}
	if hasError {
		return errResult
	}
	return nil
}

// Ack is returned for asynchronous calls.
// https://developer.intacct.com/web-services/sync-vs-async/
type Ack struct {
	Status string        `xml:"status"`
	Error  *ControlError `xml:"errormessage>error"`
}

// ResponseAuth returns the authentication result for a request
type ResponseAuth struct {
	Status           string    `xml:"status"`
	UserID           string    `xml:"userid"`
	CompanyID        string    `xml:"companyid"`
	SessionTimestamp time.Time `xml:"sessiontimestamp,omitempty"`
	SessionTimeout   time.Time `xml:"sessiontimeout,omitempty"`
}

func (ra *ResponseAuth) getTimeout() time.Time {
	if ra == nil {
		var t time.Time
		return t
	}
	return ra.SessionTimeout
}

// ControlError contains errors returned from intacct leading
// to a control failuer
// https://developer.intacct.com/web-services/error-handling/
type ControlError []ErrorDetail

// OperationError signifies that the error was returned in the operation
// section rather than the top level ResponseError
type OperationError ControlError

// Error fulfill error interface
func (e *ControlError) Error() string {
	if e != nil && len(*e) > 0 {
		return (*e)[0].errString("control")
	}
	return "No error"
}

func (e *OperationError) Error() string {
	if e != nil && len(*e) > 0 {
		return (*e)[0].errString("operation")
	}
	return "No error"

}

// ErrorDetail describes each error
type ErrorDetail struct {
	ErrorNo      string `xml:"errorno"`
	Description  string `xml:"description"`
	Description2 string `xml:"description2"`
	Correction   string `xml:"correction"`
	Err          error  `xml:"-"`
}

func (e ErrorDetail) errString(prefix string) string {
	if e.Err != nil {
		return fmt.Sprintf("%v", e.Err)
	}
	return fmt.Sprintf("%s ErrorNo: %s - %s - %s", prefix, e.ErrorNo, e.Description, e.Description2)
}

// Result wraps either an  or DataResult for a function call
type Result struct {
	Status    string        `xml:"status"`
	Function  string        `xml:"function"`
	ControlID string        `xml:"controlid"`
	ListType  *ListType     `xml:"listtype"`
	Errors    []ErrorDetail `xml:"errormessage>error"`
	Data      *ResultData   `xml:"data"`
}

// ListType describes the start/ending/remaining records from
// a v2.1 style call.
type ListType struct {
	Start int    `xml:"start,attr,omitempty"`
	End   int    `xml:"end,attr,omitempty"`
	Total int    `xml:"total,attr,omityempty"`
	Name  string `xml:",chardata"`
}

// ResultData of executing a function
type ResultData struct {
	ListType     string `xml:"listtype,attr"`
	Count        int    `xml:"count,attr"`
	TotalCount   int    `xml:"totalcount,attr"`
	NumRemaining int    `xml:"numremaining,attr"`
	ResultID     string `xml:"resultId,attr"`
	Payload      []byte `xml:",innerxml"`
}

// Decode unmarshals the results xml into dst.  dst must have
// a type of *[]S or *S.  If dst is not a pointer to a
// slice, only the first object in a list will be unmarshalled.
func (r Result) Decode(dst interface{}) error {
	if len(r.Errors) > 0 {
		return ResultsError([][]ErrorDetail{r.Errors})
	}
	if dst == nil {
		return nil
	}
	dv := reflect.ValueOf(dst)
	if dv.Kind() != reflect.Ptr || dv.IsNil() {
		return errors.New("expected a non-nil ptr")
	}

	dx := xml.NewDecoder(bytes.NewReader(r.Data.Payload))
	if dv = dv.Elem(); dv.Kind() == reflect.Slice {
		tk, err := dx.Token()
		for elementCnt := 0; err == nil; elementCnt++ {
			switch s := tk.(type) {
			case xml.StartElement:
				val := reflect.New(dv.Type().Elem()).Interface()
				if err = dx.DecodeElement(&val, &s); err != nil {
					return fmt.Errorf("%d: %v", elementCnt, err)
				}
				dv.Set(reflect.Append(dv, reflect.Indirect(reflect.ValueOf(val))))
			}
			tk, err = dx.Token()
		}
		if err == io.EOF {
			return nil
		}
		return err
	}
	return dx.Decode(dst)
}

// ResultsError contains an array of errors corresponding to the functions
// passed in Exec
type ResultsError [][]ErrorDetail

func (re ResultsError) Error() string {
	msg := bytes.NewBufferString("")
	var prefix string
	for idx, detail := range re {
		if len(detail) > 0 {
			if prefix > "" {
				prefix = fmt.Sprintf(" | result[%d]", idx)
			} else {
				prefix = fmt.Sprintf("result[%d]", idx)
			}
			msg.WriteString(detail[0].errString(prefix))
		}
	}
	return string(msg.Bytes())
}
