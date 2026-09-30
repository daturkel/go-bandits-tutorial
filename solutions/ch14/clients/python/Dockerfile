# Built with the repository (module) root as the context, because the stubs
# are generated from proto/, which lives outside clients/python:
#   docker build -f clients/python/Dockerfile .
FROM python:3.12-slim
WORKDIR /app
COPY clients/python/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY clients/python/ .
COPY proto /proto
RUN sh generate.sh /proto
CMD ["python", "demo.py"]
