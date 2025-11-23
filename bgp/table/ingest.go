package table

import (
	"FlapAlerted/bgp/common"
	"context"
	"fmt"
	"log/slog"
)

func ProcessUpdates(cancel context.CancelCauseFunc, updateChannel chan SessionUpdateMessage, table *PrefixTable) {
	for {
		u, ok := <-updateChannel
		if !ok {
			return
		}

		slog.Debug("Received update", "update", u)

		nlRi, foundNlRi, err := u.GetMpReachNLRI()
		if err != nil {
			cancel(fmt.Errorf("error getting MpReachNLRI: %w", err))
			return
		}
		unReachNlRi, foundUnreachNlRi, err := u.GetMpUnReachNLRI()
		if err != nil {
			cancel(fmt.Errorf("error getting MpUnReachNLRI: %w", err))
			return
		}

		var asPath common.AsPath
		// AS path is not included for withdrawals
		if foundNlRi || len(u.Msg.NetworkLayerReachabilityInformation) != 0 {
			var foundASPath bool
			asPath, foundASPath, err = u.GetAsPath()
			if err != nil {
				cancel(fmt.Errorf("error getting ASPath: %w", err))
				return
			}
			if !foundASPath {
				cancel(fmt.Errorf("missing ASPath attribute"))
				return
			}
		}

		if foundNlRi {
			for _, item := range nlRi.NLRI {
				table.update(item.ToNetCidr(), item.PathID, false, asPath)
			}
		}
		for _, item := range u.Msg.NetworkLayerReachabilityInformation {
			table.update(item.ToNetCidr(), item.PathID, false, asPath)
		}

		if foundUnreachNlRi {
			for _, item := range unReachNlRi.Withdrawn {
				table.update(item.ToNetCidr(), item.PathID, true, nil)
			}
		}
		for _, item := range u.Msg.WithdrawnRoutesList {
			table.update(item.ToNetCidr(), item.PathID, true, nil)
		}

	}
}
