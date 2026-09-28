package response_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"aegis/pkg/response"

	"github.com/stretchr/testify/require"
)

func TestEncodeSuccess(t *testing.T) {
	raw, err := response.EncodeSuccess(map[string]string{"status": "ok"})
	require.NoError(t, err)

	var env response.Envelope
	require.NoError(t, json.Unmarshal(raw, &env))
	require.True(t, env.Success)
	require.Nil(t, env.Error)
	require.NotNil(t, env.Data)
}

func TestEncodeError(t *testing.T) {
	raw, err := response.EncodeError("INTERNAL", "something went wrong")
	require.NoError(t, err)

	var env response.Envelope
	require.NoError(t, json.Unmarshal(raw, &env))
	require.False(t, env.Success)
	require.Nil(t, env.Data)
	require.NotNil(t, env.Error)
	require.Equal(t, "INTERNAL", env.Error.Code)
	require.Equal(t, "something went wrong", env.Error.Message)
}

func TestSuccessAndErrorHTTP(t *testing.T) {
	ok := httptest.NewRecorder()
	require.NoError(t, response.Success(ok, http.StatusOK, map[string]bool{"alive": true}))
	require.Equal(t, http.StatusOK, ok.Code)
	require.Contains(t, ok.Body.String(), `"success":true`)

	fail := httptest.NewRecorder()
	require.NoError(t, response.Error(fail, http.StatusBadRequest, "VALIDATION", "bad input"))
	require.Equal(t, http.StatusBadRequest, fail.Code)
	require.Contains(t, fail.Body.String(), `"success":false`)
	require.Contains(t, fail.Body.String(), `"code":"VALIDATION"`)
}
