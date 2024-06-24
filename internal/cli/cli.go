package cli

import "github.com/alexflint/go-arg"

type Args struct {
	Interval         uint   `arg:"required" help:"How often sync the IP (in seconds)"`
	ZoneID           string `arg:"-z,--zone-id,required" help:"The zone ID. Ref: https://developers.cloudflare.com/fundamentals/setup/find-account-and-zone-ids/"`
	DNSRecordName    string `arg:"-d,--dns-record,required" help:"Name of the DNS record to update (e.g. www.example.com)"`
	CloudflareApiKey string `arg:"-k,--key,required" help:"Cloudflare API key. Ref: https://dash.cloudflare.com/profile/api-tokens"`
	Verbose          bool   `arg:"-v,--verbose"`
}

func ParseArgs() Args {
	var args Args
	arg.MustParse(&args)

	return args
}
