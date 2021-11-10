// Copyright 2020 The Matrix.org Foundation C.I.C.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package httputil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

type InternalAPIError interface {
	error
	Temporary() bool
	Remote() bool
	isInternalAPIError()
}

// internalAPICallError represents an error reaching an internal API.
type internalAPICallError struct {
	err       error
	temporary bool
}

func (e *internalAPICallError) isInternalAPIError() {}

func (e *internalAPICallError) Error() string {
	return fmt.Sprintf("internal API call failed: %s", e.err.Error())
}

func (e *internalAPICallError) Remote() bool {
	return false
}

func (e *internalAPICallError) Temporary() bool {
	return e.temporary
}

// internalAPIRemoteError represents an error returned from a internal API.
type internalAPIRemoteError struct {
	code int
	url  string
	err  string
}

func (e *internalAPIRemoteError) isInternalAPIError() {}

func (e *internalAPIRemoteError) Error() string {
	return fmt.Sprintf("internal API %s returned HTTP %d: %s", e.url, e.code, e.err)
}

func (e *internalAPIRemoteError) Remote() bool {
	return true
}

func (e *internalAPIRemoteError) Temporary() bool {
	return e.code >= 500
}

// PostJSON performs a POST request with JSON on an internal HTTP API
func PostJSON(
	ctx context.Context, span opentracing.Span, httpClient *http.Client,
	apiURL string, request, response interface{},
) InternalAPIError {
	jsonBytes, err := json.Marshal(request)
	if err != nil {
		return &internalAPICallError{
			err:       err,
			temporary: false,
		}
	}

	parsedAPIURL, err := url.Parse(apiURL)
	if err != nil {
		return &internalAPICallError{
			err:       err,
			temporary: false,
		}
	}

	parsedAPIURL.Path = InternalPathPrefix + strings.TrimLeft(parsedAPIURL.Path, "/")
	apiURL = parsedAPIURL.String()

	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(jsonBytes))
	if err != nil {
		return &internalAPICallError{
			err:       err,
			temporary: false,
		}
	}

	// Mark the span as being an RPC client.
	ext.SpanKindRPCClient.Set(span)
	carrier := opentracing.HTTPHeadersCarrier(req.Header)
	tracer := opentracing.GlobalTracer()

	if err = tracer.Inject(span.Context(), opentracing.HTTPHeaders, carrier); err != nil {
		return &internalAPICallError{
			err:       err,
			temporary: true,
		}
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := httpClient.Do(req.WithContext(ctx))
	if res != nil {
		defer (func() { err = res.Body.Close() })()
	}
	if err != nil {
		return &internalAPICallError{
			err:       err,
			temporary: true,
		}
	}
	if res.StatusCode != http.StatusOK {
		var errorBody struct {
			Message string `json:"message"`
		}
		if msgerr := json.NewDecoder(res.Body).Decode(&errorBody); msgerr == nil {
			return &internalAPIRemoteError{
				err:  errorBody.Message,
				url:  apiURL,
				code: res.StatusCode,
			}
		}
		return &internalAPIRemoteError{
			err:  "unknown error",
			url:  apiURL,
			code: res.StatusCode,
		}
	}
	if err := json.NewDecoder(res.Body).Decode(response); err != nil {
		return &internalAPICallError{
			err:       err,
			temporary: false,
		}
	}
	return nil
}
