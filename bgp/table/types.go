package table

import (
	"FlapAlerted/bgp/common"
	"FlapAlerted/bgp/update"
	"encoding/json"
	"log/slog"
)

type SessionUpdateMessage struct {
	Msg     update.Msg
	Session *common.LocalSession
}

func (u *SessionUpdateMessage) LogValue() slog.Value {
	k, err := json.Marshal(u)
	if err != nil {
		return slog.StringValue("Failed to marshal update to JSON: " + err.Error())
	}

	asPath, err := u.GetAsPath()
	if err != nil {
		slog.Warn("error getting ASPath", "error", err)
	}

	nlRi, foundNlRi, err := u.GetMpReachNLRI()
	if err != nil {
		slog.Warn("error getting MpReachNLRI", "error", err)
	}
	prefixList := make([]string, 0)
	if foundNlRi {
		for i := range nlRi.NLRI {
			prefixList = append(prefixList, nlRi.NLRI[i].ToNetCidr().String())
		}
	}

	for _, information := range u.Msg.NetworkLayerReachabilityInformation {
		prefixList = append(prefixList, information.ToNetCidr().String())
	}

	return slog.GroupValue(
		slog.Attr{
			Key:   "full",
			Value: slog.StringValue(string(k)),
		},
		slog.Attr{
			Key:   "asPath",
			Value: slog.AnyValue(asPath),
		},
		slog.Attr{
			Key:   "prefixes",
			Value: slog.AnyValue(prefixList),
		},
	)
}

func (u *SessionUpdateMessage) GetMpReachNLRI() (update.MPReachNLRI, bool, error) {
	return u.Msg.GetMpReachNLRI(u.Session)
}

func (u *SessionUpdateMessage) GetMpUnReachNLRI() (update.MPUnReachNLRI, bool, error) {
	return u.Msg.GetMpUnReachNLRI(u.Session)
}

func (u *SessionUpdateMessage) GetAsPath() (common.AsPath, error) {
	return u.Msg.GetAsPath(u.Session)
}
