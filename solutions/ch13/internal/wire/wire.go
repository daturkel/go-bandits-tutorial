// Package wire converts between the protobuf messages and the types the rest
// of the program uses. Keeping it in one place means generated types stay at
// the edge of the system: services and policies never import them.
package wire

import (
	"banditlab/internal/bandit"
	pb "banditlab/internal/gen/bandit/v1"
)

// TotalsToProto converts totals to their wire form.
func TotalsToProto(totals []bandit.ArmTotals) []*pb.ArmTotals {
	out := make([]*pb.ArmTotals, len(totals))
	for i, t := range totals {
		out[i] = &pb.ArmTotals{Arm: int32(t.Arm), Pulls: t.Pulls, RewardSum: t.RewardSum}
	}
	return out
}

// TotalsFromProto converts totals from their wire form. Whether the result is
// valid (right length, sane counts) is for the receiver to check; a message
// can hold anything.
func TotalsFromProto(in []*pb.ArmTotals) []bandit.ArmTotals {
	out := make([]bandit.ArmTotals, len(in))
	for i, t := range in {
		out[i] = bandit.ArmTotals{Arm: int(t.GetArm()), Pulls: t.GetPulls(), RewardSum: t.GetRewardSum()}
	}
	return out
}

// StatsToProto converts a policy snapshot to its wire form.
func StatsToProto(stats []bandit.ArmStat) []*pb.ArmStat {
	out := make([]*pb.ArmStat, len(stats))
	for i, s := range stats {
		out[i] = &pb.ArmStat{Arm: int32(s.Arm), Pulls: int64(s.Pulls), Mean: s.Mean, Pending: int32(s.Pending)}
	}
	return out
}
