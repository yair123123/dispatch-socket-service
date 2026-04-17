package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"dispatch-socket-service/internal/models"
)

type CoreClient interface {
	AssignDriver(ctx context.Context, req models.CoreAssignDriverRequest) error
	ReportDispatchRoundResult(ctx context.Context, req models.DispatchRoundResultRequest) error
}

type HTTPClient struct {
	baseURL        string
	internalSecret string
	client         *http.Client
}

func NewCoreClient(baseURL, internalSecret string, timeout time.Duration) *HTTPClient {
	return &HTTPClient{baseURL: baseURL, internalSecret: internalSecret, client: &http.Client{Timeout: timeout}}
}

func (c *HTTPClient) AssignDriver(ctx context.Context, req models.CoreAssignDriverRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	hreq, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/internal/rides/%s/assign-driver", c.baseURL, req.RideID), bytes.NewReader(body))
	if err != nil {
		return err
	}
	hreq.Header.Set("Content-Type", "application/json")
	if c.internalSecret != "" {
		hreq.Header.Set("X-Internal-Secret", c.internalSecret)
	}
	resp, err := c.client.Do(hreq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("core assign failed status=%d", resp.StatusCode)
	}
	return nil
}

func (c *HTTPClient) ReportDispatchRoundResult(ctx context.Context, req models.DispatchRoundResultRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	hreq, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/internal/dispatch/round-result", c.baseURL), bytes.NewReader(body))
	if err != nil {
		return err
	}
	hreq.Header.Set("Content-Type", "application/json")
	if c.internalSecret != "" {
		hreq.Header.Set("X-Internal-Secret", c.internalSecret)
	}
	resp, err := c.client.Do(hreq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("core round result failed status=%d", resp.StatusCode)
	}
	return nil
}
