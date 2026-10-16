package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// HTTPClient centraliza el consumo HTTP de servicios externos.
type HTTPClient struct {
	Authorization string
	Client        *http.Client
}

func (c HTTPClient) httpClient() *http.Client {
	if c.Client != nil {
		return c.Client
	}

	return &http.Client{Timeout: 30 * time.Second}
}

func (c HTTPClient) do(
	ctx context.Context,
	method string,
	url string,
	body any,
	target any,
) (int, http.Header, error) {
	var requestBody io.Reader

	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return 0, nil, fmt.Errorf("no se pudo serializar el cuerpo JSON: %w", err)
		}
		requestBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, requestBody)
	if err != nil {
		return 0, nil, fmt.Errorf("no se pudo crear la petición HTTP: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if authorization := strings.TrimSpace(c.Authorization); authorization != "" {
		req.Header.Set("Authorization", authorization)
	}

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("falló la petición al servicio externo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return resp.StatusCode, resp.Header,
			fmt.Errorf("el servicio externo respondió HTTP %d", resp.StatusCode)
	}

	if target != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
			return resp.StatusCode, resp.Header,
				fmt.Errorf("no se pudo interpretar la respuesta JSON: %w", err)
		}
	} else {
		_, _ = io.Copy(io.Discard, resp.Body)
	}

	return resp.StatusCode, resp.Header, nil
}

// Get realiza una petición GET y decodifica la respuesta JSON.
func (c HTTPClient) Get(
	ctx context.Context,
	url string,
	target any,
) (int, error) {
	status, _, err := c.do(ctx, http.MethodGet, url, nil, target)
	return status, err
}

// GetWithTotalCount realiza GET y lee el encabezado Total-Count.
func (c HTTPClient) GetWithTotalCount(
	ctx context.Context,
	url string,
	target any,
) (int, int, error) {
	status, headers, err := c.do(ctx, http.MethodGet, url, nil, target)
	if err != nil {
		return status, 0, err
	}

	total, _ := strconv.Atoi(headers.Get("Total-Count"))
	return status, total, nil
}

// Post envía un cuerpo JSON mediante POST.
func (c HTTPClient) Post(
	ctx context.Context,
	url string,
	body any,
	target any,
) (int, error) {
	status, _, err := c.do(ctx, http.MethodPost, url, body, target)
	return status, err
}

// Put envía un cuerpo JSON mediante PUT.
func (c HTTPClient) Put(
	ctx context.Context,
	url string,
	body any,
	target any,
) (int, error) {
	status, _, err := c.do(ctx, http.MethodPut, url, body, target)
	return status, err
}

// Patch envía un cuerpo JSON mediante PATCH.
func (c HTTPClient) Patch(
	ctx context.Context,
	url string,
	body any,
	target any,
) (int, error) {
	status, _, err := c.do(ctx, http.MethodPatch, url, body, target)
	return status, err
}

// Delete realiza una petición DELETE.
func (c HTTPClient) Delete(
	ctx context.Context,
	url string,
	target any,
) (int, error) {
	status, _, err := c.do(ctx, http.MethodDelete, url, nil, target)
	return status, err
}
