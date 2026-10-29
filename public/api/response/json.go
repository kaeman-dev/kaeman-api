package response

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/labstack/echo/v4"
)

type JSONSerializer struct {
	echo.DefaultJSONSerializer
}

func (s JSONSerializer) Deserialize(c echo.Context, target any) error {
	return s.Decode(c.Request().Body, target)
}

func (JSONSerializer) Decode(r io.Reader, target any) error {
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
