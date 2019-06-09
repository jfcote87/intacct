package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"bitbucket.org/gotamer/cases"
	"github.com/jfcote87/intacct"
)

var configFile = flag.String("cfg", "", "file name of a json or xml file containing the service definition")
var queryFlag = flag.Bool("inline-fields", false, "flatten objects for ReadByQuery format")

const usageMsg = "usage: -cfg [SERVICE_DEF_FILE] [-inline-fields] [OBJECTNAME....]"

var dataTypeMap = map[string]string{
	"Pt_FieldDateTime":     "intacct.Datetime",
	"Pt_FieldDummy":        "string",
	"Pt_FieldRelationship": "string",
	"Pt_FieldInt":          "intacct.Int",
	"Pt_FieldString":       "string",
	"Pt_FieldText":         "string",
	"Pt_FieldBoolean":      "intacct.Bool",
	"Pt_FieldDate":         "intacct.Date",
	"Pt_FieldDouble":       "intacct.Float64",
}

var nr = strings.NewReplacer("_", "", "(", "", ")", "", "%", "", "-", "", "/", "", ".", "", "'", "", ",", "")

func main() {
	var sv *intacct.Service
	var err error
	var msg string

	flag.Parse()

	if *configFile == "" {
		msg = usageMsg
	} else if sv, err = getService(*configFile); err != nil {
		msg = fmt.Sprintf("error parsing %s: %v", *configFile, err)

	}
	if msg > "" {
		fmt.Fprintf(os.Stdout, "%s\n", msg)
		os.Exit(1)
	}
	if flag.NArg() == 0 {
		err = listObjects(sv)
	} else {
		err = writeStructs(sv, flag.Args()...)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}

}

func listObjects(sv *intacct.Service) error {
	ctx := context.Background()
	f := &intacct.Inspector{
		Object: "*",
	}
	resp, err := sv.Exec(ctx, f)
	if err != nil {
		return fmt.Errorf("exec error: %v", err)
	}
	var results []intacct.InspectName
	err = resp.Decode(&results)
	if err != nil {
		return fmt.Errorf("decode error: %v", err)
	}
	fmt.Printf("Objects:")
	for _, result := range results {
		fmt.Printf("%s: %s\n", result.TypeName, result.Name)
	}
	return nil
}

func writeStructs(sv *intacct.Service, objNames ...string) error {
	var funcs []intacct.Function
	var results []interface{}

	for _, objName := range objNames {
		funcs = append(funcs, &intacct.Inspector{
			IsDetail: 1,
			Object:   objName,
		})
		results = append(results, &intacct.InspectDetailResult{})
	}
	resp, err := sv.Exec(context.Background(), funcs...)
	if err != nil {
		return fmt.Errorf("exec error: %v", err)
	}

	if err = resp.Decode(results...); err != nil {
		return fmt.Errorf("decode error: %v", err)
	}
	for _, ix := range results {
		writeStruct(ix.(*intacct.InspectDetailResult))
	}
	return nil
}

func writeStruct(result *intacct.InspectDetailResult) {
	fldList := &fieldOutput{List: make([]structureFields, 0, len(result.Fields)), MultiField: make(map[string]*fieldOutput), PrevNames: make(map[string]string), HandleMulti: !*queryFlag}
	for idx, f := range result.Fields {
		if err := fldList.process(f.Name, f, idx); err != nil {
			log.Printf("Unable to add field# %d (%s): %v", idx, f.Name, err)
			return
		}
	}
	fldList.handleStructs()
	var baseName = result.Name
	w := os.Stdout
	fmt.Fprintf(w, "// %s (%s)\n", baseName, result.Name)
	fmt.Fprintf(w, "type %s struct {\n", baseName)

	for _, f := range fldList.List {
		if f.TopComment > "" {
			fmt.Fprintf(w, f.TopComment)
		}
		fmt.Fprintf(w, "%s %s `xml:\"%s,omitempty\"`%s\n", f.FldName, f.DataType, f.XMLNm, f.Comment)
	}
	fmt.Fprint(w, "CustomFields []intacct.CustomField `xml:\",any\"`\n")
	fmt.Fprintf(w, "}\n\n")
	fldList.writeStructs(os.Stdout)
}

func getService(fn string) (*intacct.Service, error) {
	b, err := ioutil.ReadFile(fn)
	if err != nil {
		return nil, err
	}
	return intacct.ServiceFromConfigJSON(bytes.NewReader(b))

}

type structureFields struct {
	OriIdx     int
	Idx        int
	ParentName string
	FldName    string
	DataType   string
	XMLNm      string
	IsReadOnly bool
	IsRequired bool
	Comment    string
	TopComment string
}

type fieldOutput struct {
	List        []structureFields
	MultiField  map[string]*fieldOutput
	PrevNames   map[string]string
	HandleMulti bool
}

func (fo *fieldOutput) getFieldLabels(xnm string, f intacct.FieldDetail) (string, string, string, string, error) {
	ty, ok := dataTypeMap[f.DataName]
	if !ok {
		ty = "interface{}"
	}
	var fdlbl = f.DisplayLabel
	if fdlbl == "" {
		fdlbl = strings.ToLower(f.Name)
	}
	if len(fdlbl) == 0 {
		return "", "", "", "", fmt.Errorf("Empty Name")
	}
	if int(fdlbl[0]) >= 48 && int(fdlbl[0]) <= 57 {
		fdlbl = "F" + fdlbl
	}
	fldSlice := strings.Split(fdlbl, ".")
	var splitchar = "."
	if len(fldSlice) == 0 {
		fldSlice = strings.Split(fdlbl, "-")
		splitchar = "-"
	}
	if len(fldSlice) > 0 {
		fdlbl = strings.Replace(fdlbl, splitchar, "_", -1)
	}

	snm := cases.Camel(nr.Replace(fdlbl))

	if _, ok = fo.PrevNames[snm]; ok {

		snm = makeName(xnm)

	} else {
		fo.PrevNames[snm] = ""
	}
	//}
	comment := ""
	if f.IsReadOnly {
		comment = "// Read Only"
	}
	if f.IsRequired {
		if comment > "" {
			comment = comment + " Required"
		} else {
			comment = "// Required"
		}

	}
	topComment := ""
	if f.RelatedObject > "" {
		topComment = fmt.Sprintf("// %s: %s\n", f.RelatedObject, f.Relationship)
	}
	return snm, ty, comment, topComment, nil
}

func makeName(fldNm string) string {
	if int(fldNm[0]) >= 48 && int(fldNm[0]) <= 57 {
		fldNm = "F" + fldNm
	}
	return cases.Camel(nr.Replace(fldNm))

}

func (fo *fieldOutput) process(nm string, fd intacct.FieldDetail, idx int) error {
	sx := []string{nm}
	if fo.HandleMulti {
		sx = strings.Split(nm, ".")
	}
	sf := &structureFields{}
	sf.Idx = len(fo.List)
	sf.OriIdx = idx
	if fd.RelatedObject > "" {
		sf.Comment = fmt.Sprintf("// %s: %s\n", fd.RelatedObject, fd.Relationship)
	}
	if len(sx) == 1 {
		fldNm, dataType, comment, topComment, err := fo.getFieldLabels(nm, fd)
		if err != nil {
			return err
		}

		sf.DataType = dataType
		sf.FldName = fldNm
		sf.Comment = comment
		sf.TopComment = topComment
		if fd.DataName == "Pt_FieldRelationship" {
			nm = strings.ToUpper(nm)
		}
		sf.XMLNm = nm
		fo.List = append(fo.List, *sf)

	} else {
		nextFo, ok := fo.MultiField[sx[0]]
		if !ok {
			//sf := &structureFields{}
			sf.DataType = "struct{}"
			sf.FldName = makeName(sx[0])
			sf.XMLNm = sx[0]
			fo.List = append(fo.List, *sf)
			nextFo = &fieldOutput{List: make([]structureFields, 0), MultiField: make(map[string]*fieldOutput), PrevNames: make(map[string]string)}
			fo.MultiField[sx[0]] = nextFo
		}
		return nextFo.process(strings.Join(sx[1:], "."), fd, idx)
	}
	return nil
}

func (fo *fieldOutput) writeStructs(w io.Writer) {
	for k, v := range fo.MultiField {
		v.writeStructs(w)
		strName := string(k[0]) + strings.ToLower(k[1:])
		fmt.Fprintf(w, "type %s struct {\n", strName)
		for _, fld := range v.List {
			if fld.Comment > "" {
				fmt.Fprintf(w, "// %s\n", fld.TopComment)
			}
			if fld.IsReadOnly || fld.IsRequired {

			}
			fmt.Fprintf(w, "%s %s `xml:\"%s,omitempty\"`%s\n", fld.FldName, fld.DataType, fld.XMLNm, fld.Comment)
		}
		fmt.Fprintf(w, "}\n\n")
	}
}

func (fo *fieldOutput) handleStructs() {
	for idx, fx := range fo.List {
		if fx.DataType == "struct{}" {
			flx, ok := fo.MultiField[fx.XMLNm]
			if ok {
				tMap := make(map[string]bool)
				for _, flds := range flx.List {
					tMap[flds.XMLNm] = true
				}
				if tMap["PRINTAS"] && tMap["PHONE1"] && tMap["CONTACTNAME"] && tMap["MAILADDRESS.ADDRESS1"] {
					fo.List[idx].DataType = "*intacct.Contact"
					delete(fo.MultiField, fx.XMLNm)
				} else {
					fo.List[idx].DataType = "*" + string(fx.XMLNm[0]) + strings.ToLower(string(fx.XMLNm[1:]))
				}
			}

		}
	}
	for _, v := range fo.MultiField {
		v.handleStructs()
	}
}
