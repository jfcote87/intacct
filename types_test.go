// Copyright 2019 James Cote
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package intacct_test

import (
	"encoding/xml"
	"reflect"
	"testing"
	"time"

	"github.com/jfcote87/intacct"
)

var zeroTime time.Time

func TestDate_Val(t *testing.T) {
	var zeroTime time.Time
	var nowTime = time.Now()

	tests := []struct {
		name string
		dx   intacct.Date
		want *time.Time
	}{
		{name: "zero date", dx: intacct.TimeToDate(zeroTime), want: nil},
		{name: "now", dx: intacct.TimeToDate(nowTime), want: &nowTime},
		{name: "empty", want: nil},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		got := tt.dx.Val()
		want := tt.want
		if want == nil {
			if got != nil {
				t.Errorf("%s Date.Val() = %v, want nil", tt.name, *got)
			}
		} else if got == nil {

			t.Errorf("%s Date.Val() = nil, want %v", tt.name, *want)

		} else if *got != *want {
			t.Errorf("%s Date.Val() = %v, want %v", tt.name, *got, *want)
		}
	}

	tests2 := []struct {
		name string
		dx   intacct.Date
		want string
	}{
		{name: "nil", want: ""},
		{name: "zero", dx: intacct.TimeToDate(zeroTime), want: ""},
		{name: "now", dx: intacct.TimeToDate(nowTime), want: nowTime.Format("2006-01-02")},
	}
	for _, tt := range tests2 {
		if got := tt.dx.String(); got != tt.want {
			t.Errorf("%s Date.String() = %s, want %s", tt.name, got, tt.want)
		}
	}

}

func TestDatetime_Val(t *testing.T) {
	var zeroTime time.Time
	var nowTime = time.Now()

	tests := []struct {
		name string
		dt   intacct.Datetime
		want *time.Time
	}{
		{name: "zero date", dt: intacct.TimeToDatetime(zeroTime), want: nil},
		{name: "now", dt: intacct.TimeToDatetime(nowTime), want: &nowTime},
		{name: "empty", want: nil},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		got := tt.dt.Val()
		want := tt.want
		if want == nil {
			if got != nil {
				t.Errorf("%s Date.Val() = %v, want nil", tt.name, *got)
			}
		} else if got == nil {

			t.Errorf("%s Date.Val() = nil, want %v", tt.name, *want)

		} else if *got != *want {
			t.Errorf("%s Date.Val() = %v, want %v", tt.name, *got, *want)
		}
	}

	tests2 := []struct {
		name string
		dt   intacct.Datetime
		want string
	}{
		{name: "nil", want: ""},
		{name: "zero", dt: intacct.TimeToDatetime(zeroTime), want: ""},
		{name: "now", dt: intacct.TimeToDatetime(nowTime), want: nowTime.Format(time.RFC3339)},
	}
	for _, tt := range tests2 {
		if got := tt.dt.String(); got != tt.want {
			t.Errorf("%s Date.String() = %s, want %s", tt.name, got, tt.want)
		}
	}
}

var xmlString = `<top>
<Dt1>2019-04-28</Dt1>
<Dt2>04/28/2019</Dt2>
<Dt3></Dt3>
<Dtm1>12/31/2019 18:10:01</Dtm1>
<Dtm2>2019-12-31T18:10:01Z</Dtm2>
<Dtm3></Dtm3>
<I>5</I><I2></I2>
<F>10.2</F><F2></F2>
<B>true</B><B1>x</B1><B2></B2>
<CustomA>A Value</CustomA><CustomB>X</CustomB>
</top>`

type XMLTester struct {
	Dt1    intacct.Date     `xml:"Dt1,omitempty"`
	Dt2    intacct.Date     `xml:"Dt2,omitempty"`
	Dt3    intacct.Date     `xml:"Dt3,omitempty"`
	Dtm1   intacct.Datetime `xml:"Dtm1,omitempty"`
	Dtm2   intacct.Datetime `xml:"Dtm2,omitempty"`
	Dtm3   intacct.Datetime `xml:"Dtm3,omitempty"`
	I      intacct.Int
	I2     intacct.Int
	F      intacct.Float64
	F2     intacct.Float64
	B      intacct.Bool
	B1     intacct.Bool
	B2     intacct.Bool
	Custom []intacct.CustomField `xml:",any"`
}

func TestTypesXMLUnmarshal(t *testing.T) {

	dateTest := time.Date(2019, 4, 28, 0, 0, 0, 0, time.UTC)
	dtTest := time.Date(2019, 12, 31, 18, 10, 1, 0, time.UTC)

	var xt = XMLTester{}
	if err := xml.Unmarshal([]byte(xmlString), &xt); err != nil {
		t.Errorf("xml unmarshal fail: %v", err)
		return
	}

	var expectedValues = XMLTester{
		Dt1:    intacct.TimeToDate(dateTest),
		Dt2:    intacct.TimeToDate(dateTest),
		Dtm1:   intacct.TimeToDatetime(dtTest),
		Dtm2:   intacct.TimeToDatetime(dtTest),
		I:      5,
		I2:     0,
		F:      10.2,
		F2:     0,
		B:      true,
		B1:     false,
		B2:     false,
		Custom: []intacct.CustomField{{Name: "CustomA", Value: "A Value"}, {Name: "CustomB", Value: "X"}},
	}
	if !reflect.DeepEqual(xt, expectedValues) {
		t.Errorf("unmarshal intacct types wanted %#v; got %#v", expectedValues, xt)
	}
}
