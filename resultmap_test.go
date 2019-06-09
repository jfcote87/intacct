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

func TestResultMap(t *testing.T) {
	var tdata = `<VENDOR id="abc">
	<RECORDNO>1234</RECORDNO>
	<NAME>Name 1</NAME>
	<NAME>Name 2</NAME>
	<CONTACTS>
	<CONTACT id="123"><NAME>Contact1</NAME><CITY>Carmel</CITY></CONTACT>
	<CONTACT id="124"><NAME>Contact2</NAME><CITY>Indianapolis</CITY></CONTACT>
	<CONTACT id="125"><NAME>Contact3</NAME><CITY>Indianapolis</CITY></CONTACT>
	</CONTACTS>
	<DATE0>2018-13-31</DATE0>
	<DATE1>2018-11-25</DATE1>
	<DATE2>11-25-2018</DATE2>
	<DATE3>
		<Year>2018</Year>
		<Month>11</Month>
		<Day>25</Day>
	</DATE3>
	<INT0>A</INT0>
	<INT1>98</INT1>
	<INT2>99.2</INT2>
	<FLOAT0>A</FLOAT0>
	<FLOAT1>1254.2558</FLOAT1>
	<FLOAT2>-89.08</FLOAT2>
	<TIMESTAMP0>11/25/2018 05:54:21</TIMESTAMP0>
	<TIMESTAMP1>2018-11-25T05:54:21Z</TIMESTAMP1>
	<BOOL0>true</BOOL0>
	<BOOL1>1</BOOL1>
	<BOOL2>X</BOOL2>
	<BOOL3>Y</BOOL3>
	<STRINGARR>A</STRINGARR>
	<STRINGARR>B</STRINGARR>
	<STRINGARR>C</STRINGARR>
	</VENDOR>`
	var rm = make(intacct.ResultMap)
	err := xml.Unmarshal([]byte(tdata), &rm)
	if err != nil {
		t.Fatalf("unmarshal resultMap failed %v", err)
	}

	testDate := time.Date(2018, time.Month(11), 25, 0, 0, 0, 0, time.UTC)
	testDateTm := time.Date(2018, time.Month(11), 25, 5, 54, 21, 0, time.UTC)

	var tests = []struct {
		nm       string
		value    interface{}
		expected interface{}
		isPtr    bool
	}{
		{
			nm:       `StringArray("NAME")`,
			value:    rm.StringArray("NAME"),
			expected: []string{"Name 1", "Name 2"},
		},
		{
			nm:       `Date("DATE0")`,
			value:    isNilDt(rm.Date("DATE0")),
			expected: nil,
		},
		{
			nm:       `Date("DATE1")`,
			value:    isNilDt(rm.Date("DATE1")),
			expected: testDate,
		},
		{
			nm:       `Date("DATE2")`,
			value:    isNilDt(rm.Date("DATE2")),
			expected: testDate,
		},
		{
			nm:       `Date("DATE3")`,
			value:    isNilDt(rm.Date("DATE3")),
			expected: testDate,
		},
		{
			nm:       `Int("INT0")`,
			value:    rm.Int("INT0"),
			expected: int64(0),
		},
		{
			nm:       `Int("INT1")`,
			value:    rm.Int("INT1"),
			expected: int64(98),
		},
		{
			nm:       `Int("INT2")`,
			value:    rm.Int("INT2"),
			expected: int64(99),
		},
		{
			nm:       `Float("Float0")`,
			value:    rm.Float("FLOAT0"),
			expected: float64(0),
		},
		{
			nm:       `Float("Float1")`,
			value:    rm.Float("FLOAT1"),
			expected: float64(1254.2558),
		},
		{
			nm:       `Float("Float2")`,
			value:    rm.Float("FLOAT2"),
			expected: float64(-89.08),
		},
		{
			nm:       `Timestamp("TIMESTAMP0")`,
			value:    isNilDt(rm.Timestamp("TIMESTAMP0")),
			expected: nil,
		},
		{
			nm:       `DateTime("TIMESTAMP0")`,
			value:    isNilDt(rm.DateTime("TIMESTAMP0")),
			expected: testDateTm,
		},
		{
			nm:       `Timestamp("TIMESTAMP1")`,
			value:    isNilDt(rm.Timestamp("TIMESTAMP1")),
			expected: testDateTm,
		},
		{
			nm:       `DateTime("TIMESTAMP1")`,
			value:    isNilDt(rm.DateTime("TIMESTAMP1")),
			expected: nil,
		},
		{
			nm:       `Bool("BOOL0")`,
			value:    rm.Bool("BOOL0"),
			expected: true,
		},
		{
			nm:       `Bool("BOOL0")`,
			value:    rm.Bool("BOOL0"),
			expected: true,
		},
		{
			nm:       `Bool("BOOL0")`,
			value:    rm.Bool("BOOL0"),
			expected: true,
		},
		{
			nm:       `Bool("BOOL1", "1", "X", "true")`,
			value:    rm.Bool("BOOL1", "1", "X", "true"),
			expected: true,
		},
		{
			nm:       `Bool("BOOL2", "1", "X")`,
			value:    rm.Bool("BOOL2", "1", "X"),
			expected: true,
		},
		{
			nm:       `Bool("BOOL3", "1", "X", "true")`,
			value:    rm.Bool("BOOL3", "1", "X", "true"),
			expected: false,
		},
	}
	for _, tt := range tests {
		if !reflect.DeepEqual(tt.value, tt.expected) {
			t.Errorf("expected %s = %v; got %v", tt.nm, tt.value, tt.expected)
		}
	}

	vals, err := rm.ReadArray("CONTACTS/CONTACT")
	if err != nil {
		t.Errorf("ReadArray received error %v", err)
	}
	if len(vals) != 3 || vals[0].String("@id") != "123" || vals[1].String("CITY") != "Indianapolis" {
		t.Errorf("ReadArray expected [map[@id:123 NAME:Contact1 CITY:Carmel] map[@id:124 NAME:Contact2 CITY:Indianapolis]]; got %v", vals)
	}
}

func isNilDt(dt *time.Time) interface{} {
	if dt == nil {
		return nil
	}
	return *dt
}
