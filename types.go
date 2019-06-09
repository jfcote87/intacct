// Copyright 2019 James Cote
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package intacct

import (
	"encoding/xml"
	"strconv"
	"strings"
	"time"
)

// Date used to handle intact read and readQuery date format
type Date struct {
	t *time.Time
}

// IsNil returns whether the underlying time is nil
func (dx Date) IsNil() bool {
	return dx.t == nil || dx.t.IsZero()
}

// TimeToDate converts a time.Time pointer to an intacct.Date pointer
func TimeToDate(t time.Time) Date {
	if !t.IsZero() {
		return Date{t: &t}
	}
	return Date{}

}

// Val returns intacct date at *Time.time.  Blanks returned as nil
func (dx Date) Val() *time.Time {
	if dx.IsNil() {
		return nil
	}
	return dx.t
}

// String returns the date in YYYY-MM-DD format
func (dx Date) String() string {
	if dx.IsNil() {
		return ""
	}
	return dx.t.Format("2006-01-02")
}

// MarshalXML to YYYY-MM-DD
func (dx Date) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if dx.IsNil() {
		return nil
	}
	return e.EncodeElement(dx.t.Format("2006-01-02"), start)
}

// UnmarshalXML from YYYY-MM-DD
func (dx *Date) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var s string
	err := d.DecodeElement(&s, &start)
	if err == nil {
		if s == "" { // if blank make nil
			dx.t = nil
			return nil
		}
		var t time.Time
		if strings.Count(s, "/") > 1 {
			t, err = time.Parse("01/02/2006", s)
		} else {
			t, err = time.Parse("2006-01-02", s)
		}
		dx.t = &t
	}
	return err
}

// Datetime used to handle intact read and readQuery date format
type Datetime Date

// TimeToDatetime converts a time.Time pointer to an intacct.Datetime pointer
func TimeToDatetime(t time.Time) Datetime {
	dx := TimeToDate(t)
	return Datetime(dx)
}

// IsNil returns whether the underlying time is nil
func (dt Datetime) IsNil() bool {
	return dt.t == nil || dt.t.IsZero()
}

// Val returns intacct datetime.
func (dt Datetime) Val() *time.Time {
	return Date(dt).Val()
}

// String returns an RC3339 output of the date
func (dt Datetime) String() string {
	if dt.IsNil() {
		return ""
	}
	return dt.t.Format(time.RFC3339)
}

// MarshalXML to RFC3339 format
func (dt Datetime) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if dt.IsNil() {
		return nil
	}
	return e.EncodeElement(dt.t.Format(time.RFC3339), start)
}

// UnmarshalXML from YYYY-MM-DD
func (dt *Datetime) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var s string
	err := d.DecodeElement(&s, &start)
	if err == nil {
		if s == "" {
			dt.t = nil
			return nil
		}
		var t time.Time
		if strings.Count(s, "/") > 1 {
			t, err = time.Parse("01/02/2006 15:04:05", s)
		} else {
			t, err = time.Parse(time.RFC3339, s)
		}
		dt.t = &t
	}
	return err
}

// Float64 handles intacct xml float values
type Float64 float64

// Int handles intacct xml int values
type Int int64

// Bool handles intacct xml bool values
type Bool bool

// Val returns 0 for blank
func (f Float64) Val() float64 {
	return float64(f)
}

// Val returns 0 for blank
func (f Float64) String() string {
	return strconv.FormatFloat(float64(f), 'f', -1, 64)
}

// Val returns 0 for blank
func (i Int) Val() int64 {
	return int64(i)
}

func (i Int) String() string {
	return strconv.FormatInt(int64(i), 10)
}

// Val checks for default true values, false for all others
func (b Bool) Val() bool {
	return bool(b)
}

// UnmarshalXML decodes float values and sets value to 0 on any parse errors
func (f *Float64) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var s string
	_ = d.DecodeElement(&s, &start)
	if val, err := strconv.ParseFloat(s, 64); err == nil {
		*f = Float64(val)
	}
	return nil
}

// UnmarshalXML decodes int values and sets value to 0 on any parse errors
func (i *Int) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var s string
	_ = d.DecodeElement(&s, &start)
	if val, err := strconv.ParseInt(s, 10, 64); err == nil {
		*i = Int(val)
	}
	return nil
}

// UnmarshalXML decodes bool values and sets value to false on any parse errors
func (b *Bool) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var s string
	_ = d.DecodeElement(&s, &start)
	if val, err := strconv.ParseBool(s); err == nil {
		*b = Bool(val)
	}
	return nil
}

// CustomField provides a key/pair structure for marshalling and
// unmarshalling custom fields for an Intacct object
type CustomField struct {
	Name  string
	Value string
}

// MarshalXML serializes a custom field into <NAME>VALUE</NAME>
func (c CustomField) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return e.EncodeElement(c.Value, xml.StartElement{Name: xml.Name{Local: c.Name}, Attr: start.Attr})
}

// UnmarshalXML decodes unreference xml tags into a CustomField Slice
func (c *CustomField) UnmarshalXML(d *xml.Decoder, s xml.StartElement) error {
	var val string
	if err := d.DecodeElement(&val, &s); err != nil {
		return err
	}
	*c = CustomField{s.Name.Local, val}
	return nil
}
