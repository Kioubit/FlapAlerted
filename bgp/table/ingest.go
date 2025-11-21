package table

import (
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

		asPathList, err := u.GetAsPath()
		if err != nil {
			cancel(fmt.Errorf("error getting ASPath: %w", err))
			return
		}

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

		if foundNlRi {
			for _, item := range nlRi.NLRI {
				table.update(item.ToNetCidr(), item.PathID, false, asPathList)
			}
		}
		for _, item := range u.Msg.NetworkLayerReachabilityInformation {
			table.update(item.ToNetCidr(), item.PathID, false, asPathList)
		}

		if foundUnreachNlRi {
			for _, item := range unReachNlRi.Withdrawn {
				table.update(item.ToNetCidr(), item.PathID, true, asPathList)
			}
		}
		for _, item := range u.Msg.WithdrawnRoutesList {
			table.update(item.ToNetCidr(), item.PathID, true, asPathList)
		}

	}
}
