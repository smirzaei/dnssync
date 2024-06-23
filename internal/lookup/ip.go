package lookup

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"
)

const IPLookupAddress = "https://api.ipify.org"

type ErrInvalidIP struct {
	IP string
}

func (e ErrInvalidIP) Error() string {
	return fmt.Sprintf("invalid ip: %s", e.IP)
}

type ErrStatusNotOK struct {
	statusCode int
}

func (e ErrStatusNotOK) Error() string {
	return fmt.Sprintf("received invalid status code: %d", e.statusCode)
}

type IPLookup struct {
	l          *zap.Logger
	httpClient *http.Client
}

func NewIPLookup(logger *zap.Logger) *IPLookup {
	httpClient := http.Client{
		Timeout: 10 * time.Second,
	}

	l := IPLookup{
		l:          logger,
		httpClient: &httpClient,
	}

	return &l
}

func (l *IPLookup) LookupPublicIP(ctx context.Context) (net.IP, error) {
	// ipify is created for this exact purpose
	// https://www.ipify.org/
	// Another solution is to hit this endpoint
	// https://cloudflare.com/cdn-cgi/trace

	l.l.Debug("initiate public ip lookup")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, IPLookupAddress, nil)
	if err != nil {
		return nil, err
	}

	l.l.Debug("sending http request", zap.String("address", IPLookupAddress))
	resp, err := l.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	l.l.Debug("received response", zap.Int("status_code", resp.StatusCode))
	if resp.StatusCode != http.StatusOK {
		err := ErrStatusNotOK{
			statusCode: resp.StatusCode,
		}

		return nil, err
	}

	buf := [64]byte{}
	n, err := resp.Body.Read(buf[:])
	l.l.Debug("read response body", zap.Int("n_bytes", n), zap.String("body", string(buf[0:n])))
	if err != nil {
		return nil, err
	}

	ip := net.ParseIP(string(buf[0:n]))
	if ip == nil {
		err := ErrInvalidIP{
			IP: string(buf[:]),
		}

		return nil, err
	}

	return ip, nil
}
