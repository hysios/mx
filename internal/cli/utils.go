package cli

import (
	"errors"
	"regexp"
	"strings"
)

type Method struct {
	Name       string
	HttpMethod string
	Path       string
	Input      ArgType
	Output     ArgType
}

type ArgType struct {
	Name   string
	Fields []*Field
}

type Field struct {
	Name string
	Type string
}

// CheckMethod checks if the method string is valid format.
// The method string format is:
//
//	<name>:<method>:<path>:<input{<field1>,<field2>,...<fieldn>}>:<output{<field1>,<field2>,...<fieldn>}>
//
//  name   - method name, is pure string
//  method - method type, Is one of GET, POST, PUT, DELETE, PATCH
//  path   - path is http path, is pure string
//  input  - input argument's name, and is a field of array
//  output - output argument's name, is a field of array
//  field - field is a struct, contains name and type, format is <name>:<type>
//	type - type is a proto scalar type, is one of string, int32, int64, uint32, uint64, bool, float32, float64, bytes

// Example:
//
//	 // Service Method: GetUser
//		// HTTP method: Get
//		// input: {id:int32 name:string} use space to separate fields
//		// output: {id:int32 name:string}
//		GetUser:GET::UserRequest{id:int32 name:string}:UserResponse{id:int32 name:string}
//
//	 // Service Method: CreateUser
//		// HTTP method: Post
//		// input: {name:string}
//		// output: {id:int32 name:string}
//		CreateUser:POST:/api/users:UserRequest{name:string}:UserResponse{id:int32 name:string}
func CheckMethod(s string) error {
	// implements the doc format check
	var preSections = strings.SplitN(s, ":", 4)
	// result
	// preSections[0] = "GetUser"
	// preSections[1] = "GET"
	// preSections[2] = "UserRequest{id:int32 name:string}:UserResponse{id:int32 name:string}"

	if len(preSections) != 4 {
		return errors.New("invalid method format")
	}

	// check method name
	if preSections[0] == "" {
		return errors.New("invalid method name")
	}

	if !validIdent(preSections[0]) {
		return errors.New("method name must be ident format")
	}

	if preSections[1] != "" {
		// check method type
		if !strings.Contains("GET,POST,PUT,DELETE,PATCH", preSections[1]) {
			return errors.New("invalid method type")
		}
	}

	// check input and output
	inouts := InputRegexp.FindAllStringSubmatch(preSections[3], 2)
	if len(inouts) != 2 {
		return errors.New("invalid input or output")
	}

	// check input
	if err := validArg(inouts[0]); err != nil {
		return err
	}

	// check output
	if err := validArg(inouts[1]); err != nil {
		return err
	}

	return nil
}

// CheckInput checks if the input string is ident format.
// only conatins [a-zA-Z0-9_] and start with [a-zA-Z_]
func validIdent(s string) bool {
	return regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`).MatchString(s)
}

var (
	FieldRegexp = regexp.MustCompile(`(?P<name>\w+):(?P<type>\w+)`)
	InputRegexp = regexp.MustCompile(`(?P<name>\w+)\{(?P<fields>.*?)\}`)
)

func validArg(ss []string) error {

	// check input
	if ss[1] == "" {
		return errors.New("invalid input name")
	}

	if !validIdent(ss[1]) {
		return errors.New("input name must be ident format")
	}

	// inputs has many fields
	fields := FieldRegexp.FindAllStringSubmatch(ss[2], -1)
	if len(fields) == 0 {
		return errors.New("invalid input fields")
	}

	for _, field := range fields {
		if field[1] == "" {
			return errors.New("invalid input field name")
		}

		if !validIdent(field[1]) {
			return errors.New("input field name must be ident format")
		}

		if field[2] == "" {
			return errors.New("invalid input field type")
		}

		if !strings.Contains("string,int32,int64,uint32,uint64,bool,float32,float64,bytes", field[2]) {
			return errors.New("invalid input field type")
		}
	}

	return nil
}

// ParseMethod parses the method string to Method struct.
// The method string format is:
//
// <name>:<method>:<path>:<input{<field1>,<field2>,...<fieldn>}>:<output{<field1>,<field2>,...<fieldn>}>
//
//	name   - method name, is pure string
//	method - method is http method type, Is one of GET, POST, PUT, DELETE, PATCH
//	path   - path is http path, is pure string
//	input  - input argument's name, and is a field of array
//	output - output argument's name, is a field of array
//	field - field is a struct, contains name and type, format is <name>:<type>
//	type - type is a proto scalar type, is one of string, int32, int64, uint32, uint64, bool, float32, float64, bytes
func ParseMethod(s string) (*Method, error) {
	// implements the doc format check
	var preSections = strings.SplitN(s, ":", 4)
	// result
	// preSections[0] = "GetUser"
	// preSections[1] = "GET"
	// preSections[2] = "UserRequest{id:int32,name:string}:UserResponse{id:int32,name:string}"

	if len(preSections) != 4 {
		return nil, errors.New("invalid method format")
	}

	// check method name
	if preSections[0] == "" {
		return nil, errors.New("invalid method name")
	}

	if !validIdent(preSections[0]) {
		return nil, errors.New("method name must be ident format")
	}

	if preSections[1] != "" {
		// check method type
		if !strings.Contains("GET,POST,PUT,DELETE,PATCH", preSections[1]) {
			return nil, errors.New("invalid method type")
		}
	}

	// check input and output
	inouts := InputRegexp.FindAllStringSubmatch(preSections[3], 2)
	if len(inouts) != 2 {
		return nil, errors.New("invalid input or output")
	}

	// check input
	input, err := parseArg(inouts[0])
	if err != nil {
		return nil, err
	}

	// check output
	output, err := parseArg(inouts[1])
	if err != nil {
		return nil, err
	}

	return &Method{
		Name:       preSections[0],
		HttpMethod: preSections[1],
		Path:       preSections[2],
		Input:      *input,
		Output:     *output,
	}, nil
}

// parseArg parses the input or output string to Arg struct.
func parseArg(ss []string) (*ArgType, error) {
	// check input
	if ss[1] == "" {
		return nil, errors.New("invalid input name")
	}

	if !validIdent(ss[1]) {
		return nil, errors.New("input name must be ident format")
	}

	// inputs has many fields
	fields := FieldRegexp.FindAllStringSubmatch(ss[2], -1)
	if len(fields) == 0 {
		return nil, errors.New("invalid input fields")
	}

	var fs []*Field
	for _, field := range fields {
		if field[1] == "" {
			return nil, errors.New("invalid input field name")
		}

		if !validIdent(field[1]) {
			return nil, errors.New("input field name must be ident format")
		}

		if field[2] == "" {
			return nil, errors.New("invalid input field type")
		}

		if !strings.Contains("string,int32,int64,uint32,uint64,bool,float32,float64,bytes", field[2]) {
			return nil, errors.New("invalid input field type")
		}

		fs = append(fs, &Field{
			Name: field[1],
			Type: field[2],
		})
	}

	return &ArgType{
		Name:   ss[1],
		Fields: fs,
	}, nil
}
