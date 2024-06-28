package ip

import (
	"context"
	"errors"
	"net"

	"github.com/cloudflare/cloudflare-go"
	"go.uber.org/zap"
)

type IPUpdater struct {
	l             *zap.Logger
	api           *cloudflare.API
	zoneContainer *cloudflare.ResourceContainer
}

var ErrRecordNotFound = errors.New("requested dns record was not found")

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
	u.l.Debug("get all dns records")
	listParams := cloudflare.ListDNSRecordsParams{}
	records, _, err := u.api.ListDNSRecords(ctx, u.zoneContainer, listParams)
	if err != nil {
		return err
	}
	u.l.Debug("received dns records", zap.Int("n_records", len(records)))
	var recordID string
	for _, r := range records {
		if r.Name == record {
			u.l.Debug("found corresponding", zap.Any("record", r))
			recordID = r.ID
			break
		}
	}

	if recordID == "" {
		return ErrRecordNotFound
	}

	params := cloudflare.UpdateDNSRecordParams{
		ID:      recordID,
		Content: ip.String(),
	}

	u.l.Debug("update record", zap.Any("params", params))
	_, err = u.api.UpdateDNSRecord(ctx, u.zoneContainer, params)
	return err
}
