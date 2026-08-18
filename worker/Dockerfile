# syntax=docker/dockerfile:1

FROM python:3.11-slim

ARG TORCH_VERSION=2.5.1

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    HF_HOME=/home/app/.cache/huggingface

WORKDIR /app

RUN groupadd --system app \
    && useradd --system --gid app --create-home app

COPY requirements.txt ./
RUN python -m pip install --no-cache-dir \
        --index-url https://download.pytorch.org/whl/cpu \
        "torch==${TORCH_VERSION}" \
    && python -m pip install --no-cache-dir -r requirements.txt

COPY analysis_pb2.py analysis_pb2_grpc.py main.py healthcheck.py ./

RUN mkdir -p "${HF_HOME}" \
    && chown -R app:app /app /home/app

USER app
EXPOSE 50051

HEALTHCHECK --interval=15s --timeout=5s --start-period=120s --retries=5 \
    CMD ["python", "healthcheck.py"]

ENTRYPOINT ["python", "main.py"]
