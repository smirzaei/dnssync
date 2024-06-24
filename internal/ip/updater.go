package ip

import (
	"context"
	"net"

	"github.com/cloudflare/cloudflare-go"
	"go.uber.org/zap"
)

type IPUpdater struct {
	l             *zap.Logger
	api           *cloudflare.API
	zoneContainer *cloudflare.ResourceContainer
}

func NewIPUpdater(logger *zap.Logger, apiKey, zoneID string) (*IPUpdater, error) {
	api, err := cloudflare.NewWithAPIToken(apiKey)
	if err != nil {
		return nil, err
	}

	zoneContainer := cloudflare.ZoneIdentifier(zoneID)
	updater := IPUpdater{
		l:             logger,
		api:           api,
		zoneContainer: zoneContainer,
	}

	return &updater, nil
}

func (u *IPUpdater) UpdateIP(ctx context.Context, record string, ip net.IP) error {
	u.l.Debug("update ip", zap.String("record", record), zap.String("ip", ip.String()))
	params := cloudflare.UpdateDNSRecordParams{
		Name:    record,
		Content: ip.String(),
	}

	outChan := make(chan error)
	go func() {
		_, err := u.api.UpdateDNSRecord(ctx, u.zoneContainer, params)
		outChan <- err
	}()

	select {
	case <-ctx.Done():
		return context.Canceled
	case err := <-outChan:
		return err
	}
}
