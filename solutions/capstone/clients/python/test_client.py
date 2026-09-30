"""Tests for bandit_client against fake servers running in this process.

Run with:  python -m unittest -v      (after `sh generate.sh`)
"""

import os
import threading
import unittest
from concurrent import futures

# gRPC sends connections through an HTTP proxy if the environment names one,
# and for some kinds of target that includes loopback addresses. These tests
# only talk to servers in this process, so take the proxy out of the picture
# before grpc is imported.
for var in ("http_proxy", "https_proxy", "HTTP_PROXY", "HTTPS_PROXY"):
    os.environ.pop(var, None)

import grpc

import bandit_client  # first: it puts gen/ on the import path
from bandit.v1 import bandit_pb2, bandit_pb2_grpc


class FakePolicy(bandit_pb2_grpc.PolicyServiceServicer):
    def __init__(self, name):
        self.name = name
        self.calls = 0
        self.fail_first = 0  # answer UNAVAILABLE this many times first
        self.lock = threading.Lock()

    def Select(self, request, context):
        with self.lock:
            self.calls += 1
            if self.fail_first > 0:
                self.fail_first -= 1
                context.abort(grpc.StatusCode.UNAVAILABLE, "starting up")
        return bandit_pb2.SelectResponse(request_id=f"{self.name}-{self.calls}", arm=1, instance=self.name)


class FakeFeedback(bandit_pb2_grpc.FeedbackServiceServicer):
    def __init__(self):
        self.seen = set()

    def Reward(self, request, context):
        if request.request_id == "missing":
            context.abort(grpc.StatusCode.NOT_FOUND, "unknown or expired request id")
        if request.request_id in self.seen:
            context.abort(grpc.StatusCode.ALREADY_EXISTS, "reward already recorded")
        self.seen.add(request.request_id)
        return bandit_pb2.RewardResponse(arm=1)


def serve(*servicers):
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=4))
    for register, servicer in servicers:
        register(servicer, server)
    port = server.add_insecure_port("127.0.0.1:0")
    server.start()
    return server, port


class ClientTest(unittest.TestCase):
    def setUp(self):
        self.policy = FakePolicy("p")
        self.feedback = FakeFeedback()
        self.server, port = serve(
            (bandit_pb2_grpc.add_PolicyServiceServicer_to_server, self.policy),
            (bandit_pb2_grpc.add_FeedbackServiceServicer_to_server, self.feedback),
        )
        self.addCleanup(self.server.stop, None)
        self.target = f"127.0.0.1:{port}"

    def test_select_returns_plain_values(self):
        sel = bandit_client.Client(self.target).select()
        self.assertEqual(sel.arm, 1)
        self.assertEqual(sel.instance, "p")
        self.assertTrue(sel.request_id)

    def test_retries_unavailable_then_succeeds(self):
        self.policy.fail_first = 2
        client = bandit_client.Client(self.target, backoff=0.01)
        self.assertEqual(client.select().arm, 1)
        self.assertEqual(self.policy.calls, 3)

    def test_gives_up_after_the_retries(self):
        self.policy.fail_first = 100
        client = bandit_client.Client(self.target, retries=2, backoff=0.01)
        with self.assertRaises(bandit_client.Unavailable):
            client.select()
        self.assertEqual(self.policy.calls, 3)  # the first try and two retries

    def test_client_errors_are_not_retried(self):
        client = bandit_client.Client(self.target, backoff=0.01)
        with self.assertRaises(bandit_client.UnknownRequest):
            client.reward("missing", 1.0)

    def test_repeated_reward_is_a_distinct_error(self):
        client = bandit_client.Client(self.target)
        client.reward("a", 1.0)
        with self.assertRaises(bandit_client.AlreadyRewarded):
            client.reward("a", 1.0)

    def test_deadline_is_enforced(self):
        release = threading.Event()

        class Slow(FakePolicy):
            def Select(self, request, context):
                release.wait(5)
                return bandit_pb2.SelectResponse()

        server, port = serve((bandit_pb2_grpc.add_PolicyServiceServicer_to_server, Slow("slow")))
        self.addCleanup(server.stop, None)
        self.addCleanup(release.set)
        client = bandit_client.Client(f"127.0.0.1:{port}", timeout=0.2)
        with self.assertRaises(grpc.RpcError) as ctx:
            client.select()
        self.assertEqual(ctx.exception.code(), grpc.StatusCode.DEADLINE_EXCEEDED)


class RoundRobinTest(unittest.TestCase):
    def test_calls_alternate_between_backends(self):
        a, b = FakePolicy("a"), FakePolicy("b")
        sa, pa = serve((bandit_pb2_grpc.add_PolicyServiceServicer_to_server, a))
        sb, pb = serve((bandit_pb2_grpc.add_PolicyServiceServicer_to_server, b))
        self.addCleanup(sa.stop, None)
        self.addCleanup(sb.stop, None)

        client = bandit_client.Client(f"ipv4:127.0.0.1:{pa},127.0.0.1:{pb}")
        for _ in range(40):
            client.select()
        self.assertGreater(a.calls, 12)
        self.assertGreater(b.calls, 12)


if __name__ == "__main__":
    unittest.main()
