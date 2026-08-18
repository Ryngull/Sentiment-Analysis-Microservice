import grpc
from concurrent import futures
import logging
import time
import sys
import os
from grpc_health.v1 import health, health_pb2, health_pb2_grpc
from transformers import pipeline

# Make sibling protobuf modules importable when this file is loaded from elsewhere.
sys.path.append(os.path.dirname(os.path.abspath(__file__)))
import analysis_pb2
import analysis_pb2_grpc

logging.basicConfig(
    level=os.getenv("LOG_LEVEL", "INFO"),
    format="%(asctime)s %(levelname)s %(name)s %(message)s",
)
logger = logging.getLogger(__name__)


class TextAnalysisService(analysis_pb2_grpc.TextAnalysisServiceServicer):
    """Implements the protobuf sentiment-analysis service."""

    def __init__(self):
        # Load the model once at startup because initialization is expensive.
        model_name = os.getenv(
            "MODEL_NAME",
            "distilbert/distilbert-base-uncased-finetuned-sst-2-english",
        )
        logger.info("Initializing and loading AI model: %s", model_name)

        self.classifier = pipeline("sentiment-analysis", model=model_name)

        logger.info("Model loaded; waiting for requests from the Go gateway")

    def AnalyzeText(self, request, context):
        texts = list(request.texts)
        if not texts:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "texts must not be empty")

        logger.info("Received %d text(s) for analysis", len(texts))

        start_time = time.time()

        try:
            model_outputs = self.classifier(texts)
            results_to_return = []
            for out in model_outputs:
                results_to_return.append(
                    analysis_pb2.AnalysisResult(
                        sentiment=out["label"],
                        confidence_score=float(out["score"]),
                    )
                )
        except Exception as e:
            logger.exception("Inference failed")
            context.abort(grpc.StatusCode.INTERNAL, f"Model inference failed: {e}")

        processing_time = int((time.time() - start_time) * 1000)

        return analysis_pb2.AnalyzeResponse(
            results=results_to_return,
            processing_time_ms=processing_time,
        )


def serve():
    max_workers = int(os.getenv("GRPC_MAX_WORKERS", "10"))
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=max_workers))

    analysis_pb2_grpc.add_TextAnalysisServiceServicer_to_server(
        TextAnalysisService(), server
    )

    health_servicer = health.HealthServicer()
    health_pb2_grpc.add_HealthServicer_to_server(health_servicer, server)
    health_servicer.set("", health_pb2.HealthCheckResponse.SERVING)

    bind_address = os.getenv("GRPC_BIND_ADDRESS", "[::]")
    grpc_port = os.getenv("GRPC_PORT", "50051")
    listen_address = f"{bind_address}:{grpc_port}"
    if server.add_insecure_port(listen_address) == 0:
        raise RuntimeError(f"Failed to listen on gRPC address {listen_address}")
    server.start()
    logger.info(
        "Python worker listening on %s with %d threads",
        listen_address,
        max_workers,
    )

    server.wait_for_termination()


if __name__ == "__main__":
    serve()
