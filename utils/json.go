package utils

import (
	"encoding/json"
	"errors"
	"io"
)

func DecodeJSON(r io.Reader, target any) error {
	d := json.NewDecoder(r)
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return err
	}
	if err := d.Decode(new(any)); err != io.EOF {
		if err == nil {
			return errors.New("body must contain single JSON value")
		}
		return err
	}
	return nil
}
