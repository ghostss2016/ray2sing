package ray2sing

import (
	"strconv"
	"time"

	T "github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json/badoption"
)

func Hysteria2Singbox(hysteria2Url string) (*T.Outbound, error) {
	u, err := ParseUrl(hysteria2Url, 443)
	if err != nil {
		return nil, err
	}
	decoded := u.Params
	var ObfsOpts *T.Hysteria2Obfs
	ObfsOpts = nil
	if obfs, ok := decoded["obfs"]; ok && obfs != "" {
		// normalizeStr (url_schema.go) replaces '-'/'_' with space; URL
		// param "obfs-password" (or "obfs_password") is stored as "obfs password".
		obfsPass := decoded["obfs password"]
		if obfsPass == "" {
			obfsPass = decoded["obfs-password"]
		}
		ObfsOpts = &T.Hysteria2Obfs{
			Type:     obfs,
			Password: obfsPass,
		}
	}

	valECH, hasECH := decoded["ech"]
	hasECH = hasECH && (valECH != "0")
	var ECHOpts *T.OutboundECHOptions
	ECHOpts = nil
	if hasECH {
		ECHOpts = &T.OutboundECHOptions{
			Enabled: hasECH,
		}
	}

	SNI := decoded["sni"]
	if SNI == "" {
		SNI = decoded["hostname"]
	}
	// turnRelay, err := u.GetRelayOptions()
	// if err != nil {
	// 	return nil, err
	// }
	pass := u.Username
	if u.Password != "" {
		pass += ":" + u.Password
	}

	opts := &T.Hysteria2OutboundOptions{
		ServerOptions: u.GetServerOption(),
		Obfs:          ObfsOpts,
		Password:      pass,
		OutboundTLSOptionsContainer: T.OutboundTLSOptionsContainer{
			TLS: &T.OutboundTLSOptions{
				Enabled:    true,
				Insecure:   decoded["insecure"] == "1",
				DisableSNI: isIPOnly(SNI),
				ServerName: SNI,
				ECH:        ECHOpts,
			},
		},
		// TurnRelay: turnRelay,
	}

	// Port hopping: if URL host had port range (e.g. :20000-40000), apply it
	if len(u.ServerPorts) > 0 {
		opts.ServerPorts = badoption.Listable[string](u.ServerPorts)
		// Default hop interval 30s if not specified in query
		hopSecs := 30
		hi := decoded["hop interval"]
		if hi == "" {
			hi = decoded["hop-interval"]
		}
		if hi != "" {
			if n, err := strconv.Atoi(hi); err == nil && n > 0 {
				hopSecs = n
			}
		}
		opts.HopInterval = badoption.Duration(time.Duration(hopSecs) * time.Second)
	}

	result := T.Outbound{
		Type:    "hysteria2",
		Tag:     u.Name,
		Options: opts,
	}

	return &result, nil
}
