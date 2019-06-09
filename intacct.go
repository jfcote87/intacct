// Copyright 2019 James Cote
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package intacct // import "github.com/jfcote87/intacct"

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/jfcote87/ctxclient"
)

// DefaultEndpoint used until an Authenticator returns a different one
const DefaultEndpoint = "https://api.intacct.com/ia/xml/xmlgw.phtml"

// DefaultDTDVersion used for requests.  May be overridden by using CustomControl struct
// in Service.ExecWithControl.
const DefaultDTDVersion = "3.0"

func getEndpoint(auth Authenticator) string {
	if f, ok := auth.(Endpoint); ok {
		return f.GetEndpoint()
	}
	return DefaultEndpoint
}

// ControlIDFunc generates unique Control IDs for requests
type ControlIDFunc func(ctx context.Context) string

// ID executes a ControlIDFunc while providing
// an ID generator based upon time for null instances
func (idFunc ControlIDFunc) ID(ctx context.Context) string {
	if idFunc == nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return idFunc(ctx)
}

func (idFunc ControlIDFunc) isEmpty(ctx context.Context, id string) string {
	if id != "" {
		return id
	}
	return idFunc.ID(ctx)
}

// Service stores configuration information and provides functions
// for sending requests.  It is safe for concurrent
type Service struct {
	// see https://developer.intacct.com/web-services/#authentication
	// SenderID and Password are the customer/partner login, not company login
	SenderID string
	Password string
	// must provide interface{} to marshal to a Login or SessionID element
	Authenticator Authenticator
	// ControlIDFunc may be set to create control IDs for the request header.  If
	// nil the control ID for a call will be the current time in nanoseconds.
	ControlIDFunc
	// Set if a unique client is need.
	HTTPClientFunc ctxclient.Func
}

// Authenticator returns an interface{} that will xml marshal into
// a valid login or sessionid element for a request
type Authenticator interface {
	GetAuthElement(ctx context.Context) (interface{}, error)
}

// AuthResponseChecker is an Authenticator that checks the
// response after each exec.  May be used for caching and updating
// session settings.
type AuthResponseChecker interface {
	Authenticator
	CheckResponse(context.Context, *Response)
}

// Endpoint returns an endpoint for a request and should always return
// a valid endpoint.
type Endpoint interface {
	GetEndpoint() string
}

// Login provides a username/password authentication mechanism
// ClientID and LocationID are optional.
// https://developer.intacct.com/web-services/requests/#authentication-element
type Login struct {
	UserID     string `xml:"login>userid" json:"user_id"`
	Company    string `xml:"login>companyid" json:"company"`
	Password   string `xml:"login>password" json:"password"`
	ClientID   string `xml:"login>clientid,omitempty" json:"client_id,omitempty"`
	LocationID string `xml:"login>locationid,omitempty" json:"location_id,omitempty"`
}

// GetAuthElement fulfills the Authenticator interface{}.  Returns itself which
// will marshal into a login element for the request.
func (l *Login) GetAuthElement(ctx context.Context) (interface{}, error) {
	if l == nil {
		return nil, errNilLogin
	}
	return l, nil
}

// SessionRefresher returns a function for creating a new SessionID.
func (l *Login) SessionRefresher(sv *Service) func(context.Context) (*SessionResult, error) {
	if l == nil {
		return func(ctx context.Context) (*SessionResult, error) {
			return nil, errNilLogin
		}
	}

	// copy sv fields to use User/Pass login
	sv2 := &Service{
		SenderID:       sv.SenderID,
		Password:       sv.Password,
		Authenticator:  l,
		HTTPClientFunc: sv.HTTPClientFunc,
		ControlIDFunc:  sv.ControlIDFunc,
	}
	return func(ctx context.Context) (*SessionResult, error) {
		resp, err := sv2.ExecWithControl(ctx, nil, &Writer{Cmd: "getAPISession"})
		if err != nil {
			return nil, err
		}

		var result = &SessionResult{
			Expires: resp.Auth.getTimeout(),
		}
		return result, resp.Decode(&result)
	}
}

// SessionID provides an authorization token that may be used in a Request.
type SessionID string

// GetAuthElement fulfills the Authenticator interface{}.  Returns itself which
// will marshal into a sessionid element for the request.
func (s SessionID) GetAuthElement(ctx context.Context) (interface{}, error) {
	if s == "" {
		return nil, errors.New("empty sessionid")
	}
	return s, nil
}

// MarshalXML formats sessionid for a Request.
func (s SessionID) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	sname := xml.Name{Local: "sessionid"}
	e.EncodeToken(start)
	e.EncodeToken(xml.StartElement{Name: sname})
	e.EncodeToken(xml.CharData([]byte(s)))
	e.EncodeToken(xml.EndElement{Name: sname})
	e.EncodeToken(start.End())
	return nil
}

// AuthenticationConfig provides a format for serializing a Service definition
type AuthenticationConfig struct {
	SenderID       string   `xml:"sender_id" json:"sender_id,omitempty"`   // Intacct SenderID
	SenderPassword string   `xml:"sender_pwd" json:"sender_pwd,omitempty"` // Intacct Password
	Login          *Login   `xml:"login,omitempty" json:"login,omitempty"`
	Session        *Session `xml:"session,omitempty" json:"session,omitempty"`
}

// ServiceFromConfigJSON returns a service from json representation.
// DO NOT make changes to the returned Service.  Create new service
// if necessary.
func ServiceFromConfigJSON(r io.Reader, opts ...ConfigOption) (*Service, error) {
	var cfg AuthenticationConfig
	if err := json.NewDecoder(r).Decode(&cfg); err != nil {
		return nil, err
	}
	return ServiceFromConfig(cfg, opts...)
}

// A ConfigOption is passed to ServiceFrom... funcs.
// ConfigOptions may be created using the ConfigHTTPClientFunc
// and ConfigControlIDFunc funcs.
type ConfigOption interface {
	setValue(*Service)
}

type cfgOption func(*Service)

func (co cfgOption) setValue(sv *Service) {
	co(sv)
}

// ConfigHTTPClientFunc sets the HTTPClientFunc for the Service
// created by the ServiceFrom... funcs
func ConfigHTTPClientFunc(f ctxclient.Func) ConfigOption {
	return cfgOption(func(sv *Service) {
		sv.HTTPClientFunc = f
	})
}

// ConfigControlIDFunc sets the ControlIDFunc for the Service
// created by the ServiceFrom... funcs
func ConfigControlIDFunc(f ControlIDFunc) ConfigOption {
	return cfgOption(func(sv *Service) {
		sv.ControlIDFunc = f
	})
}

// ServiceFromConfig creates a service from configuration.
//
// DO NOT make changes to the returned Service.  Create new service
// if necessary.
func ServiceFromConfig(cfg AuthenticationConfig, opts ...ConfigOption) (*Service, error) {
	sv := &Service{
		SenderID: cfg.SenderID,
		Password: cfg.SenderPassword,
	}
	for _, o := range opts {
		o.setValue(sv)
	}

	// if session specified, use session authenticator
	if cfg.Session != nil {
		cfg.Session.m.Lock()
		defer cfg.Session.m.Unlock() // shouldn't ever be a problem...

		var newSession = &Session{ // do not copy lock
			ID:          cfg.Session.ID,
			Endpoint:    cfg.Session.Endpoint,
			LocationID:  cfg.Session.LocationID,
			Expires:     cfg.Session.Expires,
			ExpiryDelta: cfg.Session.ExpiryDelta,
			RefreshFunc: cfg.Session.RefreshFunc,
		}
		// if refresh is nil and Login provided, create refresh function
		if cfg.Session.RefreshFunc == nil && cfg.Login != nil {
			newSession.RefreshFunc = cfg.Login.SessionRefresher(sv)
		}
		sv.Authenticator = newSession
		return sv, nil
	}
	if cfg.Login == nil {
		return nil, errors.New("a sessionid or login must be specified")
	}
	sv.Authenticator = cfg.Login
	return sv, nil

}

// Session caches and refreshes a sessionid. Best to create via
// ServiceFrom... funcs.
type Session struct {
	ID          SessionID
	Endpoint    string
	LocationID  string
	Expires     time.Time
	ExpiryDelta int64
	RefreshFunc func(ctx context.Context) (*SessionResult, error)
	m           sync.Mutex
}

// GetEndpoint returns the session's endpoint and
// fulfills Endpoint interface
func (s *Session) GetEndpoint() string {
	if s == nil || s.Endpoint == "" {
		return DefaultEndpoint
	}
	return s.Endpoint
}

// GetAuthElement returns a new sessionID to authenticate request
func (s *Session) GetAuthElement(ctx context.Context) (interface{}, error) {
	var err error
	// add delta to current time for comparison
	s.m.Lock()
	curTime := time.Now().Add(time.Second * time.Duration(s.ExpiryDelta))
	// check for expiration
	if len(s.ID) == 0 || curTime.Sub(s.Expires) < 0 {
		err = s.Refresh(ctx)
	}
	s.m.Unlock()

	if err != nil {
		return nil, err
	}
	return s.ID, nil
}

var xcnt = 0

// Refresh collects a new sessionid from intacct. Must protect
// this the Session with using s.m.
func (s *Session) Refresh(ctx context.Context) error {
	if s.RefreshFunc == nil {
		return errors.New("expired session, no refresh function specified")
	}
	res, err := s.RefreshFunc(ctx)
	if err != nil {
		return err
	}
	s.ID = res.SessionID
	s.Endpoint = res.Endpoint
	s.LocationID = res.LocationID
	s.Expires = res.Expires
	return nil
}

// CheckResponse fulfills the AuthResponseChecker functionality and
// sets the session timeout.
func (s *Session) CheckResponse(ctx context.Context, r *Response) {
	if r != nil {
		s.m.Lock()
		// ensure that lastest expiration is stored
		if tm := r.Auth.getTimeout(); tm.Sub(s.Expires) > 0 {
			s.Expires = tm
		}
		s.m.Unlock()
	}
}

// SessionResult is the result of a getSessionID function
type SessionResult struct {
	XMLName    xml.Name  `xml:"api"`
	SessionID  SessionID `xml:"sessionid"`
	Endpoint   string    `xml:"endpoint"`
	LocationID string    `xml:"locationid"`
	Expires    time.Time `xml:"-"`
}

// Exec executes the given functions responding with and error
// if appropriate.  Results are contained in the Response.
func (sv *Service) Exec(ctx context.Context, f ...Function) (*Response, error) {
	return sv.ExecWithControl(ctx, nil, f...)
}

func (sv *Service) validate(ctx context.Context, f ...Function) error {
	if sv == nil {
		return errors.New("nil Service")
	}
	if sv.Authenticator == nil {
		return errors.New("nil Authenticator")
	}
	if sv.SenderID == "" || sv.Password == "" {
		return errors.New("SendorID/Passowrd is empty")
	}
	if ctx == nil {
		return errors.New("nil context")
	}
	if len(f) == 0 {
		return errors.New("no functions specified")
	}
	return nil
}

// ExecWithControl adds a ControlConfig for transactional data.
func (sv *Service) ExecWithControl(ctx context.Context, cc *ControlConfig, f ...Function) (*Response, error) {
	if err := sv.validate(ctx, f...); err != nil {
		return nil, err
	}

	// create request body
	req, err := sv.makeRequest(ctx, cc, f)
	if err != nil {
		return nil, err
	}
	// handle timeouts and non 2xx responses
	res, err := sv.HTTPClientFunc.Do(ctx, req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	var body io.Reader = res.Body

	var reqResponse *Response
	if err = xml.NewDecoder(body).Decode(&reqResponse); err != nil {
		return nil, err
	}
	if checker, ok := sv.Authenticator.(AuthResponseChecker); ok {
		checker.CheckResponse(ctx, reqResponse)
	}

	return reqResponse, reqResponse.execErr()
}

// Control creates a Control struct based on ControlConfig
func (sv *Service) Control(ctx context.Context, cc *ControlConfig) Control {
	if cc == nil { // default control element
		return Control{
			SenderID:   sv.SenderID,
			Password:   sv.Password,
			DTDVersion: DefaultDTDVersion,
			ControlID:  sv.ControlIDFunc.ID(ctx),
		}
	}
	return Control{
		SenderID:          sv.SenderID,
		Password:          sv.Password,
		ControlID:         cc.ControlID, //sv.ControlIDFunc.isEmpty(ctx, cc.ControlID),
		UniqueID:          cc.IsUnique,
		DTDVersion:        isEmpty(cc.DTDVersion, DefaultDTDVersion),
		PolicyID:          cc.PolicyID,
		Debug:             cc.Debug,
		Includewhitespace: cc.IncludeWhitespace,
	}
}

func isEmpty(val, defaultVal string) string {
	if val == "" {
		return defaultVal
	}
	return val
}

// makeRequest creates an *http.Request assigning headers and body for posting to intacct
func (sv *Service) makeRequest(ctx context.Context, cc *ControlConfig, functions []Function) (*http.Request, error) {
	// Ensure Authorization
	if sv.Authenticator == nil {
		return nil, errors.New("no authentication specified")
	}
	authElement, err := sv.Authenticator.GetAuthElement(ctx) //sv.GetAuth(ctx)
	if err != nil {
		return nil, err
	}
	control := sv.Control(ctx, cc)
	reqFuncs := make([]RequestFunction, 0, len(functions))
	for _, f := range functions {
		reqFuncs = append(reqFuncs, RequestFunction{
			ControlID: isEmpty(f.GetControlID(), control.ControlID),
			Payload:   f,
		})
	}

	// add xml header to payload
	reqBuffer := bytes.NewBufferString(xml.Header)
	xmlEncoder := xml.NewEncoder(reqBuffer)
	if err := xmlEncoder.Encode(&Request{
		Control: control, //sv.Control(ctx, cc),
		Op: Operation{
			Transaction: cc != nil && cc.IsTransaction,
			Auth:        authElement,
			Content:     reqFuncs,
		},
	}); err != nil {
		return nil, fmt.Errorf("Marshal Request: %v", err)
	}

	req, _ := http.NewRequest("POST", getEndpoint(sv.Authenticator), reqBuffer)
	req.Header.Add("Content-Type", "application/xml")
	return req, nil
}

var (
	errNoLoginMethod = errors.New("no login method provided")
	//	errMaxDurationZero = errors.New("sessionId requires max duration to be greater than zero")
	errNilLogin = errors.New("userid/password not set, unable to login or refresh sessionID")
)

// ControlConfig allows developer to specify transactional
// data for a Request.
type ControlConfig struct {
	IsTransaction     bool
	IsUnique          bool
	IncludeWhitespace bool
	Debug             bool
	ControlID         string
	PolicyID          string
	DTDVersion        string // will use DefaultDTDVersion if blank

	CompanyPrefs []Preference
	ModulePrefs  []Preference
}
