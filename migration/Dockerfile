FROM python:3.11-slim

RUN apt-get update && apt-get install -y \
    curl \
    && rm -rf /var/lib/apt/lists/*

RUN pip install milvus-cli

WORKDIR /app

COPY init.sh /app/init.sh
RUN chmod +x /app/init.sh

ENTRYPOINT ["/app/init.sh"]
