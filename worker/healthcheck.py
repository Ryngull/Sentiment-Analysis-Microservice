import os

import grpc
from grpc_health.v1 import health_pb2, health_pb2_grpc


def main() -> None:
    port = os.getenv("GRPC_PORT", "50051")
    target = os.getenv("GRPC_HEALTH_TARGET", f"127.0.0.1:{port}")
    with grpc.insecure_channel(target) as channel:
        stub = health_pb2_grpc.HealthStub(channel)
        response = stub.Check(
            health_pb2.HealthCheckRequest(service=""), timeout=3
        )
        if response.status != health_pb2.HealthCheckResponse.SERVING:
            raise RuntimeError(f"gRPC health status is {response.status}")


if __name__ == "__main__":
    main()
