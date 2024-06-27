package dns

import (
	"context"
	"net"

	"github.com/cloudflare/cloudflare-go"
	"go.uber.org/zap"
)

type DNSManager struct {
	l             *zap.Logger
	api           *cloudflare.API
	zoneContainer *cloudflare.ResourceContainer
}

func NewDNSManager(logger *zap.Logger, apiKey, zoneID string) (*DNSManager, error) {
	api, err := cloudflare.NewWithAPIToken(apiKey)
	if err != nil {
		return nil, err
	}

	zoneContainer := cloudflare.ZoneIdentifier(zoneID)
	manager := DNSManager{
		l:             logger,
		api:           api,
		zoneContainer: zoneContainer,
	}

	return &manager, nil
}

func (d *DNSManager) ListRecords(ctx context.Context) ([]DNSRecord, error) {
	d.l.Debug("list dns records")
	params := cloudflare.ListDNSRecordsParams{}
	records, _, err := d.api.ListDNSRecords(ctx, d.zoneContainer, params)
	if err != nil {
		return nil, err
	}

	d.l.Debug("received records", zap.Any("records", records))

	out := make([]DNSRecord, 0, len(records))
	for _, r := range records {
		record := DNSRecord{
			ID:    r.ID,
			Name:  r.Name,
			Value: r.Content,
		}

		out = append(out, record)
	}

	return out, nil
}

func (d *DNSManager) UpdateIP(ctx context.Context, record string, ip net.IP) error {
	d.l.Debug("update ip", zap.String("record", record), zap.String("ip", ip.String()))
	params := cloudflare.UpdateDNSRecordParams{
		Name:    record,
		Content: ip.String(),
	}

	outChan := make(chan error)
	go func() {
		_, err := d.api.UpdateDNSRecord(ctx, d.zoneContainer, params)
		outChan <- err
	}()

	select {
	case <-ctx.Done():
		return context.Canceled
	case err := <-outChan:
		return err
	}
}
