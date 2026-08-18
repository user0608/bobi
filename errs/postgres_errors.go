package errs

import (
	"errors"
	"net/http"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/user0608/ifdevmode"
)

type details struct {
	message  string
	httpCode int
	loggable bool
}

type PGCode string

const (
	PgInvalidLengthError       PGCode = "22001"
	PgDuplicateRecordError     PGCode = "23505"
	PgInvalidFormatError       PGCode = "23514"
	PgDependentRecordsError    PGCode = "23503"
	PgDataIntegrityError       PGCode = "23000"
	PgOperationFailedError     PGCode = "25000"
	PgInternalProblemError     PGCode = "26000"
	PgUnauthorizedAccessError  PGCode = "28000"
	PgTransactionError         PGCode = "2D000"
	PgNonexistentResourceError PGCode = "42P01"
	PgInvalidFieldValueError   PGCode = "22P02"
	PgInvalidJSONValueError    PGCode = "22032"
	PgNonNullableFieldsError   PGCode = "23502"
)

var (
	mutex      sync.RWMutex
	devmode    = ifdevmode.Yes()
	pgErrcodes = map[PGCode]details{
		PgInvalidLengthError:       {"Verifique que los campos tengan la longitud correcta de caracteres.", http.StatusBadRequest, false},
		PgDuplicateRecordError:     {"El registro ya existe en la base de datos del sistema.", http.StatusBadRequest, false},
		PgInvalidFormatError:       {"Uno de los campos no tiene el formato correcto. Consulte con el administrador del sistema.", http.StatusBadRequest, false},
		PgDependentRecordsError:    {"Se encontraron otros registros dependientes. No podemos realizar ninguna acción mientras estas relaciones existan.", http.StatusBadRequest, false},
		PgDataIntegrityError:       {"Operación restringida debido a un problema de integridad en los datos. Consulte la documentación.", http.StatusBadRequest, false},
		PgOperationFailedError:     {"No se pudieron completar las operaciones. Por favor, informe la incidencia al equipo técnico.", http.StatusInternalServerError, true},
		PgInternalProblemError:     {"Hubo un problema interno. Por favor, informe la incidencia al equipo técnico.", http.StatusInternalServerError, true},
		PgUnauthorizedAccessError:  {"Acceso restringido. No podemos realizar la operación.", http.StatusUnauthorized, true},
		PgTransactionError:         {"Hubo un problema al realizar la transacción. Por favor, informe la incidencia al equipo técnico.", http.StatusInternalServerError, true},
		PgNonexistentResourceError: {"El registro o recurso al que intenta acceder no existe.", http.StatusBadRequest, false},
		PgInvalidFieldValueError:   {"El formato o representación de uno de los valores de campo no cumple con los requerimientos.", http.StatusBadRequest, false},
		PgInvalidJSONValueError:    {"El valor asignado a uno de los campos de tipo JSON no cumple con los requerimientos.", http.StatusBadRequest, false},
		PgNonNullableFieldsError:   {"Hay campos que no deberían ser nulos. Consulte la documentación o al administrador del sistema.", http.StatusBadRequest, false},
	}
)

func Devmode() {
	mutex.Lock()
	defer mutex.Unlock()
	devmode = true
}

func AddPgErrs(code PGCode, message string, httpCode int, loggable bool) {
	mutex.Lock()
	defer mutex.Unlock()
	pgErrcodes[code] = details{message, httpCode, loggable}
}

const (
	ErrInvalidRequestBody          = "La estructura de información enviada es inválida. Por favor, revise la documentación y vuelva a intentar."
	ErrInvalidQueryParam           = "Los parámetros de consulta son inválidos. Favor de revisar la documentación y volver a intentar."
	ErrAuthorizationHeaderNotFound = "La cabecera con el token de utilización no fue encontrada. La operación fue rechazada."
	ErrInvalidToken                = "El token que está utilizando no es válido o ha caducado. Contáctese con el equipo técnico."
	ErrSigningTokenString          = "El token que está utilizando no es genuino. Contáctese con el equipo técnico."
	ErrDatabase                    = "La operación no se pudo realizar debido a algún problema. Contáctese con el equipo técnico."

	ErrRecordNotFound        = "El registro buscado no fue encontrado."
	ErrCreating              = "No se pudo realizar la operación de registro."
	ErrUpdating              = "No se pudo realizar la operación de actualización."
	ErrUserOrPasswordInvalid = "Usuario o contraseña incorrectos."
	ErrIDNotFound            = "Parámetro ID o identificador no encontrado."
	ErrCodeNotFound          = "Parámetro código no encontrado."
	ErrNameNotFound          = "Parámetro nombre no encontrado."
	ErrNotFound              = "No se pudo encontrar ningún recurso asociado a esta consulta."
	ErrGeneric               = "Hubo un error inesperado. Favor de reportar la incidencia al equipo técnico."
	ErrInternal              = ErrGeneric
)

const message23503 = "No se puede realizar la operación debido a asociaciones incompatibles. Asegúrese de que los valores relacionados existan antes de intentar el registro."

func postgresError(err error, pgerr *pgconn.PgError) error {
	code := PGCode(pgerr.Code)

	if code == PgDependentRecordsError &&
		strings.Contains(pgerr.Message, "insert or update") {
		return newError(err, message23503, http.StatusBadRequest)
	}

	state, ok := pgDetail(code)

	if !ok {
		return newError(err, ErrDatabase, http.StatusInternalServerError)
	}

	mutex.RLock()
	developmentMode := devmode
	mutex.RUnlock()
	if state.loggable || developmentMode {
		return newError(err, state.message, state.httpCode)
	}

	return newError(nil, state.message, state.httpCode)
}

func pgDetail(code PGCode) (details, bool) {
	mutex.RLock()
	defer mutex.RUnlock()
	state, ok := pgErrcodes[code]
	return state, ok
}

func pgMessage(code PGCode) string {
	state, _ := pgDetail(code)
	return state.message
}

func IsPgErrCode(err error, code PGCode) bool {
	if pgerr, ok := errors.AsType[*pgconn.PgError](err); ok {
		return PGCode(pgerr.Code) == code
	}
	return false
}
