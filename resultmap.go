// Copyright 2019 James Cote
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package intacct

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ResultMap represents intacct response xml as a map of interfaces.
// Repeated xml tags become slices and attributes are encoded as
//  "@attributeName".  If a tag has attributes and chardata, the
// character data may be found as rm[""].
// i.e. <VENDOR><NAME type="short">Jim</NAME></VENDOR> becomes
// "VENDOR":intacct.ResultMap{"NAME":intacct.ResultMap{"@type":"short", "":"Jim"}}}
// while <VENDOR><NAME>Jim</NAME></VENDOR> becomes
// "VENDOR":intacct.ResultMap{"NAME":"Jim"}}
type ResultMap map[string]interface{}

// ReadArray returns a slice of elements from an Element index. If the
// value is nil or a single element, ReadArray returns an empty or a
// single valued slice.  A string value will return an error.
func (rm ResultMap) ReadArray(name string) ([]ResultMap, error) {
	keys := strings.SplitN(name, "/", 2)
	switch val := rm[keys[0]].(type) {
	case nil:
		return []ResultMap{}, nil
	case []ResultMap:
		if len(keys) > 1 {
			return nil, fmt.Errorf("%s is an array not an resultMap", keys[0])
		}
		return val, nil
	case ResultMap:
		if len(keys) > 1 {
			return val.ReadArray(keys[1])
		}
		return []ResultMap{val}, nil
	}
	return nil, fmt.Errorf("Not an ResultMap: %v", rm[name])
}

// String returns the string value of the Element index
// when it is a string.  Rather than return an error, ReadSlice
// returns an empty string for types other than a string.
func (rm ResultMap) String(name string) string {
	if s, ok := rm[name].(string); ok {
		return s
	}
	return ""
}

// Date returns the date (not datetime) from an
// result map index.  It first checks for the
// Year, Month and Day elements, then checks
// for a string format of YYYY-MM-DD and finally
// for MM-DD-YYYY
func (rm ResultMap) Date(name string) *time.Time {
	var tm time.Time

	switch m := rm[name].(type) {
	case ResultMap:
		yr, mth, day := -1, -1, -1
		for k, v := range m {
			sval, _ := v.(string)
			intVal, _ := strconv.Atoi(sval)
			switch k {
			case "Year", "year":
				yr = int(intVal)
			case "Month", "month":
				mth = int(intVal)
			case "Day", "day":
				day = int(intVal)
			}
		}
		if yr > 0 && mth > 0 && day > 0 {
			tm = time.Date(yr, time.Month(mth), day, 0, 0, 0, 0, time.UTC)
		}
	case string:
		var err error
		if tm, err = time.Parse("2006-01-02", m); err != nil {
			tm, _ = time.Parse("01-02-2006", m)
		}
	}
	if tm.IsZero() {
		return nil
	}
	return &tm
}

// Int parses the named field for an int64
func (rm ResultMap) Int(name string) int64 {
	if s, ok := rm[name].(string); ok {

		i, _ := strconv.ParseFloat(s, 64)
		return int64(i)
	}
	return 0
}

// Float parses the named field for a float64
func (rm ResultMap) Float(name string) float64 {
	if s, ok := rm[name].(string); ok {
		i, _ := strconv.ParseFloat(s, 64)
		return i
	}
	return 0
}

// Timestamp parses an RFC3339 formatted date string,
// Errors are simply returned as nil.
func (rm ResultMap) Timestamp(name string) *time.Time {
	if s, ok := rm[name].(string); ok {
		if tx, err := time.Parse(time.RFC3339, s); err == nil {
			return &tx
		}
	}
	return nil
}

// DateTime parses a get_list datetime string.
func (rm ResultMap) DateTime(name string) *time.Time {
	if s, ok := rm[name].(string); ok {
		if tx, err := time.Parse("01/02/2006 15:04:05", s); err == nil {
			return &tx
		}
	}
	return nil
}

// Bool parses true/false fields.  If not trueVals are
// indicated, Bool checks for a  values of "true"
func (rm ResultMap) Bool(name string, trueVals ...string) bool {
	s, _ := rm[name].(string)
	if len(trueVals) == 0 {
		return (s == "true")
	}
	for _, v := range trueVals {
		if s == v {
			return true
		}
	}
	return false
}

// StringArray returns a string slice
func (rm ResultMap) StringArray(name string) []string {
	s, _ := rm[name].([]string)
	return s
}

// UnmarshalXML turns serialized XML into a map[string]interface{}
func (rm ResultMap) UnmarshalXML(d *xml.Decoder, s xml.StartElement) error {
	// turn attributes in fields starting with @
	for _, a := range s.Attr {
		rm["@"+a.Name.Local] = a.Value
	}
	tk, err := d.Token()
	for err == nil {
		switch t := tk.(type) {
		case xml.StartElement:
			err = rm.newElement(d, t)
		case xml.CharData:
			if strings.Trim(string(t), " \n\t") > "" {
				rm[""] = string(t)
			}
		case xml.EndElement:
			return nil
		}
		if err == nil {
			tk, err = d.Token()
		}
	}
	return err
}

func (rm ResultMap) newElement(d *xml.Decoder, s xml.StartElement) error {
	newEl := make(ResultMap)
	if err := newEl.UnmarshalXML(d, s); err != nil {
		return err
	}
	tag := s.Name.Local
	var sVal string
	var isString bool
	if len(newEl) == 1 {
		sVal, isString = newEl[""].(string)
	}
	switch tVal := rm[tag].(type) {
	case []ResultMap:
		rm[tag] = append(tVal, newEl)
	case ResultMap:
		rm[tag] = []ResultMap{tVal, newEl}
	case string:
		if isString {
			rm[tag] = []string{tVal, sVal}
		}
	case []string:
		if isString {
			rm[tag] = append(tVal, sVal)
		}
	case nil:
		if isString {
			rm[tag] = sVal
		} else if len(newEl) > 0 {
			rm[tag] = newEl
		}
	}
	return nil
}
