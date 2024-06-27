package cli

import "github.com/alexflint/go-arg"

type Command string

const (
	CommandList Command = "list"
	CommandRun  Command = "run"
)

type Args struct {
	Command          string `arg:"positional,required" help:"'list' for listing all the available DNS records. 'run' for running the updater."`
	Interval         uint   `arg:"required" help:"How often sync the IP (in seconds)"`
	ZoneID           string `arg:"-z,--zone-id,required" help:"The zone ID. Ref: https://developers.cloudflare.com/fundamentals/setup/find-account-and-zone-ids/"`
	DNSRecordID      string `arg:"-d,--dns-record-id,required" help:"ID of the DNS record that you want to update. Ref: https://developers.cloudflare.com/api/operations/dns-records-for-a-zone-list-dns-records"`
	CloudflareApiKey string `arg:"-k,--key,required" help:"Cloudflare API key. Ref: https://dash.cloudflare.com/profile/api-tokens"`
	Verbose          bool   `arg:"-v,--verbose"`
}

func ParseArgs() Args {
	var args Args
	arg.MustParse(&args)

	return args
}
