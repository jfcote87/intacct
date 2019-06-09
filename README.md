# Intacct Web Services Rest Api version 3 in Go

The intacct package provides a service to execute and evaluate the generic
[Sage Intacct Web Service](https://developer.intacct.com/web-services/) functions
to read/update/describe Sage Intacct objects/tables. The following example demonstrates
how an operation is performed.

```go
ctx := context.Background()
// create Service
sv := &intacct.Service{
    SenderID: "YOUR_SENDER_ID",
    Password: "****",
    Authenticator: intacct.SessionID("YOUR_SESSION_ID"),
}
// create function
f := intacct.ReadByQuery("CONTACT", "NAME LIKE '%MITCHELL%'")
// exec function(s) into Response
resp, err := sv.Exec(ctx, f)
if err != nil {
    log.Fatalf("Exec err: %v", err)
}
// define result as either pointer to struct or slice
var result []intacct.Contact
if err = resp.Decode(&result); err != nil {
    log.Fatalf("Decode error: %v", err)
}
for _, contact := range result {
    log.Print
}
```

The generic functions are:

- Read - list using key value or list of key values
- ReadByName - read by a name key or list of names
- ReadByQuery - read using Sage Intacct query
- ReadMore - read additional results of a query
- ReadRelated - read records related to custom object
- Create
- Update
- Delete
- Inspect - list object details
- LegacyFunction - use v2.1 dtd functions

Examples may be found in the examples_test.go file.

To define a transaction, add control id, add policy id for asynchronous call, etc, create an
intacct.ControlConfig and use Service.ExecWithControl.

## Authentication

The Authenticator interface should return an intacct.SessionID or any object that will xml marshal into
an Intacct [login or sessionid element](https://developer.intacct.com/web-services/requests/#authentication-element)

The package provide three Authenticators (Login, SessionID and Session).  Each are safe to use concurrently, although a
developer may write her own.

Loading a Service from a json file using the ServerFromConfig funcs is the simplest way to create/implement authentication.
Below are json examples of service configurations.

- login only

```json
{
    "sender_id": "Your SenderID",
    "password": "Your Password",
    "login": {
        "user_id": "xml_gateway",
        "company": "Company Name",
        "password": "User Password",
        "location_id": "XYZ"
    }
}
```

- Session ID

```json
{
    "sender_id": "Your SenderID",
    "password": "Your Password",
    "session": {
        "sessionId": "*******",
    }
}
```

- Session ID w/ refresh

```go
{
    "sender_id": "Your SenderID",
    "password": "Your Password",
    "login": {
        "user_id": "xml_gateway",
        "company": "Company Name",
        "password": "User Password",
        "location_id": "XYZ"
    },
    "session": {}
}
```

## Data Types

The following types have been created to properly unmarshal the xml responses from intacct.

- intacct.Date (Pt_FieldDate) YYYY-MM-DD and YYYY/MM/DD formats
- intacct.Datetime (Pt_FieldDateTime) RFC3339 and 01/02/2006 15:04:05 formats
- intacct.Int (Pt_FieldInt)
- intacct.Float (Pt_FieldDouble)
- intacct.Bool (Pt_FieldBoolean)

Each type contains a Val() to return native values or *time.Time

## Objects

Object definitions are not provided by this package due to the lack of a canonical list from Sage Intacct.  Custom fields differ between installations, and
the inspect function does not indicate which fields are standard.  Also, difference occur between the Read/ReadByName and ReadByQuery functions when handling related records.  

When defining an object struct, DO NOT include an xml.Name field.

The genobject package contains a utility to create a struct definition for objects.

To list objects:

```sh
$ cd $GOPATH/src/github.org/jfcote87/intacct/genobject
$ go run main.go -cfg config.json
Objects:
APADJUSTMENT: AP Adjustment
APADJUSTMENTITEM: AP Adjustment Detail
APBILL: AP Bill
....
```

To generate a definition for an object or list objects

```sh
$ cd $GOPATH/src/github.org/jfcote87/intacct/genobject
$ go run main.go -cfg config.json VENDOR APBILL GLDETAIL
// VENDOR (VENDOR)
type VENDOR struct {
RecordNumber intacct.Int `xml:"RECORDNO,omitempty"`
VendorID string `xml:"VENDORID,omitempty"`// Required
...
```

An ResultMap type may be used as a result for decoding a function.  The function response xml is unmarshalled into a
map[string]interface{}.  An example is below

```go
f := Read("CUSTOMER")
resp, _ := sv.Exec(ctx, f)
var resmap = make(intacct.ResultMap)
_ := resp.Decode(&resmap)
for k, v := range resmap {
    log.Printf("%s: %v", k, v)
}
```

Paste the snippet into code and gofmt.

## Legacy Functions

The v21 package contains the version 2.1DTD definitions.  Try not to use it. 
Documentation seems mostly unavailable or sparse.  Examples are provided 
for accessing and creating documents (supdoc).

## TODO

1. Add [readViews](https://developer.intacct.com/api/platform-services/views/#list-view-records)
2. Add [custom reports](https://developer.intacct.com/api/customization-services/custom-reports/) functions 
3. Add asynchronous examples