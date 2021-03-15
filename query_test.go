package intacct_test

import (
	"bytes"
	"encoding/xml"
	"reflect"
	"testing"
	"time"

	"github.com/jfcote87/intacct"
)

func TestFilter(t *testing.T) {
	tm1 := time.Date(2020, 11, 01, 0, 0, 0, 0, time.UTC)
	tm2 := tm1.Add(time.Hour * 24 * 3)
	var f = intacct.NewFilter()
	fa := f.And()
	fa.EqualTo("FLD1", "Val1")
	fa.EqualTo("FLD1a", "")
	fa.NotEqualTo("FLD2", "Val2")
	fa.LessThan("FLD3", "Val3").LessThanOrEqualTo("FLD4", "Val4")
	fa.GreaterThan("FLD5", "Val5").GreaterThanOrEqualTo("FLD6", "Val6")
	fa.Between("FLD7", tm1, tm2).In("FLD8", "Val8a", "Val8b").IsNull("FLD9")
	fa.Like("FLD9a", "Val9a")
	fo := f.Or()
	fo.IsNotNull("FLD10").NotIn("FLD11", "Val11a", "Val11b").NotLike("FLD12", "Val12")

	if len(f.Filters) != 2 {
		t.Errorf("expected two top level filters; got %d", len(f.Filters))
		return
	}
	if f.Filters[0].XMLName.Local != "and" {
		t.Errorf("expected and filter; got %s", f.Filters[0].XMLName.Local)
		return

	}
	if f.Filters[1].XMLName.Local != "or" {
		t.Errorf("expected or filter; got %s", f.Filters[1].XMLName.Local)
		return
	}

	tests_and := []intacct.Filter{
		{XMLName: xml.Name{Local: "equalto"}, Field: "FLD1", Value: []string{"Val1"}},
		{XMLName: xml.Name{Local: "equalto"}, Field: "FLD1a", Value: []string{""}},
		{XMLName: xml.Name{Local: "notequalto"}, Field: "FLD2", Value: []string{"Val2"}},
		{XMLName: xml.Name{Local: "lessthan"}, Field: "FLD3", Value: []string{"Val3"}},
		{XMLName: xml.Name{Local: "lessthanorequalto"}, Field: "FLD4", Value: []string{"Val4"}},
		{XMLName: xml.Name{Local: "greaterthan"}, Field: "FLD5", Value: []string{"Val5"}},
		{XMLName: xml.Name{Local: "greaterthanorequalto"}, Field: "FLD6", Value: []string{"Val6"}},
		{XMLName: xml.Name{Local: "between"}, Field: "FLD7", Value: []string{"11/01/2020", "11/04/2020"}},
		{XMLName: xml.Name{Local: "in"}, Field: "FLD8", Value: []string{"Val8a", "Val8b"}},
		{XMLName: xml.Name{Local: "isnull"}, Field: "FLD9"},
		{XMLName: xml.Name{Local: "like"}, Field: "FLD9a", Value: []string{"Val9a"}},
	}

	tests_or := []intacct.Filter{
		{XMLName: xml.Name{Local: "isnotnull"}, Field: "FLD10"},
		{XMLName: xml.Name{Local: "notin"}, Field: "FLD11", Value: []string{"Val11a", "Val11b"}},
		{XMLName: xml.Name{Local: "notlike"}, Field: "FLD12", Value: []string{"Val12"}},
	}

	for i, fa := range f.Filters[0].Filters {
		if !reflect.DeepEqual(fa, tests_and[i]) {
			t.Errorf("expected %v, got %v", tests_and[i], fa)
		}
	}
	for i, fo := range f.Filters[1].Filters {
		if !reflect.DeepEqual(fo, tests_or[i]) {
			t.Errorf("expected %v, got %v", tests_or[i], fo)
		}
	}

}

func TestOrderBy_MarshalXML(t *testing.T) {
	tests := []struct {
		name    string
		orderby *intacct.QuerySort //intacct.OrderBy
		want    string
	}{
		{name: "t1", orderby: &intacct.QuerySort{Fields: []intacct.OrderBy{{Field: "F1"}}}, want: "<orderby><order><field>F1</field></order></orderby>"},
		{name: "t2", orderby: &intacct.QuerySort{Fields: []intacct.OrderBy{{Field: "F2", Descending: true}}}, want: "<orderby><order><field>F2</field><descending></descending></order></orderby>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buff := &bytes.Buffer{}
			e := xml.NewEncoder(buff)
			e.Encode(tt.orderby)
			e.Flush()
			if string(buff.Bytes()) != tt.want {
				t.Errorf("%s expected %s; got %s", tt.name, tt.want, buff.Bytes())
			}
		})
	}
}

func TestMarshal(t *testing.T) {
	var f *intacct.Filter
	sx := intacct.Query{
		Object: "PROJECT",
		Select: intacct.Select{
			Fields: []string{"RECORDNO", "PROJECTID", "NAME", "DESCRIPTION", "PARENTNAME"},
			Min:    "PROJECTID",
		},
		Sort:   &intacct.QuerySort{Fields: []intacct.OrderBy{{Field: "PROJECTID"}, {Field: "NAME", Descending: true}}},
		Filter: f.EqualTo("RECORDNO", "1").In("PROJECTID", "P1", "P2").EqualTo("NAME", ""),
	}
	b, err := xml.Marshal(sx)
	if err != nil {
		t.Fatalf("%v", err)
	}
	expect := `<query><object>PROJECT</object><select><field>RECORDNO</field><field>PROJECTID</field><field>NAME</field><field>DESCRIPTION</field><field>PARENTNAME</field><min>PROJECTID</min></select><filter><equalto><field>RECORDNO</field><value>1</value></equalto><in><field>PROJECTID</field><value>P1</value><value>P2</value></in><equalto><field>NAME</field><value></value></equalto></filter><orderby><order><field>PROJECTID</field></order><order><field>NAME</field><descending></descending></order></orderby></query>`
	if expect != string(b) {
		t.Errorf("expected marshal of %s; got %s", expect, b)
	}
}
