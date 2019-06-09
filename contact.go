// Copyright 2019 James Cote
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package intacct

// Contact describes the Contact entity
type Contact struct {
	RecordNumber            Int           `xml:"RECORDNO,omitempty"`
	ContactName             string        `xml:"CONTACTNAME,omitempty"`
	Prefix                  string        `xml:"PREFIX,omitempty"`
	FirstName               string        `xml:"FIRSTNAME,omitempty"`
	LastName                string        `xml:"LASTNAME,omitempty"`
	MI                      string        `xml:"INITIAL,omitempty"`
	CompanyName             string        `xml:"COMPANYNAME,omitempty"`
	PrintAs                 string        `xml:"PRINTAS,omitempty"`
	Taxable                 Bool          `xml:"TAXABLE,omitempty"`
	TaxGroup                string        `xml:"TAXGROUP,omitempty"`
	PhoneNumber             string        `xml:"PHONE1,omitempty"`
	Secondaryphone          string        `xml:"PHONE2,omitempty"`
	CellularPhoneNumber     string        `xml:"CELLPHONE,omitempty"`
	PagerNumber             string        `xml:"PAGER,omitempty"`
	FaxNumber               string        `xml:"FAX,omitempty"`
	EmailAddress            string        `xml:"EMAIL1,omitempty"`
	SecondaryEmailAddresses string        `xml:"EMAIL2,omitempty"`
	URL                     string        `xml:"URL1,omitempty"`
	SecondaryURL            string        `xml:"URL2,omitempty"`
	Visible                 Bool          `xml:"VISIBLE,omitempty"`
	Status                  string        `xml:"STATUS,omitempty"`
	PriceSchedule           string        `xml:"PRICESCHEDULE,omitempty"`
	Discount                string        `xml:"DISCOUNT,omitempty"`
	PriceList               string        `xml:"PRICELIST,omitempty"`
	PriceListKey            Int           `xml:"PRICELISTKEY,omitempty"`
	TaxID                   string        `xml:"TAXID,omitempty"`
	TaxGroupKey             string        `xml:"TAXGROUPKEY,omitempty"`
	PriceScheduleKey        string        `xml:"PRICESCHEDULEKEY,omitempty"`
	WhenCreated             Datetime      `xml:"WHENCREATED,omitempty"`
	WhenModified            Datetime      `xml:"WHENMODIFIED,omitempty"`
	CreatedBy               string        `xml:"CREATEDBY,omitempty"`
	ModifiedBy              string        `xml:"MODIFIEDBY,omitempty"`
	CreatedatEntityKey      Int           `xml:"MEGAENTITYKEY,omitempty"`  // Read Only
	CreatedatEntityID       string        `xml:"MEGAENTITYID,omitempty"`   // Read Only
	CreatedatEntityName     string        `xml:"MEGAENTITYNAME,omitempty"` // Read Only
	RecordURL               string        `xml:"RECORD_URL,omitempty"`     // Read Only
	Address                 *MailAddress  `xml:"MAILADDRESS,omitempty"`
	CustomFields            []CustomField `xml:",any"`
}

// MailAddress describes the mail address for a contact
type MailAddress struct {
	Addr1         string        `xml:"ADDRESS1,omitempty"`
	Addr2         string        `xml:"ADDRESS2,omitempty"`
	City          string        `xml:"CITY,omitempty"`
	StateProvince string        `xml:"STATE,omitempty"`
	ZipPostalCode string        `xml:"ZIP,omitempty"`
	Country       string        `xml:"COUNTRY,omitempty"`
	CountryCode   string        `xml:"COUNTRYCODE,omitempty"`
	Latitude      Float64       `xml:"LATITUDE,omitempty"`
	Longitude     Float64       `xml:"LONGITUDE,omitempty"`
	CustomFields  []CustomField `xml:",any"`
}
