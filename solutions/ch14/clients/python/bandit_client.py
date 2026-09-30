"""A small Python client for the bandit services.

Wraps the generated gRPC stubs so that callers deal in plain values and
exceptions instead of protobuf messages and status codes.
"""

import json
import sys
import time
from dataclasses import dataclass
from pathlib import Path

import grpc

# The stubs are generated into gen/ by generate.sh.
sys.path.insert(0, str(Path(__file__).parent / "gen"))
from bandit.v1 import bandit_pb2, bandit_pb2_grpc  # noqa: E402


class BanditError(Exception):
    """Base class for errors from the service."""


class UnknownRequest(BanditError):
    """The request id is unknown or has expired."""


class AlreadyRewarded(BanditError):
    """A reward for this request id was already recorded."""


class Unavailable(BanditError):
    """The service could not be reached, even after retries."""


@dataclass(frozen=True)
class Selection:
    request_id: str
    arm: int
    instance: str


# Spread calls over every address the target resolves to. Without this, gRPC
# uses the first address and keeps using it for the life of the connection.
ROUND_ROBIN = json.dumps({"loadBalancingConfig": [{"round_robin": {}}]})


def channel(target: str) -> grpc.Channel:
    """Open a channel to target (host:port, or dns:///name:port for a service
    with several replicas), balancing across all the addresses it resolves to."""
    return grpc.insecure_channel(target, options=[("grpc.service_config", ROUND_ROBIN)])


class Client:
    def __init__(self, policy_target: str, feedback_target: str | None = None, *,
                 timeout: float = 2.0, retries: int = 3, backoff: float = 0.1):
        self._policy = bandit_pb2_grpc.PolicyServiceStub(channel(policy_target))
        self._feedback = bandit_pb2_grpc.FeedbackServiceStub(channel(feedback_target or policy_target))
        self._timeout = timeout
        self._retries = retries
        self._backoff = backoff

    def select(self) -> Selection:
        resp = self._call(self._policy.Select, bandit_pb2.SelectRequest())
        return Selection(resp.request_id, resp.arm, resp.instance)

    def reward(self, request_id: str, reward: float) -> int:
        """Record a reward in [0, 1]. Returns the arm it was credited to."""
        req = bandit_pb2.RewardRequest(request_id=request_id, reward=reward)
        try:
            return self._call(self._feedback.Reward, req).arm
        except grpc.RpcError as err:
            code = err.code()
            if code == grpc.StatusCode.NOT_FOUND:
                raise UnknownRequest(err.details()) from None
            if code == grpc.StatusCode.ALREADY_EXISTS:
                raise AlreadyRewarded(err.details()) from None
            raise

    def stats(self):
        return self._call(self._policy.Stats, bandit_pb2.StatsRequest())

    def _call(self, method, request):
        """Call with a deadline, retrying only when retrying can help.

        UNAVAILABLE means the request may not have been processed (a replica
        was starting, a connection dropped), so trying again is reasonable.
        Anything else, including DEADLINE_EXCEEDED and every client error, is
        raised at once: repeating a call the server rejected only repeats the
        rejection, and repeating one that timed out may do the work twice.
        """
        for attempt in range(self._retries + 1):
            try:
                return method(request, timeout=self._timeout)
            except grpc.RpcError as err:
                if err.code() != grpc.StatusCode.UNAVAILABLE:
                    raise
                if attempt == self._retries:
                    raise Unavailable(err.details()) from None
                time.sleep(self._backoff * 2 ** attempt)
