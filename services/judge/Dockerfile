FROM python:3.13-slim

ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1

WORKDIR /app
COPY pyproject.toml uv.lock ./
RUN pip install --no-cache-dir uv && uv sync --frozen
COPY . .

EXPOSE 8002
CMD ["uv", "run", "uvicorn", "app:app", "--host", "0.0.0.0", "--port", "8002"]
