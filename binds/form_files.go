package binds

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/user0608/bobi/errs"
)

func FormFileBytesOptional(c *echo.Context, fieldName string) ([]byte, error) {
	fileHeader, err := c.FormFile(fieldName)
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return nil, nil
		}
		return nil, errs.InternalErrorDirect("error al obtener el archivo opcional del formulario")
	}

	return formFileBytes(fileHeader)
}

func FormFileBytesRequired(c *echo.Context, fieldName string) ([]byte, error) {
	fileHeader, err := c.FormFile(fieldName)
	if err != nil {
		return nil, errs.BadRequestf("el archivo requerido '%s' no fue enviado o es inválido", fieldName)
	}

	return formFileBytes(fileHeader)
}

func formFileBytes(fileHeader *multipart.FileHeader) ([]byte, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return nil, errs.InternalErrorDirect("error al abrir el archivo del formulario")
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, errs.InternalErrorDirect("error al leer el contenido del archivo")
	}

	return content, nil
}
