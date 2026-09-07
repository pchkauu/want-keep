package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
)

var ErrInvalidRequest = errors.New("invalid request")

type FieldError struct {
	Field string
	Code  string
}

func (e FieldError) Error() string { return "invalid request field" }
func (e FieldError) Unwrap() error { return ErrInvalidRequest }

// Boundary validates transport shape. It is not an authentication or application handler.
type Boundary struct{ schema *openapi3.T }

func NewBoundary() (*Boundary, error) {
	schema, err := generated.GetSwagger()
	if err != nil {
		return nil, err
	}
	if err := schema.Validate(context.Background()); err != nil {
		return nil, err
	}
	return &Boundary{schema: schema}, nil
}

func (b *Boundary) Decode(name string, data []byte, target any) error {
	if b == nil || b.schema == nil || len(data) > 1048576 {
		return ErrInvalidRequest
	}
	schema, ok := b.schema.Components.Schemas[name]
	if !ok || schema.Value == nil {
		return ErrInvalidRequest
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return ErrInvalidRequest
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return ErrInvalidRequest
	}
	if err := schema.Value.VisitJSON(value); err != nil {
		var violation *openapi3.SchemaError
		if errors.As(err, &violation) {
			return FieldError{Field: "/" + strings.Join(violation.JSONPointer(), "/"), Code: violation.SchemaField}
		}
		return ErrInvalidRequest
	}
	if err := json.Unmarshal(data, target); err != nil {
		return ErrInvalidRequest
	}
	return nil
}

func (b *Boundary) validateDTO(name string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return ErrInvalidRequest
	}
	var checked json.RawMessage
	return b.Decode(name, data, &checked)
}
