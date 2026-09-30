"""Selects arms from a set of policy replicas and rewards them.

Environment:
  BANDIT_POLICY_TARGET    where the policy replicas are (default localhost:9090)
  BANDIT_FEEDBACK_TARGET  where the feedback service is (default: same as policy)
"""

import collections
import os
import random

from bandit_client import AlreadyRewarded, Client, UnknownRequest

# The world the users live in: the probability that each arm pays off.
PROBS = [0.2, 0.5, 0.8]


def main():
    client = Client(
        os.environ.get("BANDIT_POLICY_TARGET", "localhost:9090"),
        os.environ.get("BANDIT_FEEDBACK_TARGET"),
    )
    rng = random.Random(1)

    answered_by = collections.Counter()
    arms = collections.Counter()
    for _ in range(60):
        sel = client.select()
        answered_by[sel.instance] += 1
        arms[sel.arm] += 1
        client.reward(sel.request_id, 1.0 if rng.random() < PROBS[sel.arm] else 0.0)

    print("answered by:", ", ".join(f"{name}={n}" for name, n in sorted(answered_by.items())))
    print("arm choices:", [arms[a] for a in range(len(PROBS))])

    # Errors arrive as exceptions, not status codes.
    sel = client.select()
    client.reward(sel.request_id, 1.0)
    try:
        client.reward(sel.request_id, 1.0)
    except AlreadyRewarded as err:
        print("second reward:", type(err).__name__)
    try:
        client.reward("no-such-id", 1.0)
    except UnknownRequest as err:
        print("unknown id:", type(err).__name__)

    stats = client.stats()
    print(f"one replica's view ({stats.instance}):",
          [f"arm {a.arm}: {a.pulls} pulls, mean {a.mean:.2f}, {a.pending} waiting" for a in stats.arms])


if __name__ == "__main__":
    main()
